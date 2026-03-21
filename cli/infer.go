package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func InferHarness(projectRoot string) string {
	if fileExists(projectRoot, "go.mod") {
		return "go build ./... && go test ./... -count=1 && go vet ./..."
	}
	if fileExists(projectRoot, "Cargo.toml") {
		return "cargo build && cargo test && cargo clippy"
	}
	if scripts := loadPackageScripts(projectRoot); scripts != nil {
		if _, ok := scripts["test"]; ok {
			return "npm test"
		}
		return "npx tsc --noEmit 2>/dev/null || true"
	}
	if fileExists(projectRoot, "pyproject.toml") || fileExists(projectRoot, "requirements.txt") {
		return "python -m pytest"
	}
	if fileExists(projectRoot, "Makefile") {
		return "make test 2>/dev/null || make"
	}
	if fileExists(projectRoot, "Dockerfile") {
		return "docker build ."
	}
	return ""
}

func InferTarget(projectRoot string) []string {
	if fileExists(projectRoot, "go.mod") || fileExists(projectRoot, "Cargo.toml") {
		return []string{"."}
	}
	if dirExists(projectRoot, "src") {
		return []string{"src/"}
	}
	return []string{"."}
}

func fileExists(root, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}

func dirExists(root, name string) bool {
	info, err := os.Stat(filepath.Join(root, name))
	return err == nil && info.IsDir()
}

func loadPackageScripts(root string) map[string]any {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]any `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	return pkg.Scripts
}
