package cli

import "strings"

type RunCloseoutFacts struct {
	StatusPhase      string `json:"status_phase,omitempty"`
	SummaryExists    bool   `json:"summary_exists,omitempty"`
	CompletionExists bool   `json:"completion_exists,omitempty"`
	MasterUnread     int    `json:"master_unread,omitempty"`
	Complete         bool   `json:"complete,omitempty"`
}

func BuildRunCloseoutFacts(runDir string) (RunCloseoutFacts, error) {
	status, err := LoadRunStatusRecord(RunStatusPath(runDir))
	if err != nil {
		return RunCloseoutFacts{}, err
	}
	facts := RunCloseoutFacts{
		SummaryExists:    fileExists(SummaryPath(runDir)),
		CompletionExists: fileExists(CompletionStatePath(runDir)),
		MasterUnread:     unreadControlInboxCount(MasterInboxPath(runDir), MasterCursorPath(runDir)),
	}
	if status != nil {
		facts.StatusPhase = strings.TrimSpace(status.Phase)
	}
	facts.Complete = facts.StatusPhase == "complete" && facts.SummaryExists && facts.CompletionExists
	return facts, nil
}
