package cli

import (
	"slices"
	"strings"
	"time"
)

type stoppedSemanticRepairOptions struct {
	Origin          string
	At              string
	EvolveReason    string
	EvolveReasonCode string
}

func repairStoppedSemanticSurfaces(runDir string, opts stoppedSemanticRepairOptions) error {
	if strings.TrimSpace(runLifecycleLabel(runDir)) != "stopped" {
		return nil
	}
	now := strings.TrimSpace(opts.At)
	if now == "" {
		now = time.Now().UTC().Format(time.RFC3339)
	}
	stoppedAt := now
	if runtimeState, err := LoadRunRuntimeState(RunRuntimeStatePath(runDir)); err == nil && runtimeState != nil && strings.TrimSpace(runtimeState.StoppedAt) != "" {
		stoppedAt = strings.TrimSpace(runtimeState.StoppedAt)
	}
	if err := repairStoppedRunStatusRecord(runDir, now); err != nil {
		return err
	}
	if err := repairStoppedCoordinationState(runDir, now); err != nil {
		return err
	}
	if err := repairStoppedEvolveFrontier(runDir, stoppedAt, opts); err != nil {
		return err
	}
	return nil
}

func repairStoppedRunStatusRecord(runDir, updatedAt string) error {
	record, err := LoadRunStatusRecord(RunStatusPath(runDir))
	if err != nil || record == nil {
		return err
	}

	requiredRemaining := 0
	openRequiredIDs := append([]string(nil), record.OpenRequiredIDs...)
	if goalState, goalErr := LoadCanonicalGoalState(runDir); goalErr != nil {
		return goalErr
	} else if goalState != nil {
		summary := SummarizeGoalState(goalState)
		requiredRemaining = summary.RequiredRemaining
		openRequiredIDs = goalRemainingRequiredIDs(goalState)
	} else if record.RequiredRemaining != nil {
		requiredRemaining = *record.RequiredRemaining
	}

	changed := false
	if record.Phase != runStatusPhaseStopped {
		record.Phase = runStatusPhaseStopped
		changed = true
	}
	if record.RequiredRemaining == nil || *record.RequiredRemaining != requiredRemaining {
		record.RequiredRemaining = intPtr(requiredRemaining)
		changed = true
	}
	if !slices.Equal(record.OpenRequiredIDs, openRequiredIDs) {
		record.OpenRequiredIDs = append([]string(nil), openRequiredIDs...)
		changed = true
	}
	if len(record.ActiveSessions) > 0 || record.ActiveSessions == nil {
		record.ActiveSessions = []string{}
		changed = true
	}
	if !changed {
		return nil
	}
	record.UpdatedAt = updatedAt
	return SaveRunStatusRecord(RunStatusPath(runDir), record)
}

func repairStoppedEvolveFrontier(runDir, stoppedAt string, opts stoppedSemanticRepairOptions) error {
	meta, err := LoadRunMetadata(RunMetadataPath(runDir))
	if err != nil || meta == nil || strings.TrimSpace(meta.Intent) != runIntentEvolve {
		return err
	}
	facts, err := BuildEvolveFacts(runDir)
	if err != nil || facts == nil || strings.TrimSpace(facts.FrontierState) == EvolveFrontierStopped {
		return err
	}
	reasonCode := strings.TrimSpace(opts.EvolveReasonCode)
	if reasonCode == "" {
		reasonCode = "user_redirected"
	}
	reason := strings.TrimSpace(opts.EvolveReason)
	if reason == "" {
		if strings.TrimSpace(opts.Origin) == "repair" {
			reason = "operator repaired a previously stopped run and recorded the closed evolve frontier"
		} else {
			reason = "run was explicitly stopped by operator"
		}
	}
	return appendEvolveStopped(runDir, EvolveStoppedBody{
		ReasonCode:       reasonCode,
		Reason:           reason,
		BestExperimentID: strings.TrimSpace(facts.BestExperimentID),
		StoppedAt:        stoppedAt,
	})
}

func repairStoppedCoordinationState(runDir, updatedAt string) error {
	coord, err := LoadCoordinationState(CoordinationPath(runDir))
	if err != nil || coord == nil {
		return err
	}
	sessionState, err := LoadSessionsRuntimeState(SessionsRuntimeStatePath(runDir))
	if err != nil {
		return err
	}
	changed := false
	for name, session := range coord.Sessions {
		if sessionState == nil || sessionState.Sessions == nil {
			continue
		}
		runtimeSession, ok := sessionState.Sessions[name]
		if !ok {
			continue
		}
		runtimeLabel := strings.TrimSpace(runtimeSession.State)
		if runtimeLabel == "" || session.State == runtimeLabel {
			continue
		}
		session.State = runtimeLabel
		coord.Sessions[name] = session
		changed = true
	}
	if !changed {
		return nil
	}
	coord.UpdatedAt = updatedAt
	return SaveCoordinationState(CoordinationPath(runDir), coord)
}
