package cli

import "fmt"

var (
	autoStart = Start
)

// Auto initializes a run, starts the master, and exits.
// The master continues orchestrating in tmux.
func Auto(projectRoot string, args []string) error {
	if wantsHelp(args) {
		fmt.Println(launchUsage("auto"))
		return nil
	}
	initArgs := append([]string(nil), args...)
	if len(initArgs) > 0 {
		hasMode := false
		for _, arg := range initArgs {
			if arg == "--research" || arg == "--develop" {
				hasMode = true
				break
			}
		}
		if !hasMode {
			initArgs = append(initArgs[:1:1], append([]string{"--research"}, initArgs[1:]...)...)
		}
	}

	if err := autoStart(projectRoot, initArgs); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	fmt.Println("Run started.")
	fmt.Println("Use `goalx status`, `goalx observe`, or `goalx attach` to monitor progress.")
	return nil
}
