package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestRenderSubagentProtocolIncludesResumeInstructions(t *testing.T) {
	runDir := t.TempDir()
	data := ProtocolData{
		Objective:    "ship it",
		Mode:         goalx.ModeDevelop,
		Target:       goalx.TargetConfig{Files: []string{"main.go"}},
		Harness:      goalx.HarnessConfig{Command: "go test ./..."},
		SessionName:  "session-1",
		JournalPath:  "/tmp/journal.jsonl",
		GuidancePath: "/tmp/guidance.md",
		WorktreePath: "/tmp/worktree",
	}

	if err := RenderSubagentProtocol(data, runDir, 0); err != nil {
		t.Fatalf("RenderSubagentProtocol: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(runDir, "program-1.md"))
	if err != nil {
		t.Fatalf("read rendered protocol: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"Before doing new work, first reconstruct context",
		"Read your existing journal",
		"Read the latest master guidance",
		"Inspect the current worktree state",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered protocol missing %q", want)
		}
	}
}

func TestRenderMasterProtocolIncludesAcceptanceChecklistInstructions(t *testing.T) {
	runDir := t.TempDir()
	data := ProtocolData{
		Objective:      "ship it",
		Mode:           goalx.ModeDevelop,
		TmuxSession:    "ar-demo",
		SummaryPath:    "/tmp/summary.md",
		AcceptancePath: "/tmp/acceptance.md",
		EngineCommand:  "claude --model claude-opus-4-6 --permission-mode auto",
		Sessions: []SessionData{
			{
				Name:         "session-1",
				WindowName:   "demo-1",
				WorktreePath: "/tmp/worktree",
				JournalPath:  "/tmp/journal.jsonl",
				GuidancePath: "/tmp/guidance.md",
			},
		},
	}

	if err := RenderMasterProtocol(data, runDir); err != nil {
		t.Fatalf("RenderMasterProtocol: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(runDir, "master.md"))
	if err != nil {
		t.Fatalf("read rendered protocol: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"Before the first heartbeat",
		"Write an acceptance checklist",
		"/tmp/acceptance.md",
		"Read the acceptance checklist first",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered master protocol missing %q", want)
		}
	}
}
