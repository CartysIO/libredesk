package pylonimport

import (
	"testing"
	"time"
)

func TestIssuesNotBefore(t *testing.T) {
	cutoff := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	issues := []Issue{
		{ID: "old", CreatedAt: cutoff.Add(-time.Hour)},
		{ID: "ok", CreatedAt: cutoff},
		{ID: "new", CreatedAt: cutoff.Add(24 * time.Hour)},
	}
	got := issuesNotBefore(issues, cutoff)
	if len(got) != 2 {
		t.Fatalf("want 2 issues, got %d", len(got))
	}
	if got[0].ID != "ok" || got[1].ID != "new" {
		t.Fatalf("got ids %q, %q", got[0].ID, got[1].ID)
	}
}

func TestIssueFetchWindowsSinceCutoff(t *testing.T) {
	oldest := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	windows := issueFetchWindows(oldest, end)
	if len(windows) != 1 {
		t.Fatalf("want 1 window, got %d", len(windows))
	}
	if !windows[0].Start.Equal(oldest) {
		t.Fatalf("start: got %s want %s", windows[0].Start, oldest)
	}
	if !windows[0].End.Equal(end) {
		t.Fatalf("end: got %s want %s", windows[0].End, end)
	}
}

func TestIssueFetchWindowsFullHistoryUses360DayChunks(t *testing.T) {
	oldest := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	windows := issueFetchWindows(oldest, end)
	if len(windows) < 2 {
		t.Fatalf("want multiple windows, got %d", len(windows))
	}
	// Newest window first.
	if !windows[0].End.Equal(end) {
		t.Fatalf("first window end: got %s want %s", windows[0].End, end)
	}
	last := windows[len(windows)-1]
	if last.Start.Before(oldest) {
		t.Fatalf("last start %s before oldest %s", last.Start, oldest)
	}
	if !last.Start.Equal(oldest) {
		t.Fatalf("last start: got %s want %s", last.Start, oldest)
	}
}
