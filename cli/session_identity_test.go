package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestSessionIdentityPathAndRoundTrip(t *testing.T) {
	runDir := t.TempDir()
	if got, want := SessionIdentityPath(runDir, "session-1"), filepath.Join(runDir, "sessions", "session-1", "identity.json"); got != want {
		t.Fatalf("SessionIdentityPath = %q, want %q", got, want)
	}

	meta := &RunMetadata{
		Version:         1,
		Objective:       "build durable knowledge base",
		ProjectRoot:     "/tmp/project",
		ProtocolVersion: 2,
		RunID:           "run_abc123",
		RootRunID:       "run_root123",
		Epoch:           3,
		CharterID:       "charter_abc123",
	}
	charter, err := NewRunCharter(runDir, "knowledge-base", "knowledge-base objective", meta)
	if err != nil {
		t.Fatalf("NewRunCharter: %v", err)
	}
	if err := SaveRunCharter(RunCharterPath(runDir), charter); err != nil {
		t.Fatalf("SaveRunCharter: %v", err)
	}

	identity, err := NewSessionIdentity(runDir, "session-1", "master-derived-develop", goalx.ModeDevelop, "codex", "gpt-5.4", goalx.TargetConfig{Files: []string{"main.go"}})
	if err != nil {
		t.Fatalf("NewSessionIdentity: %v", err)
	}
	if identity.SessionName != "session-1" {
		t.Fatalf("SessionName = %q", identity.SessionName)
	}
	if identity.RoleKind != "master-derived-develop" {
		t.Fatalf("RoleKind = %q", identity.RoleKind)
	}
	if identity.Mode != string(goalx.ModeDevelop) {
		t.Fatalf("Mode = %q", identity.Mode)
	}
	if identity.Engine != "codex" || identity.Model != "gpt-5.4" {
		t.Fatalf("engine/model = %q/%q", identity.Engine, identity.Model)
	}
	if identity.OriginCharterID != charter.CharterID {
		t.Fatalf("OriginCharterID = %q, want %q", identity.OriginCharterID, charter.CharterID)
	}
	if identity.Target.Files[0] != "main.go" {
		t.Fatalf("session identity target = %+v", identity.Target)
	}
	if identity.CreatedAt == "" {
		t.Fatal("CreatedAt empty")
	}

	if err := SaveSessionIdentity(SessionIdentityPath(runDir, "session-1"), identity); err != nil {
		t.Fatalf("SaveSessionIdentity: %v", err)
	}
	if err := SaveSessionIdentity(SessionIdentityPath(runDir, "session-1"), identity); err == nil {
		t.Fatal("second SaveSessionIdentity should fail for immutable session identity storage")
	}
	reloaded, err := LoadSessionIdentity(SessionIdentityPath(runDir, "session-1"))
	if err != nil {
		t.Fatalf("LoadSessionIdentity: %v", err)
	}
	if reloaded == nil {
		t.Fatal("reloaded session identity is nil")
	}
	if reloaded.OriginCharterID != identity.OriginCharterID {
		t.Fatalf("OriginCharterID = %q, want %q", reloaded.OriginCharterID, identity.OriginCharterID)
	}
	if reloaded.Target.Files[0] != "main.go" {
		t.Fatalf("reloaded session identity target = %+v", reloaded.Target)
	}
}

func TestNewSessionIdentityRequiresRunCharter(t *testing.T) {
	runDir := t.TempDir()
	if _, err := NewSessionIdentity(runDir, "session-1", "master-derived-develop", goalx.ModeDevelop, "codex", "gpt-5.4", goalx.TargetConfig{}); err == nil {
		t.Fatal("NewSessionIdentity should fail when run-charter.json is missing")
	}
}

func TestSessionIdentityRoundTripKeepsSourceAndRole(t *testing.T) {
	runDir := t.TempDir()
	meta := &RunMetadata{Version: 1, Objective: "ship", ProtocolVersion: 2, RunID: "run_1", RootRunID: "run_1", Epoch: 1}
	charter, err := NewRunCharter(runDir, "demo", "demo objective", meta)
	if err != nil {
		t.Fatalf("NewRunCharter: %v", err)
	}
	if err := SaveRunCharter(RunCharterPath(runDir), charter); err != nil {
		t.Fatalf("SaveRunCharter: %v", err)
	}

	identity, err := NewSessionIdentity(runDir, "session-2", "master-derived-research", goalx.ModeResearch, "claude-code", "opus", goalx.TargetConfig{Files: []string{"report.md"}})
	if err != nil {
		t.Fatalf("NewSessionIdentity: %v", err)
	}
	textBytes, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		t.Fatalf("marshal session identity: %v", err)
	}
	text := string(textBytes)
	for _, want := range []string{"session-2", "master-derived-research", "claude-code", "opus", "report.md"} {
		if !strings.Contains(text, want) {
			t.Fatalf("session identity JSON missing %q:\n%s", want, text)
		}
	}
}

func TestLoadSessionIdentityDoesNotDefaultMode(t *testing.T) {
	runDir := t.TempDir()
	path := SessionIdentityPath(runDir, "session-1")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir session identity dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"session_name":"session-1"}`), 0o644); err != nil {
		t.Fatalf("write session identity: %v", err)
	}

	identity, err := LoadSessionIdentity(path)
	if err != nil {
		t.Fatalf("LoadSessionIdentity: %v", err)
	}
	if identity.Mode != "" {
		t.Fatalf("Mode = %q, want empty", identity.Mode)
	}
}
