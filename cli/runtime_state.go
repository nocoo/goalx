package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	goalx "github.com/vonbai/goalx"
)

type RunRuntimeState struct {
	Version            int    `json:"version"`
	Run                string `json:"run"`
	Mode               string `json:"mode,omitempty"`
	Objective          string `json:"objective,omitempty"`
	Active             bool   `json:"active"`
	Phase              string `json:"phase,omitempty"`
	Recommendation     string `json:"recommendation,omitempty"`
	Heartbeat          int64  `json:"heartbeat,omitempty"`
	HeartbeatSeq       int64  `json:"heartbeat_seq,omitempty"`
	HeartbeatLag       int64  `json:"heartbeat_lag,omitempty"`
	MasterWakePending  bool   `json:"master_wake_pending,omitempty"`
	MasterStale        bool   `json:"master_stale,omitempty"`
	MasterStaleSince   string `json:"master_stale_since,omitempty"`
	AcceptanceMet      bool   `json:"acceptance_met,omitempty"`
	AcceptanceStatus   string `json:"acceptance_status,omitempty"`
	GoalContractStatus string `json:"goal_contract_status,omitempty"`
	GoalRequiredTotal  int    `json:"goal_required_total,omitempty"`
	GoalRequiredDone   int    `json:"goal_required_done,omitempty"`
	GoalRequiredRemain int    `json:"goal_required_remaining,omitempty"`
	CodeChanged        bool   `json:"code_changed,omitempty"`
	CompletionMode     string `json:"completion_mode,omitempty"`
	StartedAt          string `json:"started_at,omitempty"`
	StoppedAt          string `json:"stopped_at,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
}

type SessionRuntimeState struct {
	Name             string `json:"name"`
	State            string `json:"state,omitempty"`
	Mode             string `json:"mode,omitempty"`
	Branch           string `json:"branch,omitempty"`
	WorktreePath     string `json:"worktree_path,omitempty"`
	OwnerScope       string `json:"owner_scope,omitempty"`
	BlockedBy        string `json:"blocked_by,omitempty"`
	GuidanceVersion  int    `json:"guidance_version,omitempty"`
	GuidancePending  bool   `json:"guidance_pending,omitempty"`
	LastAckVersion   int    `json:"last_ack_version,omitempty"`
	DirtyFiles       int    `json:"dirty_files,omitempty"`
	DiffStat         string `json:"diff_stat,omitempty"`
	LastRound        int    `json:"last_round,omitempty"`
	LastTestSummary  string `json:"last_test_summary,omitempty"`
	LastJournalState string `json:"last_journal_state,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

type SessionsRuntimeState struct {
	Version   int                            `json:"version"`
	Sessions  map[string]SessionRuntimeState `json:"sessions"`
	UpdatedAt string                         `json:"updated_at,omitempty"`
}

func StateDir(runDir string) string {
	return filepath.Join(runDir, "state")
}

func RunRuntimeStatePath(runDir string) string {
	return filepath.Join(StateDir(runDir), "run.json")
}

func SessionsRuntimeStatePath(runDir string) string {
	return filepath.Join(StateDir(runDir), "sessions.json")
}

func EnsureRuntimeState(runDir string, cfg *goalx.Config) (*RunRuntimeState, error) {
	if err := os.MkdirAll(StateDir(runDir), 0o755); err != nil {
		return nil, err
	}
	state, err := LoadRunRuntimeState(RunRuntimeStatePath(runDir))
	if err != nil {
		return nil, err
	}
	if state == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		state = &RunRuntimeState{
			Version:   1,
			Run:       cfg.Name,
			Mode:      string(cfg.Mode),
			Objective: cfg.Objective,
			Active:    true,
			StartedAt: now,
			UpdatedAt: now,
		}
		if err := SaveRunRuntimeState(RunRuntimeStatePath(runDir), state); err != nil {
			return nil, err
		}
	}
	return state, nil
}

func LoadRunRuntimeState(path string) (*RunRuntimeState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	state := &RunRuntimeState{}
	if len(strings.TrimSpace(string(data))) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("parse run runtime state: %w", err)
	}
	if state.Version == 0 {
		state.Version = 1
	}
	return state, nil
}

func SaveRunRuntimeState(path string, state *RunRuntimeState) error {
	if state == nil {
		return fmt.Errorf("run runtime state is nil")
	}
	if state.Version == 0 {
		state.Version = 1
	}
	if state.UpdatedAt == "" {
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func EnsureSessionsRuntimeState(runDir string) (*SessionsRuntimeState, error) {
	if err := os.MkdirAll(StateDir(runDir), 0o755); err != nil {
		return nil, err
	}
	state, err := LoadSessionsRuntimeState(SessionsRuntimeStatePath(runDir))
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = &SessionsRuntimeState{
			Version:   1,
			Sessions:  map[string]SessionRuntimeState{},
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		if err := SaveSessionsRuntimeState(SessionsRuntimeStatePath(runDir), state); err != nil {
			return nil, err
		}
	}
	return state, nil
}

func LoadSessionsRuntimeState(path string) (*SessionsRuntimeState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	state := &SessionsRuntimeState{}
	if len(strings.TrimSpace(string(data))) == 0 {
		state.Version = 1
		state.Sessions = map[string]SessionRuntimeState{}
		return state, nil
	}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("parse sessions runtime state: %w", err)
	}
	if state.Version == 0 {
		state.Version = 1
	}
	if state.Sessions == nil {
		state.Sessions = map[string]SessionRuntimeState{}
	}
	return state, nil
}

func SaveSessionsRuntimeState(path string, state *SessionsRuntimeState) error {
	if state == nil {
		return fmt.Errorf("sessions runtime state is nil")
	}
	if state.Version == 0 {
		state.Version = 1
	}
	if state.Sessions == nil {
		state.Sessions = map[string]SessionRuntimeState{}
	}
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func UpsertSessionRuntimeState(runDir string, next SessionRuntimeState) error {
	state, err := EnsureSessionsRuntimeState(runDir)
	if err != nil {
		return err
	}
	current := state.Sessions[next.Name]
	mergeSessionRuntimeState(&current, next)
	current.Name = next.Name
	current.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	state.Sessions[next.Name] = current
	return SaveSessionsRuntimeState(SessionsRuntimeStatePath(runDir), state)
}

func mergeSessionRuntimeState(dst *SessionRuntimeState, src SessionRuntimeState) {
	if src.State != "" {
		dst.State = src.State
	}
	if src.Mode != "" {
		dst.Mode = src.Mode
	}
	if src.Branch != "" {
		dst.Branch = src.Branch
	}
	if src.WorktreePath != "" {
		dst.WorktreePath = src.WorktreePath
	}
	if src.OwnerScope != "" {
		dst.OwnerScope = src.OwnerScope
	}
	if src.BlockedBy != "" || src.State == "blocked" {
		dst.BlockedBy = src.BlockedBy
	}
	if src.GuidanceVersion != 0 || src.GuidancePending || src.LastAckVersion != 0 {
		dst.GuidanceVersion = src.GuidanceVersion
		dst.GuidancePending = src.GuidancePending
		dst.LastAckVersion = src.LastAckVersion
	}
	if src.DirtyFiles != 0 || src.DiffStat != "" {
		dst.DirtyFiles = src.DirtyFiles
		dst.DiffStat = src.DiffStat
	}
	if src.LastRound != 0 {
		dst.LastRound = src.LastRound
	}
	if src.LastTestSummary != "" {
		dst.LastTestSummary = src.LastTestSummary
	}
	if src.LastJournalState != "" {
		dst.LastJournalState = src.LastJournalState
	}
}

func SnapshotSessionRuntime(runDir, sessionName, worktreePath string) (SessionRuntimeState, error) {
	dirtyFiles, diffStat, err := snapshotWorktreeState(worktreePath)
	if err != nil {
		return SessionRuntimeState{}, err
	}
	journalEntries, _ := goalx.LoadJournal(JournalPath(runDir, sessionName))
	lastRound := 0
	lastJournalState := ""
	lastTestSummary := ""
	if len(journalEntries) > 0 {
		last := journalEntries[len(journalEntries)-1]
		lastRound = last.Round
		lastJournalState = last.Status
		lastTestSummary = summarizeJournalForTest(journalEntries)
	}
	guidance, _ := LoadSessionGuidanceState(SessionGuidanceStatePath(runDir, sessionName))
	snapshot := SessionRuntimeState{
		Name:             sessionName,
		WorktreePath:     worktreePath,
		DirtyFiles:       dirtyFiles,
		DiffStat:         diffStat,
		LastRound:        lastRound,
		LastJournalState: lastJournalState,
		LastTestSummary:  lastTestSummary,
	}
	if guidance != nil {
		snapshot.GuidanceVersion = guidance.Version
		snapshot.GuidancePending = guidance.Pending
		snapshot.LastAckVersion = guidance.LastAckVersion
	}
	return snapshot, nil
}

func snapshotWorktreeState(worktreePath string) (int, string, error) {
	statusOut, err := exec.Command("git", "-C", worktreePath, "status", "--porcelain").CombinedOutput()
	if err != nil {
		if os.IsNotExist(err) || bytes.Contains(bytes.ToLower(statusOut), []byte("not a git repository")) {
			return 0, "", nil
		}
		return 0, "", fmt.Errorf("git status in %s: %w: %s", worktreePath, err, statusOut)
	}
	dirty := 0
	for _, line := range strings.Split(strings.TrimSpace(string(statusOut)), "\n") {
		if strings.TrimSpace(line) != "" {
			dirty++
		}
	}
	diffOut, err := exec.Command("git", "-C", worktreePath, "diff", "--stat").CombinedOutput()
	if err != nil {
		if bytes.Contains(bytes.ToLower(diffOut), []byte("not a git repository")) {
			return dirty, "", nil
		}
		return dirty, "", fmt.Errorf("git diff --stat in %s: %w: %s", worktreePath, err, diffOut)
	}
	return dirty, strings.TrimSpace(string(diffOut)), nil
}

func summarizeJournalForTest(entries []goalx.JournalEntry) string {
	for i := len(entries) - 1; i >= 0; i-- {
		desc := strings.TrimSpace(entries[i].Desc)
		if desc == "" {
			continue
		}
		lower := strings.ToLower(desc)
		if strings.Contains(lower, "test") || strings.Contains(lower, "pytest") || strings.Contains(lower, "lint") || strings.Contains(lower, "build") {
			return desc
		}
	}
	return ""
}

func sortedSessionStates(state *SessionsRuntimeState) []SessionRuntimeState {
	if state == nil {
		return nil
	}
	list := make([]SessionRuntimeState, 0, len(state.Sessions))
	for _, sess := range state.Sessions {
		list = append(list, sess)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

func deriveProjectStatusFromRun(state *RunRuntimeState) []byte {
	if state == nil {
		return nil
	}
	payload := map[string]any{
		"run":                     state.Run,
		"phase":                   state.Phase,
		"recommendation":          state.Recommendation,
		"heartbeat":               state.Heartbeat,
		"heartbeat_seq":           state.HeartbeatSeq,
		"heartbeat_lag":           state.HeartbeatLag,
		"master_wake_pending":     state.MasterWakePending,
		"master_stale":            state.MasterStale,
		"master_stale_since":      state.MasterStaleSince,
		"acceptance_met":          state.AcceptanceMet,
		"acceptance_status":       state.AcceptanceStatus,
		"goal_contract_status":    state.GoalContractStatus,
		"goal_required_total":     state.GoalRequiredTotal,
		"goal_required_done":      state.GoalRequiredDone,
		"goal_required_remaining": state.GoalRequiredRemain,
		"completion_mode":         state.CompletionMode,
		"code_changed":            state.CodeChanged,
		"active":                  state.Active,
	}
	data, _ := json.Marshal(payload)
	return data
}

func syncProjectStatusCache(projectRoot string, state *RunRuntimeState) error {
	statusPath := filepath.Join(projectRoot, ".goalx", "status.json")
	if state == nil {
		if err := os.Remove(statusPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	data := deriveProjectStatusFromRun(state)
	if err := os.MkdirAll(filepath.Dir(statusPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(statusPath, data, 0o644)
}

func refreshProjectStatusCache(projectRoot string) error {
	reg, err := LoadProjectRegistry(projectRoot)
	if err != nil {
		return err
	}
	if reg.FocusedRun != "" {
		if _, ok := reg.ActiveRuns[reg.FocusedRun]; ok {
			runDir := goalx.RunDir(projectRoot, reg.FocusedRun)
			state, err := LoadRunRuntimeState(RunRuntimeStatePath(runDir))
			if err != nil {
				return err
			}
			return syncProjectStatusCache(projectRoot, state)
		}
	}
	if len(reg.ActiveRuns) == 1 {
		for name := range reg.ActiveRuns {
			runDir := goalx.RunDir(projectRoot, name)
			state, err := LoadRunRuntimeState(RunRuntimeStatePath(runDir))
			if err != nil {
				return err
			}
			return syncProjectStatusCache(projectRoot, state)
		}
	}
	return syncProjectStatusCache(projectRoot, nil)
}

func syncRunStateFromProjectStatus(projectRoot, runDir string) error {
	statusPath := filepath.Join(projectRoot, ".goalx", "status.json")
	data, err := os.ReadFile(statusPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}
	payload := map[string]any{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parse status cache: %w", err)
	}
	state, err := LoadRunRuntimeState(RunRuntimeStatePath(runDir))
	if err != nil {
		return err
	}
	if state == nil {
		return nil
	}
	if run, _ := payload["run"].(string); run != "" && state.Run != "" && run != state.Run {
		return nil
	}
	if err := updateRunStateFromStatusJSON(runDir, payload); err != nil {
		return err
	}
	state, err = LoadRunRuntimeState(RunRuntimeStatePath(runDir))
	if err != nil {
		return err
	}
	return syncProjectStatusCache(projectRoot, state)
}

func updateRunStateFromStatusJSON(runDir string, status map[string]any) error {
	state, err := LoadRunRuntimeState(RunRuntimeStatePath(runDir))
	if err != nil {
		return err
	}
	if state == nil {
		return nil
	}
	applyStatusMapToRunState(state, status)
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return SaveRunRuntimeState(RunRuntimeStatePath(runDir), state)
}

func applyStatusMapToRunState(state *RunRuntimeState, payload map[string]any) {
	if state == nil {
		return
	}
	if v, ok := payload["phase"].(string); ok {
		state.Phase = v
	}
	if v, ok := payload["recommendation"].(string); ok {
		state.Recommendation = v
	}
	if v, ok := asInt64(payload["heartbeat"]); ok {
		state.Heartbeat = v
	}
	if v, ok := asInt64(payload["heartbeat_seq"]); ok {
		state.HeartbeatSeq = v
	}
	if v, ok := asInt64(payload["heartbeat_lag"]); ok {
		state.HeartbeatLag = v
	}
	if v, ok := payload["master_wake_pending"].(bool); ok {
		state.MasterWakePending = v
	}
	if v, ok := payload["master_stale"].(bool); ok {
		state.MasterStale = v
	}
	if v, ok := payload["master_stale_since"].(string); ok {
		state.MasterStaleSince = v
	}
	if v, ok := payload["acceptance_met"].(bool); ok {
		state.AcceptanceMet = v
	}
	if v, ok := payload["acceptance_status"].(string); ok {
		state.AcceptanceStatus = v
	}
	if v, ok := payload["goal_contract_status"].(string); ok {
		state.GoalContractStatus = v
	}
	if v, ok := asInt(payload["goal_required_total"]); ok {
		state.GoalRequiredTotal = v
	}
	if v, ok := asInt(payload["goal_required_done"]); ok {
		state.GoalRequiredDone = v
	}
	if v, ok := asInt(payload["goal_required_remaining"]); ok {
		state.GoalRequiredRemain = v
	}
	if v, ok := payload["completion_mode"].(string); ok {
		state.CompletionMode = v
	}
	if v, ok := payload["code_changed"].(bool); ok {
		state.CodeChanged = v
	}
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func normalizeDiffStat(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func mergeDiffStat(lines string) string {
	if strings.TrimSpace(lines) == "" {
		return ""
	}
	var buf bytes.Buffer
	for _, line := range strings.Split(strings.TrimSpace(lines), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if buf.Len() > 0 {
			buf.WriteString(" | ")
		}
		buf.WriteString(normalizeDiffStat(line))
	}
	return buf.String()
}
