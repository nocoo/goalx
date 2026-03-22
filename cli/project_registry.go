package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	goalx "github.com/vonbai/goalx"
)

type ProjectRunRef struct {
	Name      string `json:"name"`
	Mode      string `json:"mode,omitempty"`
	Objective string `json:"objective,omitempty"`
	State     string `json:"state,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type ProjectRegistry struct {
	Version    int                      `json:"version"`
	FocusedRun string                   `json:"focused_run,omitempty"`
	ActiveRuns map[string]ProjectRunRef `json:"active_runs,omitempty"`
	SavedRuns  map[string]ProjectRunRef `json:"saved_runs,omitempty"`
	UpdatedAt  string                   `json:"updated_at,omitempty"`
}

func ProjectRegistryPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".goalx", "runs.json")
}

func LoadProjectRegistry(projectRoot string) (*ProjectRegistry, error) {
	data, err := os.ReadFile(ProjectRegistryPath(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return &ProjectRegistry{
				Version:    1,
				ActiveRuns: map[string]ProjectRunRef{},
				SavedRuns:  map[string]ProjectRunRef{},
			}, nil
		}
		return nil, fmt.Errorf("read project registry: %w", err)
	}
	reg := &ProjectRegistry{}
	if len(strings.TrimSpace(string(data))) == 0 {
		reg.Version = 1
		reg.ActiveRuns = map[string]ProjectRunRef{}
		reg.SavedRuns = map[string]ProjectRunRef{}
		return reg, nil
	}
	if err := json.Unmarshal(data, reg); err != nil {
		return nil, fmt.Errorf("parse project registry: %w", err)
	}
	if reg.Version == 0 {
		reg.Version = 1
	}
	if reg.ActiveRuns == nil {
		reg.ActiveRuns = map[string]ProjectRunRef{}
	}
	if reg.SavedRuns == nil {
		reg.SavedRuns = map[string]ProjectRunRef{}
	}
	return reg, nil
}

func SaveProjectRegistry(projectRoot string, reg *ProjectRegistry) error {
	if reg == nil {
		return fmt.Errorf("project registry is nil")
	}
	if reg.Version == 0 {
		reg.Version = 1
	}
	if reg.ActiveRuns == nil {
		reg.ActiveRuns = map[string]ProjectRunRef{}
	}
	if reg.SavedRuns == nil {
		reg.SavedRuns = map[string]ProjectRunRef{}
	}
	reg.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal project registry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(ProjectRegistryPath(projectRoot)), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(ProjectRegistryPath(projectRoot), data, 0o644); err != nil {
		return fmt.Errorf("write project registry: %w", err)
	}
	return nil
}

func RegisterActiveRun(projectRoot string, cfg *goalx.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	reg, err := LoadProjectRegistry(projectRoot)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	reg.ActiveRuns[cfg.Name] = ProjectRunRef{
		Name:      cfg.Name,
		Mode:      string(cfg.Mode),
		Objective: cfg.Objective,
		State:     "active",
		UpdatedAt: now,
	}
	if reg.FocusedRun == "" {
		reg.FocusedRun = cfg.Name
	}
	return SaveProjectRegistry(projectRoot, reg)
}

func MarkRunInactive(projectRoot, runName string) error {
	reg, err := LoadProjectRegistry(projectRoot)
	if err != nil {
		return err
	}
	delete(reg.ActiveRuns, runName)
	if reg.FocusedRun == runName {
		reg.FocusedRun = ""
		if len(reg.ActiveRuns) == 1 {
			for name := range reg.ActiveRuns {
				reg.FocusedRun = name
			}
		}
	}
	return SaveProjectRegistry(projectRoot, reg)
}

func RegisterSavedRun(projectRoot string, cfg *goalx.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	reg, err := LoadProjectRegistry(projectRoot)
	if err != nil {
		return err
	}
	reg.SavedRuns[cfg.Name] = ProjectRunRef{
		Name:      cfg.Name,
		Mode:      string(cfg.Mode),
		Objective: cfg.Objective,
		State:     "saved",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	return SaveProjectRegistry(projectRoot, reg)
}

func RemoveRunRegistration(projectRoot, runName string) error {
	reg, err := LoadProjectRegistry(projectRoot)
	if err != nil {
		return err
	}
	delete(reg.ActiveRuns, runName)
	delete(reg.SavedRuns, runName)
	if reg.FocusedRun == runName {
		reg.FocusedRun = ""
	}
	return SaveProjectRegistry(projectRoot, reg)
}

func ResolveDefaultRunName(projectRoot string) (string, error) {
	reg, err := LoadProjectRegistry(projectRoot)
	if err != nil {
		return "", err
	}
	if reg.FocusedRun != "" {
		if _, ok := reg.ActiveRuns[reg.FocusedRun]; ok {
			return reg.FocusedRun, nil
		}
	}
	if len(reg.ActiveRuns) == 1 {
		for name := range reg.ActiveRuns {
			return name, nil
		}
	}
	if len(reg.ActiveRuns) > 1 {
		return "", fmt.Errorf("multiple active runs: %s (specify --run)", strings.Join(sortedRunNames(reg.ActiveRuns), ", "))
	}

	home, _ := os.UserHomeDir()
	runsDir := filepath.Join(home, ".goalx", "runs", goalx.ProjectID(projectRoot))
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no runs found")
		}
		return "", err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	switch len(names) {
	case 0:
		return "", fmt.Errorf("no runs found")
	case 1:
		return names[0], nil
	default:
		return "", fmt.Errorf("multiple runs: %s (specify --run)", strings.Join(names, ", "))
	}
}

func sortedRunNames(m map[string]ProjectRunRef) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
