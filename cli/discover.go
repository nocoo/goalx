package cli

import (
	"os"
	"path/filepath"
	"strings"
)

// DiscoverContextFiles expands paths into a list of relevant files.
// Directories are scanned for key files; regular files are included directly.
// All returned paths are absolute.
func DiscoverContextFiles(paths []string) ([]string, error) {
	var result []string
	seen := make(map[string]bool)

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			if !seen[absPath] {
				result = append(result, absPath)
				seen[absPath] = true
			}
			continue
		}

		// Directory: discover key files
		discovered := discoverKeyFiles(absPath)
		for _, f := range discovered {
			if !seen[f] {
				result = append(result, f)
				seen[f] = true
			}
		}
	}

	return result, nil
}

// discoverKeyFiles finds important files in a project directory.
func discoverKeyFiles(dir string) []string {
	var files []string

	// Priority 1: documentation and config
	topLevel := []string{
		"README.md", "readme.md", "README",
		"CLAUDE.md", "AGENTS.md",
		"go.mod", "package.json", "pyproject.toml", "Cargo.toml",
		"Makefile", "Dockerfile",
	}
	for _, name := range topLevel {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			files = append(files, p)
		}
	}

	// Priority 2: main entry points (search common patterns)
	entryPoints := []string{
		"main.go", "cmd/*/main.go",
		"src/main.*", "src/index.*", "src/app.*",
		"lib/main.*", "app/main.*",
		"index.ts", "index.js", "app.py", "main.py",
	}
	for _, pattern := range entryPoints {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		for _, m := range matches {
			files = append(files, m)
		}
	}

	// Priority 3: doc directory (if exists, grab key files)
	docDirs := []string{"docs", "doc", "documentation"}
	for _, d := range docDirs {
		docDir := filepath.Join(dir, d)
		if info, err := os.Stat(docDir); err == nil && info.IsDir() {
			entries, _ := os.ReadDir(docDir)
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				if strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".rst") {
					files = append(files, filepath.Join(docDir, name))
				}
			}
			break // only first found doc dir
		}
	}

	return files
}
