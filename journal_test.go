package goalx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJournal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	data := `{"round":1,"commit":"abc","desc":"read code","status":"progress"}
{"round":2,"commit":"def","desc":"split file","status":"progress"}
{"round":3,"commit":"ghi","desc":"all tests pass","status":"done"}
`
	os.WriteFile(path, []byte(data), 0644)

	entries, err := LoadJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("len = %d, want 3", len(entries))
	}
	if entries[0].Round != 1 || entries[0].Commit != "abc" {
		t.Errorf("entry[0] = %+v", entries[0])
	}
	if entries[2].Status != "done" {
		t.Errorf("entry[2].Status = %q, want done", entries[2].Status)
	}
}

func TestLoadJournalMissing(t *testing.T) {
	entries, err := LoadJournal("/nonexistent/journal.jsonl")
	if err != nil {
		t.Errorf("expected no error for missing file, got: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries for missing file")
	}
}

func TestLoadJournalMaster(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "master.jsonl")
	data := `{"ts":"2026-03-19T10:00:00Z","action":"check","session":"session-1","finding":"15/23 tests pass"}
{"ts":"2026-03-19T10:05:00Z","action":"guide","session":"session-1","guidance":"fix imports"}
`
	os.WriteFile(path, []byte(data), 0644)

	entries, err := LoadJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
	if entries[0].Action != "check" || entries[1].Guidance != "fix imports" {
		t.Errorf("entries = %+v", entries)
	}
}

func TestSummary(t *testing.T) {
	entries := []JournalEntry{
		{Round: 1, Desc: "read code", Status: "progress"},
		{Round: 2, Desc: "all pass", Status: "done"},
	}
	s := Summary(entries)
	if s != "round 2: all pass (done)" {
		t.Errorf("Summary = %q", s)
	}
}

func TestSummaryEmpty(t *testing.T) {
	s := Summary(nil)
	if s != "no entries" {
		t.Errorf("Summary(nil) = %q", s)
	}
}

func TestSummaryMaster(t *testing.T) {
	entries := []JournalEntry{
		{Action: "check", Session: "session-1", Finding: "15/23 pass"},
	}
	s := Summary(entries)
	if s != "[check] session-1: 15/23 pass" {
		t.Errorf("Summary = %q", s)
	}
}
