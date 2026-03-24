package cli

import (
	"testing"
)

func TestValidateCompletionProofStructureRequiresCharter(t *testing.T) {
	runDir := t.TempDir()

	err := ValidateCompletionProofStructure(runDir, &CompletionState{Version: 1}, nil)
	if err == nil {
		t.Fatal("expected error for missing charter")
	}
}

func TestValidateRunCharterStructureRejectsEmptyID(t *testing.T) {
	if err := ValidateRunCharterStructure(&RunCharter{Version: 1}); err == nil {
		t.Fatal("expected error for empty charter_id")
	}
	if err := ValidateRunCharterStructure(&RunCharter{Version: 1, CharterID: "charter-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
