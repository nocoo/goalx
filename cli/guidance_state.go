package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SessionGuidanceState struct {
	Version        int    `json:"version"`
	Session        string `json:"session,omitempty"`
	Pending        bool   `json:"pending"`
	UpdatedAt      string `json:"updated_at,omitempty"`
	LastAckVersion int    `json:"last_ack_version,omitempty"`
	LastAckAt      string `json:"last_ack_at,omitempty"`
}

func LoadSessionGuidanceState(path string) (*SessionGuidanceState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state SessionGuidanceState
	if len(strings.TrimSpace(string(data))) == 0 {
		return &state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &state, nil
}

func SaveSessionGuidanceState(path string, state *SessionGuidanceState) error {
	if state == nil {
		return fmt.Errorf("session guidance state is nil")
	}
	if state.Version < 0 {
		state.Version = 0
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func EnsureSessionGuidanceState(runDir, sessionName string) (*SessionGuidanceState, error) {
	path := SessionGuidanceStatePath(runDir, sessionName)
	state, err := LoadSessionGuidanceState(path)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = &SessionGuidanceState{
			Version: 0,
			Session: sessionName,
		}
		if err := SaveSessionGuidanceState(path, state); err != nil {
			return nil, err
		}
		return state, nil
	}
	changed := false
	if state.Session == "" {
		state.Session = sessionName
		changed = true
	}
	if state.Version < 0 {
		state.Version = 0
		changed = true
	}
	if changed {
		if err := SaveSessionGuidanceState(path, state); err != nil {
			return nil, err
		}
	}
	return state, nil
}

func WriteSessionGuidance(runDir, sessionName, message string) error {
	if err := EnsureMasterControl(runDir); err != nil {
		return err
	}
	guidancePath := GuidancePath(runDir, sessionName)
	if err := os.MkdirAll(filepath.Dir(guidancePath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(guidancePath, []byte(strings.TrimRight(message, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	state, err := EnsureSessionGuidanceState(runDir, sessionName)
	if err != nil {
		return err
	}
	state.Version++
	state.Session = sessionName
	state.Pending = true
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return SaveSessionGuidanceState(SessionGuidanceStatePath(runDir, sessionName), state)
}

func AckSessionGuidance(runDir, sessionName string) error {
	if err := EnsureMasterControl(runDir); err != nil {
		return err
	}
	state, err := EnsureSessionGuidanceState(runDir, sessionName)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	state.Pending = false
	state.LastAckVersion = state.Version
	state.LastAckAt = now
	state.UpdatedAt = now
	return SaveSessionGuidanceState(SessionGuidanceStatePath(runDir, sessionName), state)
}
