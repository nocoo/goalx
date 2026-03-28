package cli

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestDurableLogParsesCanonicalEnvelope(t *testing.T) {
	runDir := t.TempDir()
	path := ExperimentsLogPath(runDir)
	payload := `{"version":1,"kind":"experiment.created","at":"2026-03-28T10:00:00Z","actor":"master","body":{"experiment_id":"exp-1","created_at":"2026-03-28T10:00:00Z"}}`
	if err := AppendDurableLog(path, DurableSurfaceExperiments, []byte(payload)); err != nil {
		t.Fatalf("AppendDurableLog: %v", err)
	}
	events, err := LoadDurableLog(path, DurableSurfaceExperiments)
	if err != nil {
		t.Fatalf("LoadDurableLog: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if events[0].Kind != "experiment.created" || events[0].Actor != "master" {
		t.Fatalf("unexpected event: %#v", events[0])
	}
}

func TestDurableLogRejectsUnknownKind(t *testing.T) {
	_, err := parseDurableLogBuffer([]byte(`{"version":1,"kind":"mystery","at":"2026-03-28T10:00:00Z","actor":"master","body":{"note":"x"}}`), DurableSurfaceExperiments)
	if err == nil || !strings.Contains(err.Error(), "invalid durable log kind") {
		t.Fatalf("parseDurableLogBuffer error = %v, want invalid kind", err)
	}
}

func TestDurableLogRejectsNonObjectBody(t *testing.T) {
	_, err := parseDurableLogBuffer([]byte(`{"version":1,"kind":"experiment.integrated","at":"2026-03-28T10:00:00Z","actor":"master","body":"oops"}`), DurableSurfaceExperiments)
	if err == nil || !strings.Contains(err.Error(), "body must be a JSON object") {
		t.Fatalf("parseDurableLogBuffer error = %v, want body failure", err)
	}
}

func TestAppendDurableLogConcurrentWritersPreserveAllEvents(t *testing.T) {
	runDir := t.TempDir()
	path := ExperimentsLogPath(runDir)

	const writers = 24
	start := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan error, writers)
	for i := 0; i < writers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			payload := fmt.Sprintf(`{"version":1,"kind":"experiment.created","at":"2026-03-28T10:00:%02dZ","actor":"master","body":{"experiment_id":"exp-%02d","created_at":"2026-03-28T10:00:%02dZ"}}`, i, i, i)
			if err := AppendDurableLog(path, DurableSurfaceExperiments, []byte(payload)); err != nil {
				errCh <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("AppendDurableLog concurrent append: %v", err)
		}
	}

	events, err := LoadDurableLog(path, DurableSurfaceExperiments)
	if err != nil {
		t.Fatalf("LoadDurableLog: %v", err)
	}
	if len(events) != writers {
		t.Fatalf("events len = %d, want %d", len(events), writers)
	}
}
