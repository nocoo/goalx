package cli

import (
	"path/filepath"

	goalx "github.com/vonbai/goalx"
)

func SharedProjectConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".goalx", "config.yaml")
}

func ManualDraftConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".goalx", "goalx.yaml")
}

func LoadManualDraftConfig(projectRoot, draftPath string) (*goalx.Config, map[string]goalx.EngineConfig, error) {
	if draftPath == "" {
		draftPath = ManualDraftConfigPath(projectRoot)
	}
	return goalx.LoadConfigWithManualDraft(projectRoot, draftPath)
}
