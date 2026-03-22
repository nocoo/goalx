package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestTellWritesSessionGuidanceStateAndNudges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")
	runName, runDir := writeLifecycleRunFixture(t, repo)

	orig := sendAgentNudge
	defer func() { sendAgentNudge = orig }()

	var gotTarget, gotEngine string
	sendAgentNudge = func(target, engine string) error {
		gotTarget, gotEngine = target, engine
		return nil
	}

	out := captureStdout(t, func() {
		if err := Tell(repo, []string{"--run", runName, "session-1", "focus on db race triage"}); err != nil {
			t.Fatalf("Tell: %v", err)
		}
	})

	if !strings.Contains(out, "session-1") {
		t.Fatalf("tell output = %q, want session target", out)
	}

	guidanceData, err := os.ReadFile(GuidancePath(runDir, "session-1"))
	if err != nil {
		t.Fatalf("read guidance: %v", err)
	}
	if string(guidanceData) != "focus on db race triage\n" {
		t.Fatalf("guidance = %q", string(guidanceData))
	}

	state, err := LoadSessionGuidanceState(SessionGuidanceStatePath(runDir, "session-1"))
	if err != nil {
		t.Fatalf("LoadSessionGuidanceState: %v", err)
	}
	if state == nil || state.Version != 1 || !state.Pending {
		t.Fatalf("guidance state = %#v, want pending version 1", state)
	}

	wantTarget := goalx.TmuxSessionName(repo, runName) + ":" + sessionWindowName(runName, 1)
	if gotTarget != wantTarget || gotEngine != "codex" {
		t.Fatalf("sendAgentNudge target=%q engine=%q, want %q codex", gotTarget, gotEngine, wantTarget)
	}
}

func TestTellHelpDoesNotDeliverAnything(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")
	runName, runDir := writeLifecycleRunFixture(t, repo)

	orig := sendAgentNudge
	defer func() { sendAgentNudge = orig }()

	called := false
	sendAgentNudge = func(target, engine string) error {
		called = true
		return nil
	}

	out := captureStdout(t, func() {
		if err := Tell(repo, []string{"--run", runName, "--help"}); err != nil {
			t.Fatalf("Tell --help: %v", err)
		}
	})

	if !strings.Contains(out, tellUsage) {
		t.Fatalf("tell help output = %q, want %q", out, tellUsage)
	}
	if called {
		t.Fatal("Tell --help should not nudge any target")
	}
	if _, err := os.Stat(SessionGuidanceStatePath(runDir, "session-1")); !os.IsNotExist(err) {
		t.Fatalf("guidance state should not be created by help, stat err = %v", err)
	}
}

func TestAckGuidanceMarksLatestVersionConsumed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")
	runName, runDir := writeLifecycleRunFixture(t, repo)

	if err := WriteSessionGuidance(runDir, "session-1", "focus on db race triage"); err != nil {
		t.Fatalf("WriteSessionGuidance: %v", err)
	}

	if err := AckGuidance(repo, []string{"--run", runName, "session-1"}); err != nil {
		t.Fatalf("AckGuidance: %v", err)
	}

	state, err := LoadSessionGuidanceState(SessionGuidanceStatePath(runDir, "session-1"))
	if err != nil {
		t.Fatalf("LoadSessionGuidanceState: %v", err)
	}
	if state == nil {
		t.Fatal("guidance state missing")
	}
	if state.Pending {
		t.Fatalf("guidance state pending = true, want false")
	}
	if state.LastAckVersion != state.Version {
		t.Fatalf("last ack version = %d, want %d", state.LastAckVersion, state.Version)
	}
	if state.LastAckAt == "" {
		t.Fatalf("last ack at empty")
	}
}

func TestStatusShowsGuidancePendingForSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")
	runName, runDir := writeLifecycleRunFixture(t, repo)

	if err := os.WriteFile(JournalPath(runDir, "session-1"), []byte(`{"round":2,"desc":"awaiting master","status":"idle"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write session journal: %v", err)
	}
	if err := WriteSessionGuidance(runDir, "session-1", "focus on db race triage"); err != nil {
		t.Fatalf("WriteSessionGuidance: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Status(repo, []string{"--run", runName}); err != nil {
			t.Fatalf("Status: %v", err)
		}
	})

	if !strings.Contains(out, "guidance-pending") {
		t.Fatalf("status output missing guidance-pending:\n%s", out)
	}
}

func TestRenderSubagentProtocolIncludesGuidanceStateAckPath(t *testing.T) {
	runDir := t.TempDir()
	data := ProtocolData{
		RunName:           "demo",
		Objective:         "ship it",
		Mode:              goalx.ModeDevelop,
		ProjectRoot:       "/tmp/project",
		SessionName:       "session-1",
		GuidancePath:      "/tmp/guidance/session-1.md",
		GuidanceStatePath: "/tmp/control/session-1-guidance.json",
		JournalPath:       "/tmp/journals/session-1.jsonl",
		Target:            goalx.TargetConfig{Files: []string{"."}},
		Harness:           goalx.HarnessConfig{Command: "go test ./..."},
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
		"/tmp/control/session-1-guidance.json",
		"goalx ack-guidance --run demo session-1",
		"cd /tmp/project && goalx ack-guidance --run demo session-1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered subagent protocol missing %q:\n%s", want, text)
		}
	}
}
