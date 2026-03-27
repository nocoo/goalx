package cli

import "fmt"

type DurableSurfaceClass string
type DurableSurfaceWriteMode string
type DurableSurfaceName string

const (
	DurableSurfaceClassStructuredState DurableSurfaceClass = "structured_state"
	DurableSurfaceClassEventLog        DurableSurfaceClass = "event_log"
	DurableSurfaceClassArtifact        DurableSurfaceClass = "artifact"

	DurableSurfaceWriteModeReplace DurableSurfaceWriteMode = "replace"
	DurableSurfaceWriteModeAppend  DurableSurfaceWriteMode = "append"

	DurableSurfaceGoal            DurableSurfaceName = "goal"
	DurableSurfaceAcceptance      DurableSurfaceName = "acceptance"
	DurableSurfaceCoordination    DurableSurfaceName = "coordination"
	DurableSurfaceStatus          DurableSurfaceName = "status"
	DurableSurfaceGoalLog         DurableSurfaceName = "goal-log"
	DurableSurfaceEvolution       DurableSurfaceName = "evolution"
	DurableSurfaceSummary         DurableSurfaceName = "summary"
	DurableSurfaceCompletionProof DurableSurfaceName = "completion-proof"
)

type DurableSurfaceSpec struct {
	Name               DurableSurfaceName
	Class              DurableSurfaceClass
	WriteMode          DurableSurfaceWriteMode
	Strict             bool
	FrameworkReadsBody bool
	Path               func(runDir string) string
}

var durableSurfaceRegistry = map[DurableSurfaceName]DurableSurfaceSpec{
	DurableSurfaceGoal: {
		Name:               DurableSurfaceGoal,
		Class:              DurableSurfaceClassStructuredState,
		WriteMode:          DurableSurfaceWriteModeReplace,
		Strict:             true,
		FrameworkReadsBody: true,
		Path:               GoalPath,
	},
	DurableSurfaceAcceptance: {
		Name:               DurableSurfaceAcceptance,
		Class:              DurableSurfaceClassStructuredState,
		WriteMode:          DurableSurfaceWriteModeReplace,
		Strict:             true,
		FrameworkReadsBody: true,
		Path:               AcceptanceStatePath,
	},
	DurableSurfaceCoordination: {
		Name:               DurableSurfaceCoordination,
		Class:              DurableSurfaceClassStructuredState,
		WriteMode:          DurableSurfaceWriteModeReplace,
		Strict:             true,
		FrameworkReadsBody: true,
		Path:               CoordinationPath,
	},
	DurableSurfaceStatus: {
		Name:               DurableSurfaceStatus,
		Class:              DurableSurfaceClassStructuredState,
		WriteMode:          DurableSurfaceWriteModeReplace,
		Strict:             true,
		FrameworkReadsBody: true,
		Path:               RunStatusPath,
	},
	DurableSurfaceGoalLog: {
		Name:               DurableSurfaceGoalLog,
		Class:              DurableSurfaceClassEventLog,
		WriteMode:          DurableSurfaceWriteModeAppend,
		Strict:             true,
		FrameworkReadsBody: false,
		Path:               GoalLogPath,
	},
	DurableSurfaceEvolution: {
		Name:               DurableSurfaceEvolution,
		Class:              DurableSurfaceClassEventLog,
		WriteMode:          DurableSurfaceWriteModeAppend,
		Strict:             true,
		FrameworkReadsBody: false,
		Path:               EvolutionLogPath,
	},
	DurableSurfaceSummary: {
		Name:               DurableSurfaceSummary,
		Class:              DurableSurfaceClassArtifact,
		WriteMode:          DurableSurfaceWriteModeReplace,
		Strict:             false,
		FrameworkReadsBody: false,
		Path:               SummaryPath,
	},
	DurableSurfaceCompletionProof: {
		Name:               DurableSurfaceCompletionProof,
		Class:              DurableSurfaceClassArtifact,
		WriteMode:          DurableSurfaceWriteModeReplace,
		Strict:             false,
		FrameworkReadsBody: false,
		Path:               CompletionStatePath,
	},
}

func LookupDurableSurface(name string) (DurableSurfaceSpec, error) {
	spec, ok := durableSurfaceRegistry[DurableSurfaceName(name)]
	if !ok {
		return DurableSurfaceSpec{}, fmt.Errorf("unknown durable surface %q", name)
	}
	return spec, nil
}
