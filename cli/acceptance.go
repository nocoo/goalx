package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	goalx "github.com/vonbai/goalx"
)

const (
	acceptanceStatusPending = "pending"
	acceptanceStatusPassed  = "passed"
	acceptanceStatusFailed  = "failed"

	acceptanceChangeSame      = "same"
	acceptanceChangeExpanded  = "expanded"
	acceptanceChangeRewritten = "rewritten"
	acceptanceChangeNarrowed  = "narrowed"
)

type AcceptanceResult struct {
	Status       string `json:"status,omitempty"`
	CheckedAt    string `json:"checked_at,omitempty"`
	ExitCode     *int   `json:"exit_code,omitempty"`
	EvidencePath string `json:"evidence_path,omitempty"`
}

type AcceptanceState struct {
	Version          int              `json:"version"`
	GoalVersion      int              `json:"goal_version,omitempty"`
	DefaultCommand   string           `json:"default_command,omitempty"`
	EffectiveCommand string           `json:"effective_command,omitempty"`
	ChangeKind       string           `json:"change_kind,omitempty"`
	ChangeReason     string           `json:"change_reason,omitempty"`
	UserApproved     bool             `json:"user_approved,omitempty"`
	LastResult       AcceptanceResult `json:"last_result,omitempty"`
	UpdatedAt        string           `json:"updated_at,omitempty"`
}

func AcceptanceNotesPath(runDir string) string {
	return filepath.Join(runDir, "acceptance.md")
}

func AcceptanceStatePath(runDir string) string {
	return filepath.Join(runDir, "acceptance.json")
}

func AcceptanceEvidencePath(runDir string) string {
	return filepath.Join(runDir, "acceptance-last.txt")
}

func NewAcceptanceState(cfg *goalx.Config, goalVersion int) *AcceptanceState {
	cmd, _ := goalx.ResolveAcceptanceCommandSource(cfg)
	return &AcceptanceState{
		Version:          1,
		GoalVersion:      goalVersion,
		DefaultCommand:   cmd,
		EffectiveCommand: cmd,
		ChangeKind:       acceptanceChangeSame,
		LastResult: AcceptanceResult{
			Status: acceptanceStatusPending,
		},
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func LoadAcceptanceState(path string) (*AcceptanceState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state AcceptanceState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	normalizeAcceptanceState(&state)
	return &state, nil
}

func SaveAcceptanceState(path string, state *AcceptanceState) error {
	if state == nil {
		return fmt.Errorf("acceptance state is nil")
	}
	normalizeAcceptanceState(state)
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func EnsureAcceptanceState(runDir string, cfg *goalx.Config, goalVersion int) (*AcceptanceState, error) {
	path := AcceptanceStatePath(runDir)
	state, err := LoadAcceptanceState(path)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = NewAcceptanceState(cfg, goalVersion)
		if err := SaveAcceptanceState(path, state); err != nil {
			return nil, err
		}
		return state, nil
	}

	defaultCommand, _ := goalx.ResolveAcceptanceCommandSource(cfg)
	if strings.TrimSpace(state.DefaultCommand) == "" {
		state.DefaultCommand = defaultCommand
	}
	if strings.TrimSpace(state.EffectiveCommand) == "" {
		state.EffectiveCommand = state.DefaultCommand
	}
	if state.GoalVersion <= 0 {
		state.GoalVersion = goalVersion
	}
	normalizeAcceptanceState(state)
	if err := SaveAcceptanceState(path, state); err != nil {
		return nil, err
	}
	return state, nil
}

func ValidateAcceptanceStateForVerification(state *AcceptanceState, goal *GoalState) error {
	if state == nil {
		return fmt.Errorf("acceptance state is nil")
	}
	if strings.TrimSpace(state.EffectiveCommand) == "" {
		return fmt.Errorf("no acceptance command configured")
	}
	if goal != nil && goal.Version > 0 && state.GoalVersion != goal.Version {
		return fmt.Errorf("acceptance goal_version=%d but goal.json version is %d", state.GoalVersion, goal.Version)
	}

	if strings.TrimSpace(state.DefaultCommand) == strings.TrimSpace(state.EffectiveCommand) {
		if state.ChangeKind != acceptanceChangeSame {
			return fmt.Errorf("acceptance change_kind must be %q when effective_command matches default_command", acceptanceChangeSame)
		}
		return nil
	}

	switch state.ChangeKind {
	case acceptanceChangeExpanded, acceptanceChangeRewritten, acceptanceChangeNarrowed:
	default:
		return fmt.Errorf("acceptance command differs from default_command but change_kind is missing or invalid")
	}
	if strings.TrimSpace(state.ChangeReason) == "" {
		return fmt.Errorf("acceptance command differs from default_command but change_reason is empty")
	}
	if state.ChangeKind == acceptanceChangeNarrowed && !state.UserApproved {
		return fmt.Errorf("narrowed acceptance gate requires explicit user approval")
	}
	return nil
}

func normalizeAcceptanceState(state *AcceptanceState) {
	if state.Version <= 0 {
		state.Version = 1
	}
	if strings.TrimSpace(state.EffectiveCommand) == "" {
		state.EffectiveCommand = strings.TrimSpace(state.DefaultCommand)
	}
	if strings.TrimSpace(state.DefaultCommand) == "" {
		state.DefaultCommand = strings.TrimSpace(state.EffectiveCommand)
	}
	if strings.TrimSpace(state.ChangeKind) == "" {
		if strings.TrimSpace(state.EffectiveCommand) == strings.TrimSpace(state.DefaultCommand) {
			state.ChangeKind = acceptanceChangeSame
		}
	}
	if strings.TrimSpace(state.LastResult.Status) == "" {
		state.LastResult.Status = acceptanceStatusPending
	}
}
