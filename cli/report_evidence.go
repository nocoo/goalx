package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ReportEvidenceManifest struct {
	Version           int      `json:"version"`
	ReportPath        string   `json:"report_path"`
	Covers            []string `json:"covers"`
	RepoEvidencePaths []string `json:"repo_evidence_paths,omitempty"`
	ExternalRefs      []string `json:"external_refs,omitempty"`
}

func ReportEvidenceManifestPath(reportPath string) string {
	reportPath = strings.TrimSpace(reportPath)
	if reportPath == "" {
		return ""
	}
	ext := filepath.Ext(reportPath)
	if ext == "" {
		return reportPath + ".evidence.json"
	}
	return strings.TrimSuffix(reportPath, ext) + ".evidence.json"
}

func LoadReportEvidenceManifest(reportPath string) (*ReportEvidenceManifest, error) {
	manifestPath := ReportEvidenceManifestPath(reportPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read report evidence manifest %s: %w", manifestPath, err)
	}
	manifest, err := parseReportEvidenceManifest(data, reportPath)
	if err != nil {
		return nil, fmt.Errorf("parse report evidence manifest %s: %w", manifestPath, err)
	}
	return manifest, nil
}

func SaveReportEvidenceManifest(reportPath string, manifest *ReportEvidenceManifest) error {
	if manifest == nil {
		return fmt.Errorf("report evidence manifest is nil")
	}
	normalized := *manifest
	normalized.ReportPath = strings.TrimSpace(reportPath)
	if err := validateReportEvidenceManifest(reportPath, &normalized); err != nil {
		return err
	}
	data, err := json.MarshalIndent(&normalized, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(ReportEvidenceManifestPath(reportPath), data, 0o644)
}

func parseReportEvidenceManifest(data []byte, reportPath string) (*ReportEvidenceManifest, error) {
	var manifest ReportEvidenceManifest
	if err := decodeStrictJSON(data, &manifest); err != nil {
		return nil, err
	}
	if err := validateReportEvidenceManifest(reportPath, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func validateReportEvidenceManifest(reportPath string, manifest *ReportEvidenceManifest) error {
	if manifest == nil {
		return fmt.Errorf("report evidence manifest is nil")
	}
	if manifest.Version <= 0 {
		return fmt.Errorf("report evidence manifest version must be positive")
	}
	reportPath = strings.TrimSpace(reportPath)
	if reportPath == "" {
		return fmt.Errorf("report evidence manifest report path is required")
	}
	manifest.ReportPath = strings.TrimSpace(manifest.ReportPath)
	if manifest.ReportPath == "" {
		return fmt.Errorf("report evidence manifest report_path is required")
	}
	if filepath.Clean(manifest.ReportPath) != filepath.Clean(reportPath) {
		return fmt.Errorf("report evidence manifest report_path %q does not match report %q", manifest.ReportPath, reportPath)
	}
	if !fileExists(reportPath) {
		return fmt.Errorf("report evidence manifest report %q does not exist", reportPath)
	}
	covers := trimmedGoalCovers(manifest.Covers)
	if len(covers) == 0 {
		return fmt.Errorf("report evidence manifest covers is required")
	}
	if len(covers) != len(manifest.Covers) {
		return fmt.Errorf("report evidence manifest covers entries must be non-empty")
	}
	manifest.Covers = covers
	repoEvidencePaths := trimmedStrings(manifest.RepoEvidencePaths)
	if len(repoEvidencePaths) != len(manifest.RepoEvidencePaths) {
		return fmt.Errorf("report evidence manifest repo_evidence_paths entries must be non-empty")
	}
	manifest.RepoEvidencePaths = repoEvidencePaths
	for _, evidencePath := range manifest.RepoEvidencePaths {
		if !fileExists(evidencePath) {
			return fmt.Errorf("report evidence manifest repo evidence path %q does not exist", evidencePath)
		}
	}
	externalRefs := trimmedStrings(manifest.ExternalRefs)
	if len(externalRefs) != len(manifest.ExternalRefs) {
		return fmt.Errorf("report evidence manifest external_refs entries must be non-empty")
	}
	manifest.ExternalRefs = externalRefs
	for _, ref := range manifest.ExternalRefs {
		if ref == "" {
			return fmt.Errorf("report evidence manifest external_refs entries must be non-empty")
		}
	}
	return nil
}

func copyReportWithEvidence(srcReportPath, dstReportPath string) error {
	if err := copyFileIfExists(srcReportPath, dstReportPath); err != nil {
		return err
	}
	manifest, err := LoadReportEvidenceManifest(srcReportPath)
	if err != nil {
		return err
	}
	if manifest == nil {
		return nil
	}
	return SaveReportEvidenceManifest(dstReportPath, manifest)
}

func validateResearchAcceptance(runDir string) (string, error) {
	var evidence strings.Builder
	summaryPath := SummaryPath(runDir)
	fmt.Fprintf(&evidence, "summary: %s\n", summaryPath)
	if !fileExists(summaryPath) {
		return evidence.String(), fmt.Errorf("summary missing at %s", summaryPath)
	}

	reportsDir := ReportsDir(runDir)
	entries, err := os.ReadDir(reportsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return evidence.String(), fmt.Errorf("no research reports found in %s", reportsDir)
		}
		return evidence.String(), fmt.Errorf("read reports dir: %w", err)
	}

	reportPaths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".evidence.json") {
			continue
		}
		if filepath.Ext(name) != ".md" {
			continue
		}
		reportPaths = append(reportPaths, filepath.Join(reportsDir, name))
	}
	sort.Strings(reportPaths)
	fmt.Fprintf(&evidence, "reports: %d\n", len(reportPaths))
	if len(reportPaths) == 0 {
		return evidence.String(), fmt.Errorf("no research reports found in %s", reportsDir)
	}

	contract, err := RequireObjectiveContract(runDir)
	if err != nil {
		return evidence.String(), err
	}
	if strings.TrimSpace(contract.State) != objectiveContractStateLocked {
		return evidence.String(), fmt.Errorf("objective contract must be locked")
	}

	goalClauses := objectiveClausesBySurface(contract, objectiveRequiredSurfaceGoal)
	if len(goalClauses) == 0 {
		return evidence.String(), fmt.Errorf("objective contract has no goal clauses")
	}

	coverageByClause := make(map[string]int, len(goalClauses))
	externalCoverageByClause := make(map[string]bool, len(goalClauses))
	for _, reportPath := range reportPaths {
		manifest, err := LoadReportEvidenceManifest(reportPath)
		if err != nil {
			return evidence.String(), err
		}
		if manifest == nil {
			return evidence.String(), fmt.Errorf("report evidence manifest missing for %s", reportPath)
		}
		fmt.Fprintf(&evidence, "- %s -> %s\n", reportPath, ReportEvidenceManifestPath(reportPath))
		for _, clauseID := range manifest.Covers {
			if _, ok := goalClauses[clauseID]; !ok {
				return evidence.String(), fmt.Errorf("report evidence manifest %s references unknown objective clause %q", ReportEvidenceManifestPath(reportPath), clauseID)
			}
			coverageByClause[clauseID]++
			if len(manifest.ExternalRefs) > 0 {
				externalCoverageByClause[clauseID] = true
			}
		}
	}

	missingCoverage := make([]string, 0)
	missingExternalRefs := make([]string, 0)
	for clauseID, clause := range goalClauses {
		if coverageByClause[clauseID] == 0 {
			missingCoverage = append(missingCoverage, clauseID)
			continue
		}
		if reportEvidenceRequiresExternalRefs(clause) && !externalCoverageByClause[clauseID] {
			missingExternalRefs = append(missingExternalRefs, clauseID)
		}
	}
	sort.Strings(missingCoverage)
	sort.Strings(missingExternalRefs)
	if len(missingCoverage) > 0 {
		return evidence.String(), fmt.Errorf("objective clauses missing report evidence coverage: %s", strings.Join(missingCoverage, ", "))
	}
	if len(missingExternalRefs) > 0 {
		return evidence.String(), fmt.Errorf("objective clauses require external_refs coverage: %s", strings.Join(missingExternalRefs, ", "))
	}

	fmt.Fprintf(&evidence, "coverage: %d clauses\n", len(goalClauses))
	return evidence.String(), nil
}

func reportEvidenceRequiresExternalRefs(clause ObjectiveClause) bool {
	switch strings.TrimSpace(clause.Kind) {
	case objectiveClauseKindVerification, objectiveClauseKindQualityBar:
		return true
	default:
		return false
	}
}

func trimmedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
