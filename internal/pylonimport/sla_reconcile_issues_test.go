package pylonimport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWeekIssuesNov9Detail(t *testing.T) {
	path := filepath.Join("..", "..", ".scratch", "pylon-issues.json")
	if _, err := os.Stat(path); err != nil {
		t.Skip(path)
	}
	issues, err := LoadIssuesJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	const week = "2026-11-09"
	tz := "America/New_York"
	rows, err := weekIssuesFromCache(issues, week, tz)
	if err != nil {
		t.Fatal(err)
	}
	PrintWeekIssueTable(rows, week)
	if len(rows) < 1 {
		t.Fatal("expected issues in week")
	}
}