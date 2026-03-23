package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	goalx "github.com/vonbai/goalx"
)

// Next detects the current pipeline state and suggests the next action.
func Next(projectRoot string, args []string) error {
	if hasHelpArg(args) {
		fmt.Println("usage: goalx next")
		return nil
	}
	runsDir := ProjectDataDir(projectRoot)
	reg, _ := LoadProjectRegistry(projectRoot)
	focusedRun := ""
	if reg != nil && reg.FocusedRun != "" {
		if _, ok := reg.ActiveRuns[reg.FocusedRun]; ok {
			focusedRun = reg.FocusedRun
		}
	}

	// Check for active runs
	activeRuns := findActiveRuns(reg, projectRoot, runsDir)
	if len(activeRuns) == 1 {
		fmt.Printf("Active run: %s\n", activeRuns[0])
		fmt.Printf("  → goalx attach --run %s\n", activeRuns[0])
		return nil
	}
	if len(activeRuns) > 1 {
		fmt.Printf("Active runs: %s\n", strings.Join(activeRuns, ", "))
		if focusedRun != "" {
			fmt.Printf("Focused run: %s\n", focusedRun)
		}
		fmt.Println("  → goalx focus --run NAME   # choose the default run")
		fmt.Println("  → goalx list")
		fmt.Println("  → goalx attach --run NAME")
		return nil
	}

	// Check for completed (not yet saved) runs
	completedRuns := findCompletedRuns(reg, projectRoot, runsDir)
	if len(completedRuns) == 1 {
		completedRun := completedRuns[0]
		fmt.Printf("Completed run: %s (not yet saved)\n", completedRun)
		fmt.Printf("  → goalx save %s    # save artifacts to user-scoped durable storage\n", completedRun)
		fmt.Printf("  → goalx review %s  # inspect results\n", completedRun)
		fmt.Printf("  → goalx drop %s    # clean up worktrees\n", completedRun)
		return nil
	}
	if len(completedRuns) > 1 {
		fmt.Printf("Completed unsaved runs: %s\n", strings.Join(completedRuns, ", "))
		fmt.Println("  → goalx save NAME")
		fmt.Println("  → goalx review --run NAME")
		fmt.Println("  → goalx drop --run NAME")
		return nil
	}

	// Check saved runs in durable storage, preferring user scope and falling back to legacy project scope.
	hasSaves := false
	latestDebate := ""
	latestResearch := ""
	latestAny := ""
	latestDebateTime := int64(0)
	latestResearchTime := int64(0)
	latestAnyTime := int64(0)

	if locations, err := ListSavedRunLocations(projectRoot); err == nil {
		for _, loc := range locations {
			hasSaves = true
			cfg, err := LoadSavedRunSpec(loc.Dir)
			if err != nil {
				continue
			}
			info, err := os.Stat(loc.Dir)
			if err != nil {
				continue
			}
			modTime := info.ModTime().Unix()
			if modTime >= latestAnyTime {
				latestAnyTime = modTime
				latestAny = loc.Name
			}
			meta, _ := LoadRunMetadata(filepath.Join(loc.Dir, "run-metadata.json"))
			phaseKind := ""
			if meta != nil {
				phaseKind = meta.PhaseKind
			}
			if phaseKind == "debate" && modTime >= latestDebateTime {
				latestDebateTime = modTime
				latestDebate = loc.Name
			}
			if cfg.Mode == goalx.ModeResearch && modTime >= latestResearchTime {
				latestResearchTime = modTime
				latestResearch = loc.Name
			}
		}
	}

	if latestDebate != "" {
		fmt.Printf("Debate completed: %s\n", latestDebate)
		fmt.Printf("  → goalx implement --from %s\n", latestDebate)
		fmt.Printf("  → goalx explore --from %s    # extend debate findings if needed\n", latestDebate)
		return nil
	}

	if latestResearch != "" {
		fmt.Printf("Research completed: %s\n", latestResearch)
		fmt.Printf("  → goalx debate --from %s\n", latestResearch)
		fmt.Printf("  → goalx implement --from %s\n", latestResearch)
		fmt.Println()
		fmt.Printf("  Or continue exploration:\n  → goalx explore --from %s\n", latestResearch)
		return nil
	}

	if hasSaves {
		fmt.Println("Saved runs exist but no clear next step detected.")
		fmt.Println("  → goalx list        # see all runs")
		if latestAny != "" {
			fmt.Printf("  → goalx result %s\n", latestAny)
		}
		fmt.Println("  → goalx auto \"...\"  # start a new autonomous run")
		return nil
	}

	// Nothing exists
	fmt.Println("No runs or saved results found.")
	fmt.Println()
	fmt.Println("Quickstart:")
	fmt.Println("  goalx auto \"your objective\"")
	return nil
}

func findActiveRuns(reg *ProjectRegistry, projectRoot, runsDir string) []string {
	if reg != nil && len(reg.ActiveRuns) > 0 {
		return sortedRunNames(reg.ActiveRuns)
	}
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil
	}
	var active []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "saved" {
			continue
		}
		tmuxSess := goalx.TmuxSessionName(projectRoot, e.Name())
		if SessionExists(tmuxSess) {
			active = append(active, e.Name())
		}
	}
	sort.Strings(active)
	return active
}

func findCompletedRuns(reg *ProjectRegistry, projectRoot, runsDir string) []string {
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil
	}
	active := map[string]struct{}{}
	if reg != nil {
		for name := range reg.ActiveRuns {
			active[name] = struct{}{}
		}
	}
	var completed []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "saved" {
			continue
		}
		name := e.Name()
		if _, ok := active[name]; ok {
			continue
		}
		tmuxSess := goalx.TmuxSessionName(projectRoot, name)
		if !SessionExists(tmuxSess) {
			completed = append(completed, name)
		}
	}
	sort.Strings(completed)
	return completed
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
