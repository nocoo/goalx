package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	goalx "github.com/vonbai/goalx"
)

// Status shows the current progress for each session in a run.
func Status(projectRoot string, args []string) error {
	runName, sessionFilter, err := parseStatusArgs(args)
	if err != nil {
		return err
	}

	rc, err := ResolveRun(projectRoot, runName)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SESSION\tLAST_ROUND\tSTATUS\tSUMMARY")
	coord, _ := LoadCoordinationState(CoordinationPath(rc.RunDir))

	// Session journals
	indexes, err := existingSessionIndexes(rc.RunDir)
	if err != nil {
		return err
	}
	if len(indexes) == 0 {
		for i := range goalx.ExpandSessions(rc.Config) {
			indexes = append(indexes, i+1)
		}
	}
	for _, num := range indexes {
		sName := SessionName(num)
		if sessionFilter != "" && sName != sessionFilter {
			continue
		}
		jPath := JournalPath(rc.RunDir, sName)
		entries, _ := goalx.LoadJournal(jPath)

		lastRound := "-"
		status := "pending"
		if len(entries) > 0 {
			last := entries[len(entries)-1]
			if last.Round > 0 {
				lastRound = fmt.Sprintf("%d", last.Round)
			}
			if last.Status != "" {
				status = last.Status
			}
		}

		summary := goalx.Summary(entries)
		if coord != nil {
			if sess, ok := coord.Sessions[sName]; ok {
				if sess.LastRound > 0 {
					lastRound = fmt.Sprintf("%d", sess.LastRound)
				}
				if sess.State != "" {
					status = sess.State
				}
				switch sess.State {
				case "parked":
					if sess.Scope != "" {
						summary = "parked: " + sess.Scope
					} else {
						summary = "parked"
					}
				case "blocked":
					if sess.BlockedBy != "" {
						summary = "blocked: " + sess.BlockedBy
					}
				case "active":
					if summary == "no entries" && sess.Scope != "" {
						summary = "active: " + sess.Scope
					}
				}
			}
		}
		if guidanceState, err := LoadSessionGuidanceState(SessionGuidanceStatePath(rc.RunDir, sName)); err == nil && guidanceState != nil && guidanceState.Pending {
			if status == "idle" || status == "pending" {
				status = "guidance-pending"
			}
			if summary == "no entries" {
				summary = "guidance pending"
			} else if !strings.Contains(summary, "guidance pending") {
				summary += " | guidance pending"
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", sName, lastRound, status, summary)
	}

	// Master journal
	masterPath := filepath.Join(rc.RunDir, "master.jsonl")
	masterEntries, _ := goalx.LoadJournal(masterPath)
	masterSummary := goalx.Summary(masterEntries)
	fmt.Fprintf(w, "master\t-\t-\t%s\n", masterSummary)

	return w.Flush()
}
