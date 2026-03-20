package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAutoPostsCompletionWebhookWhenConfigured(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".goalx"), 0o755); err != nil {
		t.Fatalf("mkdir goalx dir: %v", err)
	}

	var payload autoCompletionPayload
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := []byte(strings.TrimSpace(`
name: demo-run
objective: ship it
target:
  files: [README.md]
harness:
  command: go test ./...
serve:
  notification_url: ` + server.URL + `
`) + "\n")
	if err := os.WriteFile(filepath.Join(projectRoot, ".goalx", "goalx.yaml"), cfg, 0o644); err != nil {
		t.Fatalf("write goalx.yaml: %v", err)
	}

	oldInit := autoInit
	oldStart := autoStart
	oldSave := autoSave
	oldKeep := autoKeep
	oldDrop := autoDrop
	oldPollUntilComplete := autoPollUntilComplete
	autoInit = func(string, []string) error { return nil }
	autoStart = func(string, []string) error { return nil }
	autoSave = func(string, []string) error { return nil }
	autoKeep = func(string, []string) error { return nil }
	autoDrop = func(string, []string) error { return nil }
	autoPollUntilComplete = func(string, time.Duration, time.Duration) (*statusJSON, error) {
		return &statusJSON{
			Phase:          "complete",
			Recommendation: "done",
			AcceptanceMet:  true,
			KeepSession:    "session-1",
			NextObjective:  "",
		}, nil
	}
	defer func() {
		autoInit = oldInit
		autoStart = oldStart
		autoSave = oldSave
		autoKeep = oldKeep
		autoDrop = oldDrop
		autoPollUntilComplete = oldPollUntilComplete
	}()

	if err := Auto(projectRoot, []string{"ship it"}); err != nil {
		t.Fatalf("Auto: %v", err)
	}

	if authHeader != "" {
		t.Fatalf("Authorization header = %q, want empty", authHeader)
	}
	if payload.Event != "goalx.auto.complete" {
		t.Fatalf("event = %q, want goalx.auto.complete", payload.Event)
	}
	if payload.Run != "demo-run" {
		t.Fatalf("run = %q, want demo-run", payload.Run)
	}
	if payload.Recommendation != "done" {
		t.Fatalf("recommendation = %q, want done", payload.Recommendation)
	}
	if !payload.AcceptanceMet {
		t.Fatal("acceptance_met = false, want true")
	}
	if payload.KeepSession != "session-1" {
		t.Fatalf("keep_session = %q, want session-1", payload.KeepSession)
	}
	if payload.CompletedAt == "" {
		t.Fatal("completed_at is empty")
	}
}

func TestAutoIgnoresCompletionWebhookFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".goalx"), 0o755); err != nil {
		t.Fatalf("mkdir goalx dir: %v", err)
	}
	cfg := []byte(strings.TrimSpace(`
name: demo-run
objective: ship it
target:
  files: [README.md]
harness:
  command: go test ./...
serve:
  notification_url: ://bad-url
`) + "\n")
	if err := os.WriteFile(filepath.Join(projectRoot, ".goalx", "goalx.yaml"), cfg, 0o644); err != nil {
		t.Fatalf("write goalx.yaml: %v", err)
	}

	oldInit := autoInit
	oldStart := autoStart
	oldSave := autoSave
	oldKeep := autoKeep
	oldDrop := autoDrop
	oldPollUntilComplete := autoPollUntilComplete
	autoInit = func(string, []string) error { return nil }
	autoStart = func(string, []string) error { return nil }
	autoSave = func(string, []string) error { return nil }
	autoKeep = func(string, []string) error { return nil }
	autoDrop = func(string, []string) error { return nil }
	autoPollUntilComplete = func(string, time.Duration, time.Duration) (*statusJSON, error) {
		return &statusJSON{
			Phase:          "complete",
			Recommendation: "done",
			AcceptanceMet:  true,
		}, nil
	}
	defer func() {
		autoInit = oldInit
		autoStart = oldStart
		autoSave = oldSave
		autoKeep = oldKeep
		autoDrop = oldDrop
		autoPollUntilComplete = oldPollUntilComplete
	}()

	if err := Auto(projectRoot, []string{"ship it"}); err != nil {
		t.Fatalf("Auto should ignore webhook failure, got: %v", err)
	}
}
