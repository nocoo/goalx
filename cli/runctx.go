package cli

import (
	"fmt"
	"path/filepath"

	goalx "github.com/vonbai/goalx"
)

// RunContext holds resolved paths for a run.
type RunContext struct {
	Name        string
	RunDir      string
	TmuxSession string
	ProjectRoot string
	Config      *goalx.Config
}

// ResolveRun resolves run context. If runName is empty, reads goalx.yaml from
// projectRoot to get the name.
func ResolveRun(projectRoot, runName string) (*RunContext, error) {
	if runName == "" {
		cfg, err := goalx.LoadYAML[goalx.Config](filepath.Join(projectRoot, ".goalx", "goalx.yaml"))
		if err != nil {
			return nil, fmt.Errorf("load goalx.yaml: %w", err)
		}
		if cfg.Name == "" {
			return nil, fmt.Errorf("no run name: set name in goalx.yaml or pass --name")
		}
		runName = cfg.Name
	}

	runDir := goalx.RunDir(projectRoot, runName)
	snapshot, err := goalx.LoadYAML[goalx.Config](filepath.Join(runDir, "goalx.yaml"))
	if err != nil {
		return nil, fmt.Errorf("load run snapshot: %w", err)
	}
	if snapshot.Name == "" {
		return nil, fmt.Errorf("run %q not found (no snapshot at %s)", runName, runDir)
	}

	return &RunContext{
		Name:        runName,
		RunDir:      runDir,
		TmuxSession: goalx.TmuxSessionName(projectRoot, runName),
		ProjectRoot: projectRoot,
		Config:      &snapshot,
	}, nil
}
