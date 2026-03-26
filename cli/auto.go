package cli

import "fmt"

func Auto(projectRoot string, args []string) error {
	if wantsHelp(args) {
		fmt.Println(launchUsage("auto"))
		return nil
	}
	return runEntrypoint(projectRoot, prependRunIntent(args, runIntentDeliver), nil)
}

func autoWithOptions(projectRoot string, opts launchOptions) error {
	return startResolvedLaunch(projectRoot, opts)
}

func printAutoStarted() {
	fmt.Println("Run started.")
	fmt.Println("Use `goalx status`, `goalx observe`, or `goalx attach` to monitor progress.")
}
