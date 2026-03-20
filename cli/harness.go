package cli

import "strings"

// ResolveHarness replaces variables in a harness command template.
func ResolveHarness(command, worktreePath string) string {
	r := strings.NewReplacer(
		"{worktree}", worktreePath,
	)
	return r.Replace(command)
}
