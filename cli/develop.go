package cli

import (
	"fmt"
)

func Develop(projectRoot string, args []string) error {
	if wantsHelp(args) {
		fmt.Println(launchUsage("develop"))
		return nil
	}
	return runEntrypoint(projectRoot, prependRunIntent(args, runIntentDevelop), nil)
}
