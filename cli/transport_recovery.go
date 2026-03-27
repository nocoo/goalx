package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type TransportRecoveryState struct {
	Version   int                                `json:"version"`
	Targets   map[string]TransportRecoveryTarget `json:"targets,omitempty"`
	UpdatedAt string                             `json:"updated_at,omitempty"`
}

type TransportRecoveryTarget struct {
	Target                    string `json:"target,omitempty"`
	LastWakeSubmitAt          string `json:"last_wake_submit_at,omitempty"`
	LastEnterRepairAt         string `json:"last_enter_repair_at,omitempty"`
	LastInterruptEscalationAt string `json:"last_interrupt_escalation_at,omitempty"`
	LastInterruptReason       string `json:"last_interrupt_reason,omitempty"`
	LastInterruptResultingState string `json:"last_interrupt_resulting_state,omitempty"`
	UrgentEscalationAttempts  int    `json:"urgent_escalation_attempts,omitempty"`
}

func TransportRecoveryPath(runDir string) string {
	return filepath.Join(ControlDir(runDir), "transport-recovery.json")
}

func LoadTransportRecovery(path string) (*TransportRecoveryState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	state := &TransportRecoveryState{}
	if len(strings.TrimSpace(string(data))) == 0 {
		state.Version = 1
		state.Targets = map[string]TransportRecoveryTarget{}
		return state, nil
	}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("parse transport recovery: %w", err)
	}
	if state.Version == 0 {
		state.Version = 1
	}
	if state.Targets == nil {
		state.Targets = map[string]TransportRecoveryTarget{}
	}
	return state, nil
}

func SaveTransportRecovery(path string, state *TransportRecoveryState) error {
	if state == nil {
		return fmt.Errorf("transport recovery is nil")
	}
	if state.Version == 0 {
		state.Version = 1
	}
	if state.Targets == nil {
		state.Targets = map[string]TransportRecoveryTarget{}
	}
	if state.UpdatedAt == "" {
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return writeJSONFile(path, state)
}

func ensureTransportRecovery(runDir string) error {
	if _, err := LoadTransportRecovery(TransportRecoveryPath(runDir)); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return SaveTransportRecovery(TransportRecoveryPath(runDir), &TransportRecoveryState{
			Version: 1,
			Targets: map[string]TransportRecoveryTarget{},
		})
	}
	return nil
}

func loadTransportRecoveryTarget(runDir, target string) TransportRecoveryTarget {
	state, err := LoadTransportRecovery(TransportRecoveryPath(runDir))
	if err != nil || state == nil || state.Targets == nil {
		return TransportRecoveryTarget{}
	}
	return state.Targets[target]
}

func updateTransportRecoveryTarget(runDir, target string, mutate func(*TransportRecoveryTarget)) error {
	if err := EnsureControlState(runDir); err != nil {
		return err
	}
	state, err := LoadTransportRecovery(TransportRecoveryPath(runDir))
	if err != nil {
		return err
	}
	entry := state.Targets[target]
	entry.Target = target
	mutate(&entry)
	if strings.TrimSpace(entry.Target) == "" {
		delete(state.Targets, target)
	} else {
		state.Targets[target] = entry
	}
	state.UpdatedAt = ""
	return SaveTransportRecovery(TransportRecoveryPath(runDir), state)
}

func recordWakeSubmit(runDir, target string, outcome TransportDeliveryOutcome) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return updateTransportRecoveryTarget(runDir, target, func(entry *TransportRecoveryTarget) {
		switch strings.TrimSpace(outcome.SubmitMode) {
		case "enter_only_repair":
			entry.LastEnterRepairAt = now
		case "payload_enter", "payload_then_enter":
			entry.LastWakeSubmitAt = now
		}
	})
}

func recordInterruptEscalation(runDir, target, reason string, outcome TransportDeliveryOutcome) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return updateTransportRecoveryTarget(runDir, target, func(entry *TransportRecoveryTarget) {
		entry.LastInterruptEscalationAt = now
		entry.LastInterruptReason = strings.TrimSpace(reason)
		entry.LastInterruptResultingState = strings.TrimSpace(outcome.TransportState)
		entry.UrgentEscalationAttempts++
	})
}

func resetUrgentEscalationAttempts(runDir, target string) error {
	return updateTransportRecoveryTarget(runDir, target, func(entry *TransportRecoveryTarget) {
		entry.UrgentEscalationAttempts = 0
	})
}
