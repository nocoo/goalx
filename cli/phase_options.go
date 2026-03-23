package cli

import (
	"fmt"
	"strconv"
	"strings"

	goalx "github.com/vonbai/goalx"
)

type phaseOptions struct {
	From           string
	Name           string
	Objective      string
	Parallel       int
	ContextPaths   []string
	Strategies     []string
	DiversityHints []string
	Master         string
	ResearchRole   string
	DevelopRole    string
	Preset         string
	BudgetSeconds  int
	WriteConfig    bool
}

func phaseUsage(command string) string {
	return fmt.Sprintf(`usage: goalx %s --from RUN [--name NAME] [--objective TEXT] [--parallel N] [--preset NAME] [--master ENGINE/MODEL] [--research-role ENGINE/MODEL] [--develop-role ENGINE/MODEL] [--context PATHS] [--strategy NAMES] [--budget-seconds N] [--write-config]

notes:
  --from RUN is required and must reference a saved run.
  --parallel is optional initial fan-out for the new phase run.
  direct start is the default; use --write-config only for advanced config-first control.`, command)
}

func parsePhaseOptions(command string, args []string) (phaseOptions, error) {
	opts := phaseOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --from")
			}
			i++
			opts.From = args[i]
		case "--name":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --name")
			}
			i++
			opts.Name = args[i]
		case "--objective":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --objective")
			}
			i++
			opts.Objective = args[i]
		case "--parallel":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --parallel")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 {
				return opts, fmt.Errorf("invalid --parallel value %q", args[i])
			}
			opts.Parallel = n
		case "--context":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --context")
			}
			i++
			opts.ContextPaths = strings.Split(args[i], ",")
		case "--strategy":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --strategy")
			}
			i++
			opts.Strategies = strings.Split(args[i], ",")
		case "--master":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --master")
			}
			i++
			opts.Master = args[i]
		case "--research-role":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --research-role")
			}
			i++
			opts.ResearchRole = args[i]
		case "--develop-role":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --develop-role")
			}
			i++
			opts.DevelopRole = args[i]
		case "--preset":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --preset")
			}
			i++
			opts.Preset = args[i]
		case "--budget-seconds":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --budget-seconds")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 0 {
				return opts, fmt.Errorf("invalid --budget-seconds value %q", args[i])
			}
			opts.BudgetSeconds = n
		case "--write-config":
			opts.WriteConfig = true
		case "--engine", "--model":
			return opts, fmt.Errorf("%s is ambiguous; use --master, --research-role, or --develop-role", args[i])
		default:
			return opts, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if strings.TrimSpace(opts.From) == "" {
		return opts, fmt.Errorf("usage: goalx %s --from RUN [flags]", command)
	}
	return opts, nil
}

func mergeNextConfigIntoPhaseOptions(opts phaseOptions, nc *nextConfigJSON, phaseMode goalx.Mode) phaseOptions {
	if nc == nil {
		return opts
	}
	if opts.Parallel == 0 && nc.Parallel > 0 {
		opts.Parallel = nc.Parallel
	}
	if opts.Objective == "" && nc.Objective != "" {
		opts.Objective = nc.Objective
	}
	if opts.Preset == "" && nc.Preset != "" {
		opts.Preset = nc.Preset
	}
	if opts.BudgetSeconds == 0 && nc.BudgetSeconds > 0 {
		opts.BudgetSeconds = nc.BudgetSeconds
	}
	if len(opts.ContextPaths) == 0 && len(nc.Context) > 0 {
		opts.ContextPaths = append([]string(nil), nc.Context...)
	}
	if len(opts.Strategies) == 0 && len(nc.Strategies) > 0 {
		opts.Strategies = append([]string(nil), nc.Strategies...)
	}
	if len(opts.DiversityHints) == 0 && len(nc.DiversityHints) > 0 {
		opts.DiversityHints = append([]string(nil), nc.DiversityHints...)
	}
	if opts.Master == "" && nc.MasterEngine != "" && nc.MasterModel != "" {
		opts.Master = nc.MasterEngine + "/" + nc.MasterModel
	}
	if nc.Engine != "" && nc.Model != "" {
		targetMode := phaseMode
		if nc.Mode == "research" {
			targetMode = goalx.ModeResearch
		} else if nc.Mode == "develop" {
			targetMode = goalx.ModeDevelop
		}
		switch targetMode {
		case goalx.ModeResearch:
			if opts.ResearchRole == "" {
				opts.ResearchRole = nc.Engine + "/" + nc.Model
			}
		case goalx.ModeDevelop:
			if opts.DevelopRole == "" {
				opts.DevelopRole = nc.Engine + "/" + nc.Model
			}
		}
	}
	return opts
}
