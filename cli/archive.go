package cli

import "fmt"

// Archive creates a git tag for a session branch, preserving it.
func Archive(projectRoot string, args []string) error {
	runName, rest, err := extractRunFlag(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return fmt.Errorf("usage: goalx archive [--run NAME] <session-name>")
	}
	sessionName := rest[0]

	rc, err := ResolveRun(projectRoot, runName)
	if err != nil {
		return err
	}

	idx, err := parseSessionIndex(sessionName)
	if err != nil {
		return err
	}
	if idx > sessionCount(rc.Config) {
		return fmt.Errorf("session %q out of range for run %q", sessionName, rc.Name)
	}

	branch := fmt.Sprintf("goalx/%s/%d", rc.Config.Name, idx)
	tag := fmt.Sprintf("goalx-archive/%s/%d", rc.Config.Name, idx)

	if err := TagArchive(rc.ProjectRoot, branch, tag); err != nil {
		return fmt.Errorf("tag %s: %w", tag, err)
	}
	fmt.Printf("Archived %s as tag %s\n", sessionName, tag)
	return nil
}
