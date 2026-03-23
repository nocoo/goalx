package cli

import "testing"

func TestParsePhaseOptions(t *testing.T) {
	opts, err := parsePhaseOptions("debate", []string{
		"--from", "research-a",
		"--objective", "debate findings",
		"--parallel", "3",
		"--master", "codex/best",
		"--research-role", "claude-code/opus",
		"--develop-role", "codex/fast",
		"--strategy", "depth,adversarial",
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
