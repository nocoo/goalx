package cli

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func refreshDisplayFacts(rc *RunContext) error {
	if rc == nil {
		return nil
	}
	_ = reconcileControlDeliveries(rc.RunDir)
	if err := RefreshRunMemoryContext(rc.RunDir); err != nil {
		return err
	}
	if snapshot, err := BuildActivitySnapshot(rc.ProjectRoot, rc.Name, rc.RunDir); err == nil && snapshot != nil {
		_ = SaveActivitySnapshot(rc.RunDir, snapshot)
	}
	masterEngine := ""
	if rc.Config != nil {
		masterEngine = rc.Config.Master.Engine
	}
	if SessionExists(rc.TmuxSession) {
		if facts, err := BuildTransportFacts(rc.RunDir, rc.TmuxSession, masterEngine); err == nil && facts != nil {
			_ = SaveTransportFacts(rc.RunDir, facts)
		}
		return nil
	}
	_ = SaveTransportFacts(rc.RunDir, &TransportFacts{
		Version:   1,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func printRunAdvisories(rc *RunContext) error {
	advisories, err := collectRunAdvisories(rc)
	if err != nil {
		return err
	}
	if len(advisories) == 0 {
		return nil
	}
	fmt.Println("### advisories")
	for _, advisory := range advisories {
		fmt.Printf("- %s\n", advisory)
	}
	fmt.Println()
	return nil
}

func collectRunAdvisories(rc *RunContext) ([]string, error) {
	if rc == nil {
		return nil, nil
	}
	status, err := LoadRunStatusRecord(RunStatusPath(rc.RunDir))
	closeoutFacts, closeoutErr := BuildRunCloseoutFacts(rc.RunDir)
	if closeoutErr != nil {
		closeoutFacts = RunCloseoutFacts{}
	}
	summaryExists := closeoutFacts.SummaryExists
	completionExists := closeoutFacts.CompletionExists
	advisories := make([]string, 0, 2)
	if err == nil && status != nil && status.RequiredRemaining != nil && *status.RequiredRemaining == 0 && (!summaryExists || !completionExists) {
		advisories = append(advisories, fmt.Sprintf("Closeout artifacts missing: required_remaining=0 summary_exists=%t completion_proof_exists=%t", summaryExists, completionExists))
	} else if err != nil {
		return nil, err
	}
	if activity, err := LoadActivitySnapshot(ActivityPath(rc.RunDir)); err == nil && activity != nil {
		if activity.Budget.MaxDurationSeconds > 0 && activity.Budget.Exhausted && activity.Lifecycle.RunActive {
			advisories = append(advisories, "Budget exhausted: "+formatBudgetSummary(activity.Budget))
		}
		coverage := activity.Coverage
		if coverage.OwnersPresent && (len(coverage.UnmappedOpenIDs) > 0 || len(coverage.OwnerSessionMissingIDs) > 0) {
			parts := make([]string, 0, 3)
			if len(coverage.UnmappedOpenIDs) > 0 {
				parts = append(parts, "unmapped_open="+strings.Join(coverage.UnmappedOpenIDs, ","))
			}
			if len(coverage.OwnerSessionMissingIDs) > 0 {
				parts = append(parts, "owner_session_missing="+strings.Join(coverage.OwnerSessionMissingIDs, ","))
			}
			reusable := append([]string{}, coverage.IdleReusableSessions...)
			reusable = append(reusable, coverage.ParkedReusableSessions...)
			if len(reusable) > 0 {
				parts = append(parts, "reusable_sessions="+strings.Join(reusable, ","))
			}
			advisories = append(advisories, "Coverage facts: "+strings.Join(parts, " "))
		}
	}
	meta, err := LoadRunMetadata(RunMetadataPath(rc.RunDir))
	if err != nil || meta == nil || strings.TrimSpace(meta.Intent) != runIntentEvolve {
		return advisories, nil
	}
	if status == nil || strings.TrimSpace(status.Phase) != "review" || (summaryExists && completionExists) {
		return advisories, nil
	}
	evolutionEntries, lastTrialAt, err := evolutionLogFacts(EvolutionLogPath(rc.RunDir))
	if err != nil {
		return nil, err
	}
	parts := []string{
		"phase=review",
		fmt.Sprintf("active_sessions=%d", len(status.ActiveSessions)),
		fmt.Sprintf("evolution_entries=%d", evolutionEntries),
		fmt.Sprintf("summary_exists=%t", summaryExists),
		fmt.Sprintf("completion_proof_exists=%t", completionExists),
	}
	if lastTrialAt != "" {
		parts = append(parts, "last_trial_record_at="+lastTrialAt)
	}
	advisories = append(advisories, "Potential evolve stall: "+strings.Join(parts, " "))
	return advisories, nil
}

func formatCoverageSummary(coverage RequiredCoverage) string {
	if len(coverage.OpenRequiredIDs) == 0 && len(coverage.OwnedOpenIDs) == 0 && len(coverage.UnmappedOpenIDs) == 0 && len(coverage.OwnerSessionMissingIDs) == 0 {
		return ""
	}
	parts := make([]string, 0, 7)
	if !coverage.OwnersPresent {
		parts = append(parts, "coverage=unknown")
	} else {
		parts = append(parts, "coverage=explicit")
	}
	appendCoverageSummaryPart(&parts, "open_required", coverage.OpenRequiredIDs)
	appendCoverageSummaryPart(&parts, "owned_open", coverage.OwnedOpenIDs)
	appendCoverageSummaryPart(&parts, "unmapped_open", coverage.UnmappedOpenIDs)
	appendCoverageSummaryPart(&parts, "owner_session_missing", coverage.OwnerSessionMissingIDs)
	appendCoverageSummaryPart(&parts, "idle_reusable", coverage.IdleReusableSessions)
	appendCoverageSummaryPart(&parts, "parked_reusable", coverage.ParkedReusableSessions)
	return strings.Join(parts, " ")
}

func formatBudgetSummary(budget ActivityBudget) string {
	if budget.MaxDurationSeconds <= 0 {
		return ""
	}
	parts := []string{
		"max_duration=" + formatBudgetDurationSeconds(budget.MaxDurationSeconds),
	}
	if budget.StartedAt != "" {
		parts = append(parts, "started_at="+budget.StartedAt)
	}
	if budget.DeadlineAt != "" {
		parts = append(parts, "deadline_at="+budget.DeadlineAt)
	}
	if budget.ElapsedSeconds > 0 {
		parts = append(parts, "elapsed="+formatBudgetDurationSeconds(budget.ElapsedSeconds))
	}
	switch {
	case budget.RemainingSeconds > 0:
		parts = append(parts, "remaining="+formatBudgetDurationSeconds(budget.RemainingSeconds))
	case budget.RemainingSeconds < 0:
		parts = append(parts, "overrun="+formatBudgetDurationSeconds(-budget.RemainingSeconds))
	}
	parts = append(parts, fmt.Sprintf("exhausted=%t", budget.Exhausted))
	return strings.Join(parts, " ")
}

func formatBudgetDurationSeconds(seconds int64) string {
	return (time.Duration(seconds) * time.Second).String()
}

func formatMemorySummary(runDir string) string {
	queryPresent := fileExists(MemoryQueryPath(runDir))
	contextPresent := fileExists(MemoryContextPath(runDir))
	if !queryPresent && !contextPresent {
		return ""
	}
	parts := []string{
		fmt.Sprintf("query_present=%t", queryPresent),
		fmt.Sprintf("context_present=%t", contextPresent),
	}
	if context, err := LoadMemoryContextFile(MemoryContextPath(runDir)); err == nil && context != nil && strings.TrimSpace(context.BuiltAt) != "" {
		parts = append(parts, "built_at="+context.BuiltAt)
	}
	return strings.Join(parts, " ")
}

func appendCoverageSummaryPart(parts *[]string, label string, values []string) {
	if len(values) == 0 {
		return
	}
	*parts = append(*parts, label+"="+strings.Join(values, ","))
}

func evolutionLogFacts(path string) (int, string, error) {
	events, err := LoadDurableLog(path, DurableSurfaceEvolution)
	if err != nil {
		return 0, "", err
	}
	if len(events) == 0 {
		return 0, "", nil
	}
	return len(events), events[len(events)-1].At, nil
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
