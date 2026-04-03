// +build linux

package cli

func ProbePlatformResourceState() (*ResourceState, error) {
	return ProbeLinuxResourceState()
}

func collectPlatformProcessFacts(runDir string) (*GoalXProcessFacts, error) {
	return collectGoalXProcessFacts(runDir)
}
