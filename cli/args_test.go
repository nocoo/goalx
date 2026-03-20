package cli

import (
	"testing"

	ar "github.com/vonbai/autoresearch"
)

func TestExtractRunFlag(t *testing.T) {
	run, rest, err := extractRunFlag([]string{"--run", "demo", "session-2"})
	if err != nil {
		t.Fatalf("extractRunFlag: %v", err)
	}
	if run != "demo" {
		t.Fatalf("run = %q, want demo", run)
	}
	if len(rest) != 1 || rest[0] != "session-2" {
		t.Fatalf("rest = %#v, want [session-2]", rest)
	}
}

func TestExtractRunFlagMissingValue(t *testing.T) {
	_, _, err := extractRunFlag([]string{"--run"})
	if err == nil {
		t.Fatal("expected error for missing --run value")
	}
}

func TestParseStartInitArgs(t *testing.T) {
	opts, err := parseStartInitArgs([]string{
		"ship feature",
		"--research",
		"--parallel", "3",
		"--name", "demo-run",
	})
	if err != nil {
		t.Fatalf("parseStartInitArgs: %v", err)
	}
	if opts.Objective != "ship feature" {
		t.Fatalf("objective = %q", opts.Objective)
	}
	if opts.Mode != ar.ModeResearch {
		t.Fatalf("mode = %q, want %q", opts.Mode, ar.ModeResearch)
	}
	if opts.Parallel != 3 {
		t.Fatalf("parallel = %d, want 3", opts.Parallel)
	}
	if opts.Name != "demo-run" {
		t.Fatalf("name = %q, want demo-run", opts.Name)
	}
}

func TestParseStatusArgs(t *testing.T) {
	run, session, err := parseStatusArgs([]string{"--run", "demo", "session-1"})
	if err != nil {
		t.Fatalf("parseStatusArgs: %v", err)
	}
	if run != "demo" || session != "session-1" {
		t.Fatalf("got run=%q session=%q", run, session)
	}
}

func TestSessionCountPrefersExplicitSessions(t *testing.T) {
	cfg := &ar.Config{
		Parallel: 1,
		Sessions: []ar.SessionConfig{{Hint: "a"}, {Hint: "b"}},
	}
	if got := sessionCount(cfg); got != 2 {
		t.Fatalf("sessionCount = %d, want 2", got)
	}
}
