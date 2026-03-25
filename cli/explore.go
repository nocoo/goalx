package cli

import (
	"fmt"

	goalx "github.com/vonbai/goalx"
)

// Explore starts a follow-up research run from an explicit saved run.
func Explore(projectRoot string, args []string) error {
	if wantsHelp(args) {
		fmt.Println(phaseUsage("explore"))
		return nil
	}
	opts, err := parsePhaseOptions("explore", args)
	if err != nil {
		return err
	}
	source, err := loadSavedPhaseSource(projectRoot, opts.From)
	if err != nil {
		return err
	}
	if len(source.Context) == 0 {
		return fmt.Errorf("no reports found in %s", source.Dir)
	}

	cfg, engines, err := resolvePhaseConfig(projectRoot, "explore", goalx.ModeResearch, source, opts)
	if err != nil {
		return err
	}
	contextFiles, err := phaseContextFiles(cfg, source, opts.ContextPaths)
	if err != nil {
		return err
	}
	defaultHints := []string{
		"继续扩大证据覆盖范围，优先验证原结论的盲点、缺失案例和失败模式。",
		"从替代架构路径、反例和更高 ROI 方案入手，补充可派发的新切片。",
	}
	hints, err := applyPhaseDimensions(defaultHints, cfg.Parallel, opts)
	if err != nil {
		return err
	}
	applySessionHints(cfg, hints)
	cfg.Context = goalx.ContextConfig{Files: contextFiles, Refs: cfg.Context.Refs}

	if opts.WriteConfig {
		if err := writePhaseConfig(projectRoot, cfg, fmt.Sprintf("# goalx manual draft — explore based on %s\n", source.Run)); err != nil {
			return err
		}
		fmt.Printf("Generated manual draft %s (explore from %s)\n", ManualDraftConfigPath(projectRoot), source.Run)
		fmt.Println("\n  Next: review .goalx/goalx.yaml, then goalx start --config .goalx/goalx.yaml")
		return nil
	}

	return startWithConfig(projectRoot, cfg, engines, phaseRunMetadataPatch(source, "explore"), false)
}
