package cli

import "strings"

type RunCloseoutFacts struct {
	StatusPhase      string `json:"status_phase,omitempty"`
	SummaryExists    bool   `json:"summary_exists,omitempty"`
	CompletionExists bool   `json:"completion_exists,omitempty"`
	MasterUnread     int    `json:"master_unread,omitempty"`
	Complete         bool   `json:"complete,omitempty"`
}

type RunCloseoutMaintenanceAction string

const (
	RunCloseoutMaintenanceActionNone          RunCloseoutMaintenanceAction = ""
	RunCloseoutMaintenanceActionRecoverMaster RunCloseoutMaintenanceAction = "recover_master"
	RunCloseoutMaintenanceActionFinalize      RunCloseoutMaintenanceAction = "finalize"
)

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

func (facts RunCloseoutFacts) ReadyToFinalize() bool {
	return facts.Complete && facts.MasterUnread == 0
}

func (facts RunCloseoutFacts) MaintenanceAction(master TargetPresenceFacts) RunCloseoutMaintenanceAction {
	if facts.ReadyToFinalize() {
		return RunCloseoutMaintenanceActionFinalize
	}
	if facts.Complete && facts.MasterUnread > 0 && targetPresenceMissing(master) {
		return RunCloseoutMaintenanceActionRecoverMaster
	}
	return RunCloseoutMaintenanceActionNone
}
