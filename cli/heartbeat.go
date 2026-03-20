package cli

import "fmt"

// HeartbeatCommand returns the shell command for the heartbeat tmux window.
// It runs a sleep loop that sends "Heartbeat: check now." to the master window.
func HeartbeatCommand(tmuxSession string, checkIntervalSeconds int) string {
	return fmt.Sprintf(
		`while sleep %d; do tmux send-keys -t %s:master 'Heartbeat: execute check cycle now.' Enter; done`,
		checkIntervalSeconds, tmuxSession,
	)
}
