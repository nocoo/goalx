package cli

import (
	"fmt"

	goalx "github.com/vonbai/goalx"
)

// Implement starts a develop-mode implementation run from an explicit saved run.
func Implement(projectRoot string, args []string, nc *nextConfigJSON) error {
	if wantsHelp(args) {
		fmt.Println(phaseUsage("implement"))
		return nil
	}
	opts, err := parsePhaseOptions("implement", args)
	if err != nil {
		return err
	}
	opts = mergeNextConfigIntoPhaseOptions(opts, nc, goalx.ModeDevelop)
	source, err := loadSavedPhaseSource(projectRoot, opts.From)
	if err != nil {
		return err
	}
	if len(source.Context) == 0 {
		return fmt.Errorf("no reports/summary found in %s", source.Dir)
	}

	cfg, engines, err := resolvePhaseConfig(projectRoot, "implement", goalx.ModeDevelop, source, opts)
	if err != nil {
		return err
	}
	contextFiles, err := phaseContextFiles(cfg, source, opts.ContextPaths)
	if err != nil {
		return err
	}
	defaultHints := []string{
		"你负责优先级最高的修复项（P0 + P1 中不依赖其他文件的项）。逐个修复，每个修完跑一次 gate 验证。",
		"你负责剩余修复项（P2 + 重构类 P1）。先做独立的删除/清理，再做涉及多文件的重构。每步跑 gate。",
	}
	hints, err := applyPhaseDimensions(defaultHints, cfg.Parallel, opts)
	if err != nil {
		return err
	}

	applySessionHints(cfg, hints)
	cfg.Context = goalx.ContextConfig{Files: contextFiles, Refs: cfg.Context.Refs}

	if opts.WriteConfig {
		if err := writePhaseConfig(projectRoot, cfg, fmt.Sprintf("# goalx manual draft — implement fixes from %s\n", source.Run)); err != nil {
			return err
		}
		fmt.Printf("Generated manual draft %s (implement from %s)\n", ManualDraftConfigPath(projectRoot), source.Run)
		fmt.Println("\n  Next: review .goalx/goalx.yaml, then goalx start --config .goalx/goalx.yaml")
		return nil
	}

	return startWithConfig(projectRoot, cfg, engines, phaseRunMetadataPatch(source, "implement"), false)
}
