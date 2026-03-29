package cli

import "strings"

type RunCloseoutFacts struct {
	StatusPhase                string   `json:"status_phase,omitempty"`
	SummaryExists              bool     `json:"summary_exists,omitempty"`
	CompletionExists           bool     `json:"completion_exists,omitempty"`
	MasterUnread               int      `json:"master_unread,omitempty"`
	ObjectiveContractPresent   bool     `json:"objective_contract_present,omitempty"`
	ObjectiveContractLocked    bool     `json:"objective_contract_locked,omitempty"`
	ObjectiveIntegrityReady    bool     `json:"objective_integrity_ready,omitempty"`
	ObjectiveIntegrityOK       bool     `json:"objective_integrity_ok,omitempty"`
	MissingGoalClauseIDs       []string `json:"missing_goal_clause_ids,omitempty"`
	MissingAcceptanceClauseIDs []string `json:"missing_acceptance_clause_ids,omitempty"`
	Complete                   bool     `json:"complete,omitempty"`
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
	integrity, err := BuildObjectiveIntegritySummary(runDir)
	if err != nil {
		return RunCloseoutFacts{}, err
	}
	facts.ObjectiveContractPresent = integrity.ContractPresent
	facts.ObjectiveContractLocked = integrity.ContractLocked
	facts.ObjectiveIntegrityReady = integrity.ReadyForNoShrinkEnforcement()
	facts.ObjectiveIntegrityOK = integrity.IntegrityOK()
	facts.MissingGoalClauseIDs = append([]string(nil), integrity.MissingGoalClauseIDs...)
	facts.MissingAcceptanceClauseIDs = append([]string(nil), integrity.MissingAcceptanceClauseIDs...)
	facts.Complete = facts.StatusPhase == "complete" && facts.SummaryExists && facts.CompletionExists
	return facts, nil
}

func (facts RunCloseoutFacts) ReadyToFinalize() bool {
	return facts.Complete && facts.MasterUnread == 0 && facts.objectiveCloseoutReady()
}

func (facts RunCloseoutFacts) objectiveCloseoutReady() bool {
	if facts.ObjectiveContractPresent && !facts.ObjectiveContractLocked {
		return false
	}
	if facts.ObjectiveContractPresent && !facts.ObjectiveIntegrityOK {
		return false
	}
	return true
}

func (facts RunCloseoutFacts) needsMasterFollowup() bool {
	if !facts.Complete {
		return false
	}
	if facts.MasterUnread > 0 {
		return true
	}
	return !facts.objectiveCloseoutReady()
}

func (facts RunCloseoutFacts) MaintenanceAction(master TargetPresenceFacts) RunCloseoutMaintenanceAction {
	if facts.ReadyToFinalize() {
		return RunCloseoutMaintenanceActionFinalize
	}
	if facts.needsMasterFollowup() && targetPresenceMissing(master) {
		return RunCloseoutMaintenanceActionRecoverMaster
	}
	return RunCloseoutMaintenanceActionNone
}
