package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	goalx "github.com/vonbai/goalx"
)

// relaunchMaster recreates the master window from durable run state.
func relaunchMaster(projectRoot, runDir, tmuxSession string, cfg *goalx.Config) error {
	if cfg == nil {
		return fmt.Errorf("run config is nil")
	}

	engines, err := loadEngineCatalog(projectRoot)
	if err != nil {
		return fmt.Errorf("load config for engine resolution: %w", err)
	}
	spec, err := goalx.ResolveLaunchSpec(engines, goalx.LaunchRequest{
		Engine: cfg.Master.Engine,
		Model:  cfg.Master.Model,
		Effort: cfg.Master.Effort,
	})
	if err != nil {
		return fmt.Errorf("resolve engine: %w", err)
	}
	engineCmd := spec.Command
	protocolPath := filepath.Join(runDir, "master.md")
	prompt := goalx.ResolvePrompt(engines, cfg.Master.Engine, protocolPath)

	meta, err := EnsureRunMetadata(runDir, projectRoot, cfg.Objective)
	if err != nil {
		return fmt.Errorf("load run metadata: %w", err)
	}
	if err := ensureEvolutionSurface(runDir, meta); err != nil {
		return fmt.Errorf("init evolution surface: %w", err)
	}
	masterData, err := buildMasterProtocolData(projectRoot, runDir, tmuxSession, cfg, engines, engineCmd, meta)
	if err != nil {
		return fmt.Errorf("build master protocol data: %w", err)
	}
	if err := RenderMasterProtocol(masterData, runDir); err != nil {
		return fmt.Errorf("render master protocol: %w", err)
	}
	goalxBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve goalx executable: %w", err)
	}
	checkSec, _ := normalizeSidecarInterval(cfg.Master.CheckInterval)
	masterLeaseTTL := time.Duration(checkSec) * time.Second * 2
	workdir := RunWorktreePath(runDir)
	launchCmd := buildMasterLaunchCommand(goalxBin, cfg.Name, runDir, meta.RunID, meta.Epoch, masterLeaseTTL, engineCmd, prompt)

	if !SessionExists(tmuxSession) {
		if err := NewSessionWithCommand(tmuxSession, "master", workdir, launchCmd); err != nil {
			return fmt.Errorf("create master session: %w", err)
		}
		return nil
	}
	_ = KillWindow(tmuxSession, "master")
	if err := NewWindowWithCommand(tmuxSession, "master", workdir, launchCmd); err != nil {
		return fmt.Errorf("create master window: %w", err)
	}
	return nil
}
