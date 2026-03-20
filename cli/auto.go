package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// statusJSON matches the structure master writes to .goalx/status.json
type statusJSON struct {
	Phase          string `json:"phase"`
	Recommendation string `json:"recommendation"`
	Heartbeat      int    `json:"heartbeat"`
}

// Auto runs the full goalx pipeline: research -> (debate?) -> implement -> keep.
func Auto(projectRoot string, args []string) error {
	// 1. Init
	if err := Init(projectRoot, args); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	// 2. Start
	if err := Start(projectRoot, nil); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// 3. Poll until complete
	fmt.Println("Waiting for run to complete...")
	statusPath := filepath.Join(projectRoot, ".goalx", "status.json")
	status, err := pollUntilComplete(statusPath, 30*time.Second, 4*time.Hour)
	if err != nil {
		return fmt.Errorf("poll: %w", err)
	}

	// 4. Save
	if err := Save(projectRoot, nil); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	// 5. Drop the completed run
	if err := Drop(projectRoot, nil); err != nil {
		fmt.Fprintf(os.Stderr, "warning: drop failed: %v\n", err)
	}

	// 6. Route based on recommendation
	rec := status.Recommendation
	fmt.Printf("Master recommendation: %s\n", rec)

	switch rec {
	case "debate":
		fmt.Println("Starting debate round...")
		if err := Debate(projectRoot, nil); err != nil {
			return fmt.Errorf("debate: %w", err)
		}
		if err := Start(projectRoot, nil); err != nil {
			return fmt.Errorf("start debate: %w", err)
		}
		if _, err := pollUntilComplete(statusPath, 30*time.Second, 4*time.Hour); err != nil {
			return fmt.Errorf("poll debate: %w", err)
		}
		if err := Save(projectRoot, nil); err != nil {
			return fmt.Errorf("save debate: %w", err)
		}
		if err := Drop(projectRoot, nil); err != nil {
			fmt.Fprintf(os.Stderr, "warning: drop failed: %v\n", err)
		}
		// After debate, implement
		fmt.Println("Starting implementation...")
		if err := Implement(projectRoot, nil); err != nil {
			return fmt.Errorf("implement: %w", err)
		}
		if err := Start(projectRoot, nil); err != nil {
			return fmt.Errorf("start implement: %w", err)
		}
		if _, err := pollUntilComplete(statusPath, 30*time.Second, 4*time.Hour); err != nil {
			return fmt.Errorf("poll implement: %w", err)
		}
		fmt.Println("Implementation complete. Run: goalx review -> goalx keep")

	case "implement":
		fmt.Println("Skipping debate -- findings are consistent. Starting implementation...")
		if err := Implement(projectRoot, nil); err != nil {
			return fmt.Errorf("implement: %w", err)
		}
		if err := Start(projectRoot, nil); err != nil {
			return fmt.Errorf("start implement: %w", err)
		}
		if _, err := pollUntilComplete(statusPath, 30*time.Second, 4*time.Hour); err != nil {
			return fmt.Errorf("poll implement: %w", err)
		}
		fmt.Println("Implementation complete. Run: goalx review -> goalx keep")

	case "done":
		fmt.Println("Research objective achieved. No code changes needed.")
		fmt.Println("Results saved to .goalx/runs/")

	case "more-research":
		fmt.Println("Master recommends more research. Review the summary and add new directions.")
		fmt.Println("Run: goalx observe or goalx review to see findings so far.")

	default:
		fmt.Printf("Unknown recommendation %q. Review results manually.\n", rec)
		fmt.Println("Run: goalx review -> goalx next")
	}

	return nil
}

// pollUntilComplete reads status.json every interval until phase=complete or timeout.
func pollUntilComplete(statusPath string, interval, timeout time.Duration) (*statusJSON, error) {
	deadline := time.Now().Add(timeout)
	lastHB := -1

	for time.Now().Before(deadline) {
		data, err := os.ReadFile(statusPath)
		if err == nil && len(data) > 0 {
			var s statusJSON
			if json.Unmarshal(data, &s) == nil {
				if s.Heartbeat != lastHB {
					fmt.Printf("  heartbeat %d -- phase: %s\n", s.Heartbeat, s.Phase)
					lastHB = s.Heartbeat
				}
				if s.Phase == "complete" {
					return &s, nil
				}
			}
		}
		time.Sleep(interval)
	}
	return nil, fmt.Errorf("timeout after %v waiting for completion", timeout)
}
