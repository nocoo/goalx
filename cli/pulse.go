package cli

import (
	"fmt"
	"time"
)

// Pulse records a durable heartbeat tick, then nudges the master to read control files.
func Pulse(projectRoot string, args []string) error {
	runName, rest, err := extractRunFlag(args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return fmt.Errorf("usage: goalx pulse [--run NAME]")
	}

	rc, err := ResolveRun(projectRoot, runName)
	if err != nil {
		return err
	}
	if _, err := RecordHeartbeatTick(rc.RunDir); err != nil {
		return fmt.Errorf("record heartbeat tick: %w", err)
	}
	state, heartbeat, err := RefreshMasterHeartbeatState(rc.RunDir)
	if err != nil {
		return fmt.Errorf("refresh heartbeat state: %w", err)
	}
	runState, err := EnsureRuntimeState(rc.RunDir, rc.Config)
	if err != nil {
		return fmt.Errorf("load runtime state: %w", err)
	}
	runState.Active = true
	runState.HeartbeatSeq = heartbeat.Seq
	runState.HeartbeatLag = state.HeartbeatLag
	runState.MasterWakePending = state.WakePending
	runState.MasterStale = state.StaleSince != ""
	runState.MasterStaleSince = state.StaleSince
	runState.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := SaveRunRuntimeState(RunRuntimeStatePath(rc.RunDir), runState); err != nil {
		return fmt.Errorf("save runtime state: %w", err)
	}
	if err := syncProjectStatusCache(projectRoot, runState); err != nil {
		return fmt.Errorf("update project status cache: %w", err)
	}
	if !SessionExists(rc.TmuxSession) {
		return nil
	}
	if err := sendAgentNudge(rc.TmuxSession+":master", rc.Config.Master.Engine); err != nil {
		return fmt.Errorf("nudge master: %w", err)
	}
	return nil
}
