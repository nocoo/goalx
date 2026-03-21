package cli

import "fmt"

var (
	autoInit  = Init
	autoStart = Start
)

// Auto initializes a run, starts the master, and exits.
// The master continues orchestrating in tmux.
func Auto(projectRoot string, args []string) error {
	initArgs := append([]string(nil), args...)
	if len(initArgs) > 0 && !hasMode(initArgs) {
		initArgs = append(initArgs[:1:1], append([]string{"--research"}, initArgs[1:]...)...)
	}

	if err := autoInit(projectRoot, initArgs); err != nil {
		return fmt.Errorf("init: %w", err)
	}
	if err := autoStart(projectRoot, nil); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	fmt.Println("Run started.")
	fmt.Println("Use `goalx status`, `goalx observe`, or `goalx attach` to monitor progress.")
	return nil
}
