package cli

import (
	"fmt"

	goalx "github.com/vonbai/goalx"
)

func Research(projectRoot string, args []string) error {
	if wantsHelp(args) {
		fmt.Println(launchUsage("research"))
		return nil
	}
	opts, err := parseLaunchOptions(args, goalx.ModeResearch, false)
	if err != nil {
		return err
	}
	cfg, err := buildLaunchConfig(projectRoot, opts)
	if err != nil {
		return err
	}
	_, engines, err := loadLaunchEngines(projectRoot)
	if err != nil {
		return fmt.Errorf("load base config: %w", err)
	}
	return startWithConfig(projectRoot, cfg, engines, nil)
}

func loadLaunchEngines(projectRoot string) (*goalx.Config, map[string]goalx.EngineConfig, error) {
	return goalx.LoadRawBaseConfig(projectRoot)
}
