package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type displayStatusRecord struct {
	Phase             string   `json:"phase,omitempty"`
	RequiredRemaining *int     `json:"required_remaining,omitempty"`
	ActiveSessions    []string `json:"active_sessions,omitempty"`
}

func refreshDisplayFacts(rc *RunContext) {
	if rc == nil {
		return
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
		return
	}
	_ = SaveTransportFacts(rc.RunDir, &TransportFacts{
		Version:   1,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func printRunAdvisories(rc *RunContext) {
	advisories := collectRunAdvisories(rc)
	if len(advisories) == 0 {
		return
	}
	fmt.Println("### advisories")
	for _, advisory := range advisories {
		fmt.Printf("- %s\n", advisory)
	}
	fmt.Println()
}

func collectRunAdvisories(rc *RunContext) []string {
	if rc == nil {
		return nil
	}
	status, err := loadDisplayStatusRecord(RunStatusPath(rc.RunDir))
	if err != nil || status == nil {
		return nil
	}
	summaryExists := fileExists(SummaryPath(rc.RunDir))
	completionExists := fileExists(CompletionStatePath(rc.RunDir))
	advisories := make([]string, 0, 2)
	if status.RequiredRemaining != nil && *status.RequiredRemaining == 0 && (!summaryExists || !completionExists) {
		advisories = append(advisories, fmt.Sprintf("Closeout artifacts missing: required_remaining=0 summary_exists=%t completion_proof_exists=%t", summaryExists, completionExists))
	}
	meta, err := LoadRunMetadata(RunMetadataPath(rc.RunDir))
	if err != nil || meta == nil || strings.TrimSpace(meta.Intent) != runIntentEvolve {
		return advisories
	}
	if strings.TrimSpace(status.Phase) != "review" || (summaryExists && completionExists) {
		return advisories
	}
	evolutionEntries, lastTrialAt := evolutionLogFacts(EvolutionLogPath(rc.RunDir))
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
	return advisories
}

func loadDisplayStatusRecord(path string) (*displayStatusRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var record displayStatusRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func evolutionLogFacts(path string) (int, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, ""
	}
	count := len(splitNonEmptyLines(string(data)))
	if count == 0 {
		return 0, ""
	}
	info, err := os.Stat(path)
	if err != nil {
		return count, ""
	}
	return count, info.ModTime().UTC().Format(time.RFC3339)
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
