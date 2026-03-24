package cli

import (
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestParsePhaseOptions(t *testing.T) {
	opts, err := parsePhaseOptions("debate", []string{
		"--from", "research-a",
		"--objective", "debate findings",
		"--parallel", "3",
		"--master", "codex/best",
		"--research-role", "claude-code/opus",
		"--develop-role", "codex/fast",
		"--master-effort", "high",
		"--research-effort", "medium",
		"--develop-effort", "low",
		"--dimension", "depth,adversarial",
		"--effort", "minimal",
		"--context", "README.md,docs/arch.md",
		"--budget-seconds", "900",
		"--write-config",
	})
	if err != nil {
		t.Fatalf("parsePhaseOptions: %v", err)
	}
	if opts.From != "research-a" {
		t.Fatalf("from = %q", opts.From)
	}
	if opts.Parallel != 3 {
		t.Fatalf("parallel = %d", opts.Parallel)
	}
	if opts.Master != "codex/best" {
		t.Fatalf("master = %q", opts.Master)
	}
	if opts.ResearchRole != "claude-code/opus" {
		t.Fatalf("research-role = %q", opts.ResearchRole)
	}
	if opts.DevelopRole != "codex/fast" {
		t.Fatalf("develop-role = %q", opts.DevelopRole)
	}
	if opts.Effort != goalx.EffortMinimal {
		t.Fatalf("effort = %q, want %q", opts.Effort, goalx.EffortMinimal)
	}
	if opts.MasterEffort != goalx.EffortHigh {
		t.Fatalf("master-effort = %q, want %q", opts.MasterEffort, goalx.EffortHigh)
	}
	if opts.ResearchEffort != goalx.EffortMedium {
		t.Fatalf("research-effort = %q, want %q", opts.ResearchEffort, goalx.EffortMedium)
	}
	if opts.DevelopEffort != goalx.EffortLow {
		t.Fatalf("develop-effort = %q, want %q", opts.DevelopEffort, goalx.EffortLow)
	}
	if len(opts.Dimensions) != 2 || opts.Dimensions[0] != "depth" || opts.Dimensions[1] != "adversarial" {
		t.Fatalf("dimensions = %#v, want [depth adversarial]", opts.Dimensions)
	}
	if opts.BudgetSeconds != 900 {
		t.Fatalf("budget-seconds = %d", opts.BudgetSeconds)
	}
	if !opts.WriteConfig {
		t.Fatal("write-config = false, want true")
	}
}

func TestParsePhaseOptionsRequiresFrom(t *testing.T) {
	if _, err := parsePhaseOptions("debate", nil); err == nil {
		t.Fatal("expected missing --from error")
	}
}
