package cli

import "testing"

func TestRunCloseoutFactsReadyToFinalize(t *testing.T) {
	tests := []struct {
		name  string
		facts RunCloseoutFacts
		want  bool
	}{
		{
			name:  "incomplete closeout is not ready",
			facts: RunCloseoutFacts{Complete: false, MasterUnread: 0},
			want:  false,
		},
		{
			name:  "complete closeout with no unread inbox is ready",
			facts: RunCloseoutFacts{Complete: true, MasterUnread: 0, ObjectiveIntegrityOK: true},
			want:  true,
		},
		{
			name:  "complete closeout with unread inbox stays open",
			facts: RunCloseoutFacts{Complete: true, MasterUnread: 1, ObjectiveIntegrityOK: true},
			want:  false,
		},
		{
			name:  "complete closeout with unlocked objective contract stays open",
			facts: RunCloseoutFacts{Complete: true, MasterUnread: 0, ObjectiveContractPresent: true, ObjectiveContractLocked: false, ObjectiveIntegrityOK: false},
			want:  false,
		},
		{
			name:  "complete closeout with missing objective coverage stays open",
			facts: RunCloseoutFacts{Complete: true, MasterUnread: 0, ObjectiveContractPresent: true, ObjectiveContractLocked: true, ObjectiveIntegrityOK: false},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.facts.ReadyToFinalize(); got != tt.want {
				t.Fatalf("ReadyToFinalize() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestRunCloseoutFactsMaintenanceAction(t *testing.T) {
	tests := []struct {
		name   string
		facts  RunCloseoutFacts
		master TargetPresenceFacts
		want   RunCloseoutMaintenanceAction
	}{
		{
			name:   "incomplete closeout does nothing",
			facts:  RunCloseoutFacts{Complete: false, MasterUnread: 0},
			master: TargetPresenceFacts{State: TargetPresencePresent},
			want:   RunCloseoutMaintenanceActionNone,
		},
		{
			name:   "ready closeout finalizes even with live master",
			facts:  RunCloseoutFacts{Complete: true, MasterUnread: 0, ObjectiveIntegrityOK: true},
			master: TargetPresenceFacts{State: TargetPresencePresent},
			want:   RunCloseoutMaintenanceActionFinalize,
		},
		{
			name:   "unread inbox with live master stays open",
			facts:  RunCloseoutFacts{Complete: true, MasterUnread: 1, ObjectiveIntegrityOK: true},
			master: TargetPresenceFacts{State: TargetPresencePresent},
			want:   RunCloseoutMaintenanceActionNone,
		},
		{
			name:   "unread inbox with missing master requests recovery",
			facts:  RunCloseoutFacts{Complete: true, MasterUnread: 1, ObjectiveIntegrityOK: true},
			master: TargetPresenceFacts{State: TargetPresenceWindowMissing},
			want:   RunCloseoutMaintenanceActionRecoverMaster,
		},
		{
			name:   "objective integrity gap with missing master requests recovery",
			facts:  RunCloseoutFacts{Complete: true, MasterUnread: 0, ObjectiveContractPresent: true, ObjectiveContractLocked: true, ObjectiveIntegrityOK: false},
			master: TargetPresenceFacts{State: TargetPresenceWindowMissing},
			want:   RunCloseoutMaintenanceActionRecoverMaster,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.facts.MaintenanceAction(tt.master); got != tt.want {
				t.Fatalf("MaintenanceAction() = %q, want %q", got, tt.want)
			}
		})
	}
}
