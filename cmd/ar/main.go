package main

import (
	"fmt"
	"os"
)

const usage = `ar — autonomous research CLI

Usage:
  ar start   [--config ar.yaml]    Init workspace + launch tmux + AI agent
  ar status  [session-name]        Show experiment progress from journal
  ar attach  [session-name]        Attach to tmux session
  ar stop    [session-name]        Graceful shutdown
  ar report  [session-name]        Generate markdown report
  ar close   [session-name]        Cleanup branch + worktree

Run 'ar <command> --help' for details.`

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(0)
	}

	cmd := os.Args[1]
	switch cmd {
	case "start":
		fmt.Println("ar start: not yet implemented")
	case "status":
		fmt.Println("ar status: not yet implemented")
	case "attach":
		fmt.Println("ar attach: not yet implemented")
	case "stop":
		fmt.Println("ar stop: not yet implemented")
	case "report":
		fmt.Println("ar report: not yet implemented")
	case "close":
		fmt.Println("ar close: not yet implemented")
	case "--help", "-h", "help":
		fmt.Println(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
}
