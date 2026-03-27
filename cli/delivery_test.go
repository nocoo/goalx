package cli

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestDeliverControlNudgeRecordsSentAndDedupesByKey(t *testing.T) {
	runDir := t.TempDir()
	if err := EnsureControlState(runDir); err != nil {
		t.Fatalf("EnsureControlState: %v", err)
	}

	calls := 0
	send := func(target, engine string) (TransportDeliveryOutcome, error) {
		calls++
		return TransportDeliveryOutcome{SubmitMode: "payload_enter", TransportState: "queued"}, nil
	}

	if _, err := DeliverControlNudge(runDir, "tell:1", "tell:1", "gx-demo:master", "codex", send); err != nil {
		t.Fatalf("DeliverControlNudge first: %v", err)
	}
	if _, err := DeliverControlNudge(runDir, "tell:1", "tell:1", "gx-demo:master", "codex", send); err != nil {
		t.Fatalf("DeliverControlNudge second: %v", err)
	}

	deliveries, err := LoadControlDeliveries(ControlDeliveriesPath(runDir))
	if err != nil {
		t.Fatalf("LoadControlDeliveries: %v", err)
	}
	if len(deliveries.Items) != 1 {
		t.Fatalf("deliveries len = %d, want 1", len(deliveries.Items))
	}
	if deliveries.Items[0].Status != "accepted" || deliveries.Items[0].DedupeKey != "tell:1" {
		t.Fatalf("unexpected delivery: %+v", deliveries.Items[0])
	}
	if deliveries.Items[0].SubmitMode != "payload_enter" || deliveries.Items[0].TransportState != "queued" {
		t.Fatalf("delivery metadata = %+v, want submit_mode payload_enter and transport_state queued", deliveries.Items[0])
	}
	if calls != 1 {
		t.Fatalf("deliver calls = %d, want 1", calls)
	}
}

func TestDeliverControlNudgeRecordsFailure(t *testing.T) {
	runDir := t.TempDir()
	if err := EnsureControlState(runDir); err != nil {
		t.Fatalf("EnsureControlState: %v", err)
	}

	_, err := DeliverControlNudge(runDir, "tell:2", "tell:2", "gx-demo:master", "codex", func(target, engine string) (TransportDeliveryOutcome, error) {
		return TransportDeliveryOutcome{}, errors.New("tmux unavailable")
	})
	if err == nil {
		t.Fatal("DeliverControlNudge error = nil, want failure")
	}

	deliveries, loadErr := LoadControlDeliveries(ControlDeliveriesPath(runDir))
	if loadErr != nil {
		t.Fatalf("LoadControlDeliveries: %v", loadErr)
	}
	if len(deliveries.Items) != 1 {
		t.Fatalf("deliveries len = %d, want 1", len(deliveries.Items))
	}
	if deliveries.Items[0].Status != "failed" || deliveries.Items[0].LastError == "" {
		t.Fatalf("unexpected delivery: %+v", deliveries.Items[0])
	}
}

func TestDeliverControlNudgeRecordsBufferedTransportMetadata(t *testing.T) {
	runDir := t.TempDir()
	if err := EnsureControlState(runDir); err != nil {
		t.Fatalf("EnsureControlState: %v", err)
	}

	if _, err := DeliverControlNudge(runDir, "tell:3", "tell:3", "gx-demo:session-2", "codex", func(target, engine string) (TransportDeliveryOutcome, error) {
		return TransportDeliveryOutcome{SubmitMode: "enter_only_repair", TransportState: "buffered"}, nil
	}); err != nil {
		t.Fatalf("DeliverControlNudge: %v", err)
	}

	deliveries, err := LoadControlDeliveries(ControlDeliveriesPath(runDir))
	if err != nil {
		t.Fatalf("LoadControlDeliveries: %v", err)
	}
	if len(deliveries.Items) != 1 {
		t.Fatalf("deliveries len = %d, want 1", len(deliveries.Items))
	}
	got := deliveries.Items[0]
	if got.Status != "buffered" || got.TransportState != "buffered" || got.SubmitMode != "enter_only_repair" {
		t.Fatalf("unexpected buffered delivery: %+v", got)
	}
}

func TestDeliverControlNudgePersistsAcceptedAtWithoutLegacyAckedAt(t *testing.T) {
	runDir := t.TempDir()
	if err := EnsureControlState(runDir); err != nil {
		t.Fatalf("EnsureControlState: %v", err)
	}

	if _, err := DeliverControlNudge(runDir, "tell:4", "tell:4", "gx-demo:master", "codex", func(target, engine string) (TransportDeliveryOutcome, error) {
		return TransportDeliveryOutcome{SubmitMode: "payload_enter", TransportState: "queued"}, nil
	}); err != nil {
		t.Fatalf("DeliverControlNudge: %v", err)
	}

	data, err := os.ReadFile(ControlDeliveriesPath(runDir))
	if err != nil {
		t.Fatalf("ReadFile deliveries: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "\"accepted_at\":") {
		t.Fatalf("deliveries should persist accepted_at:\n%s", text)
	}
	if strings.Contains(text, "\"acked_at\":") {
		t.Fatalf("deliveries should not persist legacy acked_at:\n%s", text)
	}
}

func TestDeliverControlNudgeAcceptsOnlyCanonicalAcceptedStates(t *testing.T) {
	runDir := t.TempDir()
	if err := EnsureControlState(runDir); err != nil {
		t.Fatalf("EnsureControlState: %v", err)
	}

	tests := []struct {
		name         string
		state        string
		wantAccepted bool
	}{
		{name: "queued", state: "queued", wantAccepted: true},
		{name: "working", state: "working", wantAccepted: true},
		{name: "compacting", state: "compacting", wantAccepted: true},
		{name: "buffered", state: "buffered_input", wantAccepted: false},
		{name: "interrupted", state: "interrupted", wantAccepted: false},
		{name: "provider dialog", state: "provider_dialog", wantAccepted: false},
		{name: "unknown", state: "unknown", wantAccepted: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dedupeKey := "tell:" + tt.state
			got, err := DeliverControlNudge(runDir, dedupeKey, dedupeKey, "gx-demo:master", "codex", func(target, engine string) (TransportDeliveryOutcome, error) {
				return TransportDeliveryOutcome{SubmitMode: "payload_enter", TransportState: tt.state}, nil
			})
			if err != nil {
				t.Fatalf("DeliverControlNudge: %v", err)
			}
			if tt.wantAccepted {
				if got.Status != "accepted" {
					t.Fatalf("status = %q, want accepted for %s", got.Status, tt.state)
				}
				if got.AcceptedAt == "" {
					t.Fatalf("accepted_at empty for accepted state %s: %+v", tt.state, got)
				}
				return
			}
			if got.AcceptedAt != "" {
				t.Fatalf("accepted_at = %q, want empty for non-accepted state %s", got.AcceptedAt, tt.state)
			}
			if got.Status != tt.state {
				t.Fatalf("status = %q, want %q", got.Status, tt.state)
			}
		})
	}
}

func TestAckControlInboxReconcilesPendingSessionDelivery(t *testing.T) {
	runDir := t.TempDir()
	if err := EnsureControlState(runDir); err != nil {
		t.Fatalf("EnsureControlState: %v", err)
	}
	if err := EnsureSessionControl(runDir, "session-1"); err != nil {
		t.Fatalf("EnsureSessionControl: %v", err)
	}
	msg, err := AppendControlInboxMessage(runDir, "session-1", "develop", "master", "take the next slice")
	if err != nil {
		t.Fatalf("AppendControlInboxMessage: %v", err)
	}
	if err := SaveControlDeliveries(ControlDeliveriesPath(runDir), &ControlDeliveries{
		Version: 1,
		Items: []ControlDelivery{
			{
				DeliveryID:  "del-1",
				MessageID:   "session-inbox:session-1:1",
				DedupeKey:   "session-inbox:session-1:1",
				Target:      "gx-demo:session-1",
				Status:      "pending",
				AttemptedAt: "2026-03-27T00:00:00Z",
			},
		},
	}); err != nil {
		t.Fatalf("SaveControlDeliveries: %v", err)
	}

	cursor, err := AckControlInbox(runDir, "session-1")
	if err != nil {
		t.Fatalf("AckControlInbox: %v", err)
	}
	if cursor.LastSeenID != msg.ID {
		t.Fatalf("last seen id = %d, want %d", cursor.LastSeenID, msg.ID)
	}

	deliveries, err := LoadControlDeliveries(ControlDeliveriesPath(runDir))
	if err != nil {
		t.Fatalf("LoadControlDeliveries: %v", err)
	}
	if len(deliveries.Items) != 1 {
		t.Fatalf("deliveries len = %d, want 1", len(deliveries.Items))
	}
	got := deliveries.Items[0]
	if got.Status != "accepted" {
		t.Fatalf("delivery status = %q, want accepted after cursor reconciliation", got.Status)
	}
	if got.TransportState != "" {
		t.Fatalf("transport state = %q, want empty when cursor ack reconciles a previously unknown transport state", got.TransportState)
	}
	if got.AcceptedAt == "" {
		t.Fatalf("accepted_at empty after cursor reconciliation: %+v", got)
	}
}
