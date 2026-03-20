package cli

import "fmt"

// Stop kills the tmux session for the current run.
func Stop(projectRoot string, args []string) error {
	runName, rest, err := extractRunFlag(args)
	if err != nil {
		return err
	}
	if runName == "" && len(rest) == 1 {
		runName = rest[0]
		rest = nil
	}
	if len(rest) > 0 {
		return fmt.Errorf("usage: goalx stop [--run NAME]")
	}

	rc, err := ResolveRun(projectRoot, runName)
	if err != nil {
		return err
	}

	if !SessionExists(rc.TmuxSession) {
		fmt.Printf("Run '%s' is not active (no tmux session).\n", rc.Name)
		return nil
	}

	if err := KillSession(rc.TmuxSession); err != nil {
		return fmt.Errorf("kill tmux session %s: %w", rc.TmuxSession, err)
	}
	fmt.Printf("Run '%s' stopped (tmux session %s killed).\n", rc.Name, rc.TmuxSession)
	return nil
}
