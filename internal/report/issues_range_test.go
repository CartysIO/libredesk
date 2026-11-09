package report

import (
	"testing"
	"time"
)

func TestResolveIssuesRangeBoundsPresetDays(t *testing.T) {
	b, err := ResolveIssuesRangeBounds(IssuesRangeInput{Days: 30, AppTimezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	if !b.EndsAt.After(b.StartsAt) {
		t.Fatalf("ends before start: %+v", b)
	}
	if b.EndsAt.Sub(b.StartsAt) < 29*24*time.Hour {
		t.Fatalf("window too short: %v", b.EndsAt.Sub(b.StartsAt))
	}
}

func TestResolveIssuesRangeBoundsCustom(t *testing.T) {
	b, err := ResolveIssuesRangeBounds(IssuesRangeInput{
		StartDate:   "2026-01-01",
		EndDate:     "2026-01-07",
		AppTimezone: "America/Chicago",
	})
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("America/Chicago")
	wantStart := time.Date(2026, 1, 1, 0, 0, 0, 0, loc).UTC()
	wantEnd := time.Date(2026, 1, 8, 0, 0, 0, 0, loc).UTC()
	if !b.StartsAt.Equal(wantStart) || !b.EndsAt.Equal(wantEnd) {
		t.Fatalf("bounds = %+v, want start %v end %v", b, wantStart, wantEnd)
	}
}

func TestResolveIssuesRangeBoundsRequiresPair(t *testing.T) {
	_, err := ResolveIssuesRangeBounds(IssuesRangeInput{StartDate: "2026-01-01", AppTimezone: "UTC"})
	if err == nil {
		t.Fatal("expected error for partial custom range")
	}
}
