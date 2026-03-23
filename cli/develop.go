package cli

import (
	"fmt"

	goalx "github.com/vonbai/goalx"
)

func Develop(projectRoot string, args []string) error {
	if wantsHelp(args) {
		fmt.Println(launchUsage("develop"))
		return nil
	}
	opts, err := parseLaunchOptions(args, goalx.ModeDevelop, false)
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
