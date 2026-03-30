package cli

import (
	"fmt"
	"strings"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestRunStartsDeliverPathByDefault(t *testing.T) {
	oldAuto := runAutoWithOptions
	defer func() { runAutoWithOptions = oldAuto }()

	calls := 0
	runAutoWithOptions = func(projectRoot string, opts launchOptions) error {
		calls++
		if projectRoot == "" {
			t.Fatal("projectRoot should not be empty")
		}
		if opts.Objective != "ship it" {
			t.Fatalf("objective = %q, want ship it", opts.Objective)
		}
		if opts.Mode != goalx.ModeAuto {
			t.Fatalf("mode = %q, want %q", opts.Mode, goalx.ModeAuto)
		}
		return nil
	}

	out := captureStdout(t, func() {
		if err := Run(t.TempDir(), []string{"ship it"}); err != nil {
			t.Fatalf("Run: %v", err)
		}
	})

	if calls != 1 {
		t.Fatalf("auto calls = %d, want 1", calls)
	}
	for _, want := range []string{"Run started.", "goalx status", "goalx observe", "goalx attach"} {
		if !strings.Contains(out, want) {
			t.Fatalf("run output missing %q:\n%s", want, out)
		}
	}
}

func TestRunIntentEvolveUsesAutoLaunchMode(t *testing.T) {
	oldAuto := runAutoWithOptions
	defer func() { runAutoWithOptions = oldAuto }()

	calls := 0
	runAutoWithOptions = func(projectRoot string, opts launchOptions) error {
		calls++
		if projectRoot == "" {
			t.Fatal("projectRoot should not be empty")
		}
		if opts.Objective != "ship auth" {
			t.Fatalf("objective = %q, want ship auth", opts.Objective)
		}
		if opts.Mode != goalx.ModeAuto {
			t.Fatalf("mode = %q, want %q", opts.Mode, goalx.ModeAuto)
		}
		if opts.Intent != runIntentEvolve {
			t.Fatalf("intent = %q, want %q", opts.Intent, runIntentEvolve)
		}
		return nil
	}

	out := captureStdout(t, func() {
		if err := Run(t.TempDir(), []string{"ship auth", "--intent", "evolve"}); err != nil {
			t.Fatalf("Run: %v", err)
		}
	})

	if calls != 1 {
		t.Fatalf("auto calls = %d, want 1", calls)
	}
	if !strings.Contains(out, "Run started.") {
		t.Fatalf("run output missing start summary:\n%s", out)
	}
}

func TestRunIntentDebateUsesPhasePath(t *testing.T) {
	oldDebate := runDebateIntent
	defer func() { runDebateIntent = oldDebate }()

	calls := 0
	runDebateIntent = func(projectRoot string, args []string) error {
		calls++
		if projectRoot == "" {
			t.Fatal("projectRoot should not be empty")
		}
		want := []string{"--from", "research-a", "--write-config"}
		if len(args) != len(want) {
			t.Fatalf("args = %v, want %v", args, want)
		}
		for i := range want {
			if args[i] != want[i] {
				t.Fatalf("args = %v, want %v", args, want)
			}
		}
		return nil
	}

	if err := Run(t.TempDir(), []string{"--from", "research-a", "--intent", "debate", "--write-config"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if calls != 1 {
		t.Fatalf("debate calls = %d, want 1", calls)
	}
}

func TestRunRejectsUnknownIntent(t *testing.T) {
	err := Run(t.TempDir(), []string{"ship it", "--intent", "mystery"})
	if err == nil || !strings.Contains(err.Error(), `unknown --intent "mystery"`) {
		t.Fatalf("Run error = %v, want unknown intent", err)
	}
}

func TestRunRejectsRemovedResearchAndDevelopIntents(t *testing.T) {
	for _, intent := range []string{"research", "develop"} {
		err := Run(t.TempDir(), []string{"ship it", "--intent", intent})
		if err == nil || !strings.Contains(err.Error(), fmt.Sprintf(`unknown --intent %q`, intent)) {
			t.Fatalf("Run(%q) error = %v, want unknown intent", intent, err)
		}
	}
}

func TestRunHelpPrintsUsage(t *testing.T) {
	out := captureStdout(t, func() {
		if err := Run(t.TempDir(), []string{"--help"}); err != nil {
			t.Fatalf("Run --help: %v", err)
		}
	})
	if !strings.Contains(out, "usage: goalx run") {
		t.Fatalf("run help missing usage:\n%s", out)
	}
	for _, unwanted := range []string{"research", "develop"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("run help should omit removed intent %q:\n%s", unwanted, out)
		}
	}
	if strings.Contains(out, "legacy command names remain temporary aliases") {
		t.Fatalf("run help should not mention legacy aliases:\n%s", out)
	}
}

func TestDebateRoutesThroughRunEntrypoint(t *testing.T) {
	oldRun := runEntrypoint
	defer func() { runEntrypoint = oldRun }()

	runEntrypoint = func(_ string, args []string) error {
		want := []string{"--intent", runIntentDebate, "--from", "research-a"}
		if len(args) != len(want) {
			t.Fatalf("args = %v, want %v", args, want)
		}
		for i := range want {
			if args[i] != want[i] {
				t.Fatalf("args = %v, want %v", args, want)
			}
		}
		return nil
	}

	if err := Debate(t.TempDir(), []string{"--from", "research-a"}); err != nil {
		t.Fatalf("Debate: %v", err)
	}
}
