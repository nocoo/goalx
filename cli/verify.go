package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Verify executes the run's acceptance checks and records the result.
// It does not detect completion, validate proof, or update state —
// the master agent reads the recorded result and decides what it means.
func Verify(projectRoot string, args []string) error {
	if printUsageIfHelp(args, "usage: goalx verify [--run NAME]") {
		return nil
	}
	runName, rest, err := extractRunFlag(args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("usage: goalx verify [--run NAME]")
	}

	rc, err := ResolveRun(projectRoot, runName)
	if err != nil {
		return err
	}

	goalState, err := LoadGoalState(GoalPath(rc.RunDir))
	if err != nil {
		return fmt.Errorf("load goal state: %w", err)
	}
	if goalState == nil {
		return fmt.Errorf("load goal state: goal state is missing")
	}
	state, err := EnsureAcceptanceState(rc.RunDir, rc.Config, goalState.Version)
	if err != nil {
		return fmt.Errorf("load acceptance state: %w", err)
	}

	activeChecks := make([]AcceptanceCheck, 0, len(state.Checks))
	for _, check := range state.Checks {
		if normalizeAcceptanceCheckState(check.State) == acceptanceCheckStateActive {
			activeChecks = append(activeChecks, check)
		}
	}
	if len(activeChecks) == 0 {
		return fmt.Errorf("no acceptance checks configured")
	}

	timeout := rc.Config.Acceptance.Timeout
	now := time.Now().UTC().Format(time.RFC3339)
	exitCode := 0
	var aggregate bytes.Buffer
	results := make([]AcceptanceCheckResult, 0, len(activeChecks))
	for _, check := range activeChecks {
		ctx := context.Background()
		cancel := func() {}
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, timeout)
		}
		cmd := exec.CommandContext(ctx, "bash", "-lc", check.Command)
		cmd.Dir = RunWorktreePath(rc.RunDir)
		if info, err := os.Stat(cmd.Dir); err != nil || !info.IsDir() {
			cmd.Dir = rc.ProjectRoot
		}
		output, runErr := cmd.CombinedOutput()
		cancel()

		checkExitCode := 0
		switch {
		case runErr == nil:
		case errors.Is(runErr, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded:
			checkExitCode = 124
		default:
			var exitErr *exec.ExitError
			if errors.As(runErr, &exitErr) {
				checkExitCode = exitErr.ExitCode()
			} else {
				checkExitCode = 1
			}
		}
		if exitCode == 0 && checkExitCode != 0 {
			exitCode = checkExitCode
		}
		evidencePath := AcceptanceCheckEvidencePath(rc.RunDir, check.ID)
		if err := os.WriteFile(evidencePath, output, 0o644); err != nil {
			return fmt.Errorf("write acceptance evidence for %s: %w", check.ID, err)
		}
		results = append(results, AcceptanceCheckResult{
			ID:           check.ID,
			Command:      check.Command,
			ExitCode:     intPtr(checkExitCode),
			EvidencePath: evidencePath,
		})
		if aggregate.Len() > 0 {
			aggregate.WriteString("\n")
		}
		aggregate.WriteString("=== ")
		aggregate.WriteString(check.ID)
		aggregate.WriteString(" ===\n")
		aggregate.Write(output)
	}

	evidencePath := AcceptanceEvidencePath(rc.RunDir)
	if err := os.WriteFile(evidencePath, aggregate.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write acceptance evidence: %w", err)
	}
	state.LastResult = AcceptanceResult{
		CheckedAt:    now,
		ExitCode:     intPtr(exitCode),
		EvidencePath: evidencePath,
		CheckResults: results,
	}
	if err := SaveAcceptanceState(AcceptanceStatePath(rc.RunDir), state); err != nil {
		return fmt.Errorf("save acceptance state: %w", err)
	}
	if err := AppendMemorySeedFromVerifyResult(rc.RunDir); err != nil {
		return fmt.Errorf("append memory seed from verify result: %w", err)
	}

	if exitCode != 0 {
		return fmt.Errorf("acceptance checks failed (%d)", exitCode)
	}

	fmt.Printf("Acceptance passed for run '%s'\n", rc.Name)
	fmt.Printf("  checks: %d\n", len(activeChecks))
	fmt.Printf("  evidence: %s\n", evidencePath)
	return nil
}

func intPtr(v int) *int {
	return &v
}
