package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	ar "github.com/vonbai/autoresearch"
)

// List scans all runs for the current project and prints a table.
func List(projectRoot string, _ []string) error {
	home, _ := os.UserHomeDir()
	runsDir := filepath.Join(home, ".autoresearch", "runs", ar.ProjectID(projectRoot))

	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No runs found.")
			return nil
		}
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tMODE\tSTATUS\tSESSIONS\tCREATED")

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		cfg, err := ar.LoadYAML[ar.Config](filepath.Join(runsDir, name, "goalx.yaml"))
		if err != nil {
			continue
		}

		status := "completed"
		tmuxSess := ar.TmuxSessionName(projectRoot, name)
		if SessionExists(tmuxSess) {
			status = "active"
		}

		sessions := sessionCount(&cfg)

		info, _ := e.Info()
		created := ""
		if info != nil {
			created = info.ModTime().Format("2006-01-02 15:04")
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", name, cfg.Mode, status, sessions, created)
	}
	return w.Flush()
}
