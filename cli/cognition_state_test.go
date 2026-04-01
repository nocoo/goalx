package cli

import (
	"os"
	"strings"
	"testing"
)

func TestSaveCognitionStateRoundTrip(t *testing.T) {
	runDir := t.TempDir()
	path := CognitionStatePath(runDir)
	state := &CognitionState{
		Version: 1,
		Scopes: []CognitionScopeState{
			{
				Scope:        "run-root",
				WorktreePath: "/abs/run-root",
				Providers: []CognitionProviderState{
					{
						Name:           "repo-native",
						InvocationKind: "builtin",
						Available:      true,
						HeadRevision:   "def456",
						Capabilities:   []string{"file_search", "git_diff"},
					},
				},
			},
		},
	}

	if err := SaveCognitionState(path, state); err != nil {
		t.Fatalf("SaveCognitionState: %v", err)
	}
	loaded, err := LoadCognitionState(path)
	if err != nil {
		t.Fatalf("LoadCognitionState: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadCognitionState returned nil state")
	}
	if len(loaded.Scopes) != 1 || len(loaded.Scopes[0].Providers) != 1 {
		t.Fatalf("scopes = %#v, want one scope with one provider", loaded.Scopes)
	}
}

func TestLoadCognitionStateRejectsProviderWithoutCapabilities(t *testing.T) {
	path := CognitionStatePath(t.TempDir())
	if err := os.WriteFile(path, []byte(`{
  "version": 1,
  "scopes": [
    {
      "scope": "run-root",
      "providers": [
        {
          "name": "repo-native",
          "invocation_kind": "builtin",
          "available": true
        }
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := LoadCognitionState(path)
	if err == nil {
		t.Fatal("LoadCognitionState should reject provider without capabilities")
	}
	if !strings.Contains(err.Error(), "capabilities") {
		t.Fatalf("LoadCognitionState error = %v, want capabilities hint", err)
	}
}
