package cli

import (
	"fmt"
)

func Research(projectRoot string, args []string) error {
	if wantsHelp(args) {
		fmt.Println(launchUsage("research"))
		return nil
	}
	return runEntrypoint(projectRoot, prependRunIntent(args, runIntentResearch), nil)
}
