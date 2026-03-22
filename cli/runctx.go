package cli

import (
	"fmt"

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

// ResolveRun resolves run context. If runName is empty, it resolves the
// focused/only active run from the project registry.
func ResolveRun(projectRoot, runName string) (*RunContext, error) {
	if runName == "" {
		var err error
		runName, err = ResolveDefaultRunName(projectRoot)
		if err != nil {
			return nil, err
		}
	}

	runDir := goalx.RunDir(projectRoot, runName)
	snapshot, err := LoadRunSpec(runDir)
	if err != nil {
		return nil, fmt.Errorf("load run spec: %w", err)
	}

	return &RunContext{
		Name:        runName,
		RunDir:      runDir,
		TmuxSession: goalx.TmuxSessionName(projectRoot, runName),
		ProjectRoot: projectRoot,
		Config:      snapshot,
	}, nil
}
