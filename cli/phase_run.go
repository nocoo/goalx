package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	goalx "github.com/vonbai/goalx"
	"gopkg.in/yaml.v3"
)

type savedPhaseSource struct {
	Run          string
	Dir          string
	Config       *goalx.Config
	Metadata     *RunMetadata
	Context      []string
	SessionNames []string
}

func loadSavedPhaseSource(projectRoot, runName string) (*savedPhaseSource, error) {
	runName = strings.TrimSpace(runName)
	if runName == "" {
		return nil, fmt.Errorf("saved run name is required")
	}
	location, err := ResolveSavedRunLocation(projectRoot, runName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("saved run %q not found", runName)
		}
		return nil, err
	}
	runDir := location.Dir
	cfg, err := LoadSavedRunSpec(runDir)
	if err != nil {
		return nil, fmt.Errorf("load saved run %q: %w", runName, err)
	}
	contextFiles, sessionNames, err := CollectSavedResearchContext(runDir)
	if err != nil {
		return nil, fmt.Errorf("collect saved run context for %q: %w", runName, err)
	}
	meta, _ := LoadRunMetadata(filepath.Join(runDir, "run-metadata.json"))
	return &savedPhaseSource{
		Run:          runName,
		Dir:          runDir,
		Config:       cfg,
		Metadata:     meta,
		Context:      contextFiles,
		SessionNames: sessionNames,
	}, nil
}

func derivePhaseRunName(sourceRun, phaseKind string, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if sourceRun == "" {
		return phaseKind
	}
	return goalx.Slugify(sourceRun + "-" + phaseKind)
}

func phaseSourceKind(source *savedPhaseSource) string {
	if source == nil {
		return ""
	}
	if source.Metadata != nil && source.Metadata.PhaseKind != "" {
		return source.Metadata.PhaseKind
	}
	if source.Config != nil && source.Config.Mode != "" {
		return string(source.Config.Mode)
	}
	return ""
}

func phaseRunMetadataPatch(source *savedPhaseSource, phaseKind string) *RunMetadata {
	patch := &RunMetadata{PhaseKind: phaseKind}
	if source == nil {
		return patch
	}
	patch.SourceRun = source.Run
	patch.SourcePhase = phaseSourceKind(source)
	patch.ParentRun = source.Run
	if source.Metadata == nil {
		return patch
	}
	if source.Metadata.RootRunID != "" {
		patch.RootRunID = source.Metadata.RootRunID
	} else if source.Metadata.RunID != "" {
		patch.RootRunID = source.Metadata.RunID
	}
	return patch
}

func buildPhaseConfigFromSource(projectRoot string, phaseKind string, mode goalx.Mode, source *savedPhaseSource, opts phaseOptions) (*goalx.Config, map[string]goalx.EngineConfig, error) {
	if source == nil || source.Config == nil {
		return nil, nil, fmt.Errorf("saved phase source is required")
	}
	baseCfg, engines, err := goalx.LoadRawBaseConfig(projectRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("load base config: %w", err)
	}

	cfg := *source.Config
	cfg.Name = derivePhaseRunName(source.Run, phaseKind, opts.Name)
	cfg.Mode = mode
	cfg.Sessions = nil
	cfg.Context = goalx.ContextConfig{}
	cfg.Acceptance = goalx.AcceptanceConfig{}
	if opts.Preset != "" {
		cfg.Preset = opts.Preset
	}
	goalx.ApplyPreset(&cfg)
	if err := applyLaunchRoleOverrides(&cfg, launchOptions{
		Master:         opts.Master,
		ResearchRole:   opts.ResearchRole,
		DevelopRole:    opts.DevelopRole,
		Effort:         opts.Effort,
		MasterEffort:   opts.MasterEffort,
		ResearchEffort: opts.ResearchEffort,
		DevelopEffort:  opts.DevelopEffort,
	}); err != nil {
		return nil, nil, err
	}
	if opts.Parallel > 0 {
		cfg.Parallel = opts.Parallel
	}
	if cfg.Parallel < 1 {
		cfg.Parallel = source.Config.Parallel
	}
	if cfg.Parallel < 1 {
		cfg.Parallel = 1
	}
	if opts.BudgetSeconds > 0 {
		cfg.Budget.MaxDuration = time.Duration(opts.BudgetSeconds) * time.Second
	}
	if cfg.Budget.MaxDuration == 0 {
		cfg.Budget = baseCfg.Budget
	}
	return &cfg, engines, nil
}

func mergePhaseContext(base []string, extra []string) ([]string, error) {
	if len(extra) == 0 {
		return append([]string(nil), base...), nil
	}
	resolved, err := DiscoverContextFiles(extra)
	if err != nil {
		return nil, fmt.Errorf("discover context: %w", err)
	}
	merged := append([]string(nil), base...)
	seen := map[string]bool{}
	for _, path := range merged {
		seen[path] = true
	}
	for _, path := range resolved {
		if !seen[path] {
			merged = append(merged, path)
			seen[path] = true
		}
	}
	return merged, nil
}

func applyPhaseDimensions(defaultHints []string, parallel int, opts phaseOptions) ([]string, error) {
	if len(opts.Dimensions) == 0 {
		return nextConfigHints(defaultHints, parallel, nil), nil
	}
	hints, err := goalx.ResolveDimensions(opts.Dimensions)
	if err != nil {
		return nil, err
	}
	return normalizeNextConfigHints(hints, parallel), nil
}

func applySessionHints(cfg *goalx.Config, hints []string) {
	if cfg == nil {
		return
	}
	size := cfg.Parallel
	if size < len(hints) {
		size = len(hints)
	}
	if size == 0 {
		cfg.Sessions = nil
		return
	}
	cfg.Sessions = make([]goalx.SessionConfig, size)
	sessionMode := goalx.ResolveSessionMode(cfg.Mode, "")
	for i, hint := range hints {
		cfg.Sessions[i] = goalx.SessionConfig{Hint: hint, Mode: sessionMode}
	}
}

func writePhaseConfig(projectRoot string, cfg *goalx.Config, header string) error {
	goalxDir := filepath.Join(projectRoot, ".goalx")
	if err := os.MkdirAll(goalxDir, 0o755); err != nil {
		return err
	}
	outPath := ManualDraftConfigPath(projectRoot)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, append([]byte(header), data...), 0o644)
}
