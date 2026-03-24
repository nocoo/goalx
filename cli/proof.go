package cli

import (
	"fmt"
	"strings"
)

// CompletionProofItem is a master-supplied proof item. Framework stores it
// but never populates verdict, basis, or evidence — that's the master's job.
type CompletionProofItem struct {
	GoalItemID    string   `json:"goal_item_id"`
	Verdict       string   `json:"verdict,omitempty"`
	Basis         string   `json:"basis,omitempty"`
	EvidencePaths []string `json:"evidence_paths,omitempty"`
	Note          string   `json:"note,omitempty"`
}

// ValidateCompletionProofStructure checks structural consistency of a
// master-written proof manifest: charter linkage and item count matching goal.
// It does not enforce verdict, basis, or satisfaction — those are master decisions.
func ValidateCompletionProofStructure(runDir string, completion *CompletionState, goal *GoalState) error {
	if completion == nil {
		return fmt.Errorf("completion proof manifest is missing")
	}
	charter, err := RequireRunCharter(runDir)
	if err != nil {
		return err
	}
	if completion.CharterID != charter.CharterID {
		return fmt.Errorf("completion proof charter_id=%q but run charter is %q", completion.CharterID, charter.CharterID)
	}
	charterHash, err := hashRunCharter(charter)
	if err != nil {
		return err
	}
	if completion.CharterHash != charterHash {
		return fmt.Errorf("completion proof charter_hash=%q but run charter hash is %q", completion.CharterHash, charterHash)
	}
	if goal != nil && len(goal.Required) > 0 {
		proofs := make(map[string]struct{}, len(completion.Items))
		for _, item := range completion.Items {
			proofs[item.GoalItemID] = struct{}{}
		}
		for _, item := range goal.Required {
			if _, ok := proofs[item.ID]; !ok {
				return fmt.Errorf("completion proof missing goal_item_id %s", item.ID)
			}
		}
	}
	return nil
}

// ValidateRunCharterStructure checks that the charter has a valid charter_id.
func ValidateRunCharterStructure(charter *RunCharter) error {
	if charter == nil {
		return fmt.Errorf("run charter is missing")
	}
	if strings.TrimSpace(charter.CharterID) == "" {
		return fmt.Errorf("run charter is missing charter_id")
	}
	return nil
}

