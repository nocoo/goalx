package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	goalx "github.com/vonbai/goalx"
)

type selectionJSON struct {
	Kept   string `json:"kept"`
	Branch string `json:"branch"`
}

type resultRun struct {
	Name   string
	Dir    string
	Config *goalx.Config
	Saved  bool
}

// Result prints the canonical run-level result surface when present, then falls
// back to supporting reports or kept-branch metadata for older runs.
func Result(projectRoot string, args []string) error {
	if printUsageIfHelp(args, "usage: goalx result [NAME] [--full]") {
		return nil
	}
	runName, rest, err := extractRunFlag(args)
	if err != nil {
		return err
	}
	full := false
	var positional []string
	for _, arg := range rest {
		if arg == "--full" {
			full = true
			continue
		}
		positional = append(positional, arg)
	}
	if runName == "" && len(positional) == 1 {
		runName = positional[0]
		positional = nil
	}
	if len(positional) > 0 {
		return fmt.Errorf("usage: goalx result [NAME] [--full]")
	}

	target, err := resolveResultRun(projectRoot, runName)
	if err != nil {
		return err
	}

	if data, err := loadResultSurface(target.Dir); err == nil {
		if full {
			fmt.Print(string(data))
			return nil
		}
		printRunResult(data)
		return nil
	}

	selection, err := loadResultSelection(projectRoot, target.Dir, target.Config.Name)
	if err != nil {
		return err
	}

	fmt.Printf("Kept: %s\n", selection.Kept)
	logOut, err := exec.Command("git", "-C", projectRoot, "log", "--oneline", "-5", selection.Branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git log %s: %w: %s", selection.Branch, err, logOut)
	}
	fmt.Print(string(logOut))

	diffOut, err := exec.Command("git", "-C", projectRoot, "show", "--stat", "--format=", selection.Branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git show %s: %w: %s", selection.Branch, err, diffOut)
	}
	fmt.Print(string(diffOut))
	return nil
}

func resolveResultRun(projectRoot, runName string) (*resultRun, error) {
	location, err := ResolveSavedRunLocation(projectRoot, runName)
	if err == nil {
		cfg, loadErr := LoadSavedRunSpec(location.Dir)
		if loadErr != nil {
			return nil, fmt.Errorf("load saved config: %w", loadErr)
		}
		return &resultRun{
			Name:   cfg.Name,
			Dir:    location.Dir,
			Config: cfg,
			Saved:  true,
		}, nil
	}

	var multipleErr MultipleSavedRunsError
	switch {
	case errors.As(err, &multipleErr):
		return nil, fmt.Errorf("%s (specify NAME)", multipleErr.Error())
	case !errors.Is(err, os.ErrNotExist):
		return nil, err
	}

	rc, activeErr := ResolveRun(projectRoot, runName)
	if activeErr == nil {
		return &resultRun{
			Name:   rc.Name,
			Dir:    rc.RunDir,
			Config: rc.Config,
			Saved:  false,
		}, nil
	}

	if strings.TrimSpace(runName) != "" {
		return nil, fmt.Errorf("saved run %q not found", runName)
	}
	return nil, fmt.Errorf("no saved runs found")
}

func loadResultSurface(runDir string) ([]byte, error) {
	data, err := os.ReadFile(SummaryPath(runDir))
	if err == nil && len(data) > 0 {
		return data, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read summary: %w", err)
	}
	return loadResultFallback(runDir)
}

func loadResultFallback(runDir string) ([]byte, error) {
	contextFiles, _, err := CollectSavedResearchContext(runDir)
	if err != nil {
		contextFiles = nil
	}
	for _, path := range contextFiles {
		if path == SummaryPath(runDir) || filepath.Base(path) == "summary.md" {
			continue
		}
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			return data, nil
		}
	}
	reportFiles, err := loadRunScopedReportFiles(runDir)
	if err == nil {
		for _, data := range reportFiles {
			if len(data) > 0 {
				return data, nil
			}
		}
	}
	return nil, fmt.Errorf("no saved research report found in %s", runDir)
}

func loadRunScopedReportFiles(runDir string) ([][]byte, error) {
	entries, err := os.ReadDir(ReportsDir(runDir))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	out := make([][]byte, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(ReportsDir(runDir), name))
		if err == nil && len(data) > 0 {
			out = append(out, data)
		}
	}
	return out, nil
}

func parseSections(data []byte) map[string]string {
	sections := make(map[string]string)
	var current string
	var body []string

	flush := func() {
		if current == "" {
			return
		}
		sections[current] = strings.TrimSpace(strings.Join(body, "\n"))
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			body = body[:0]
			continue
		}
		if current != "" {
			body = append(body, line)
		}
	}
	flush()
	return sections
}

func printRunResult(data []byte) {
	fmt.Println("=== Result ===")
	fmt.Print(renderResultSummary(data))
	fmt.Println()
	fmt.Println()
	fmt.Println("Full report: goalx result --full")
}

func renderResultSummary(data []byte) string {
	sections := parseSections(data)
	var parts []string

	if recommendation := firstNonEmptyLine(sections["Recommendation"]); recommendation != "" {
		parts = append(parts, "Recommendation: "+recommendation)
	}

	if findings := summarizeSectionLines(sections["Key Findings"], 5); findings != "" {
		parts = append(parts, "Key Findings:\n"+findings)
	}

	if fixes := strings.TrimSpace(sections["Priority Fix List"]); fixes != "" {
		parts = append(parts, "Priority Fix List:\n"+fixes)
	}

	if len(parts) == 0 {
		return strings.TrimSpace(string(data))
	}
	return strings.Join(parts, "\n\n")
}

func firstNonEmptyLine(section string) string {
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func summarizeSectionLines(section string, limit int) string {
	if limit < 1 {
		return ""
	}

	var lines []string
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}
	if len(lines) <= limit {
		return strings.Join(lines, "\n")
	}
	return strings.Join(append(lines[:limit], fmt.Sprintf("... (%d more lines)", len(lines)-limit)), "\n")
}

func loadResultSelection(projectRoot, savedRunDir, runName string) (*selectionJSON, error) {
	for _, path := range []string{
		filepath.Join(savedRunDir, "selection.json"),
		filepath.Join(goalx.RunDir(projectRoot, runName), "selection.json"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var selection selectionJSON
		if err := json.Unmarshal(data, &selection); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if selection.Kept == "" || selection.Branch == "" {
			return nil, fmt.Errorf("selection in %s is incomplete", path)
		}
		return &selection, nil
	}

	return nil, fmt.Errorf("selection.json not found for develop run %q", runName)
}
