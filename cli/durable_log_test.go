package cli

import (
	"strings"
	"testing"
)

func TestDurableLogParsesCanonicalEnvelope(t *testing.T) {
	runDir := t.TempDir()
	path := EvolutionLogPath(runDir)
	payload := `{"version":1,"kind":"trial","at":"2026-03-28T10:00:00Z","actor":"master","body":{"hypothesis":"x"}}`
	if err := AppendDurableLog(path, DurableSurfaceEvolution, []byte(payload)); err != nil {
		t.Fatalf("AppendDurableLog: %v", err)
	}
	events, err := LoadDurableLog(path, DurableSurfaceEvolution)
	if err != nil {
		t.Fatalf("LoadDurableLog: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if events[0].Kind != "trial" || events[0].Actor != "master" {
		t.Fatalf("unexpected event: %#v", events[0])
	}
}

func TestDurableLogRejectsUnknownKind(t *testing.T) {
	_, err := parseDurableLogBuffer([]byte(`{"version":1,"kind":"mystery","at":"2026-03-28T10:00:00Z","actor":"master","body":{"note":"x"}}`), DurableSurfaceEvolution)
	if err == nil || !strings.Contains(err.Error(), "invalid durable log kind") {
		t.Fatalf("parseDurableLogBuffer error = %v, want invalid kind", err)
	}
}

func TestDurableLogRejectsNonObjectBody(t *testing.T) {
	_, err := parseDurableLogBuffer([]byte(`{"version":1,"kind":"trial","at":"2026-03-28T10:00:00Z","actor":"master","body":"oops"}`), DurableSurfaceEvolution)
	if err == nil || !strings.Contains(err.Error(), "body must be a JSON object") {
		t.Fatalf("parseDurableLogBuffer error = %v, want body failure", err)
	}
}
