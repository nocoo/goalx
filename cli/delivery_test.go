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
		return TransportDeliveryOutcome{SubmitMode: "payload_enter", TransportState: "sent"}, nil
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
	if deliveries.Items[0].Status != "sent" || deliveries.Items[0].DedupeKey != "tell:1" {
		t.Fatalf("unexpected delivery: %+v", deliveries.Items[0])
	}
	if deliveries.Items[0].SubmitMode != "payload_enter" || deliveries.Items[0].TransportState != "sent" {
		t.Fatalf("delivery metadata = %+v, want submit_mode payload_enter and transport_state sent", deliveries.Items[0])
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
		return TransportDeliveryOutcome{SubmitMode: "payload_enter", TransportState: "sent"}, nil
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
