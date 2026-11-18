package pylonimport

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestReconcileWeekNov2026Production(t *testing.T) {
	path := os.Getenv("PYLON_ISSUES_JSON")
	if path == "" {
		path = filepath.Join("..", "..", ".scratch", "pylon-issues.json")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no issues cache at %s: %v", path, err)
	}
	issues, err := LoadIssuesJSON(path)
	if err != nil {
		t.Fatal(err)
	}

	const week = "2026-11-09"
	tz := os.Getenv("APP_TIMEZONE")
	if tz == "" {
		tz = "America/New_York"
	}
	targets := PylonChartTargets{
		MedianFirstResponseSec: 19*3600 + 15*60,
		MedianResolutionSec:    22*3600 + 53*60,
		HaveFR:                 true,
		HaveRes:                true,
	}
	if v := os.Getenv("APP_TIMEZONE"); v != "" {
		// allow override when reconciling other envs
		_ = v
	}

	cache, counts, err := ReconcileFromPylonCache(issues, week, tz)
	if err != nil {
		t.Fatal(err)
	}

	report := SLAReconcileWeek{
		Week:         week,
		Timezone:     tz,
		PylonTargets: targets,
		PylonCache:   cache,
		Counts:       counts,
	}
	PrintReconcileReport(report)
	ReconcileWeekSweep(issues, week, tz)

	// Sanity: we expect data in that week from production import.
	if counts.CacheInWeek < 1 {
		t.Fatalf("expected issues in week %s, got %d", week, counts.CacheInWeek)
	}
}

func TestReconcileWeekSweepOctNov2026(t *testing.T) {
	path := filepath.Join("..", "..", ".scratch", "pylon-issues.json")
	if _, err := os.Stat(path); err != nil {
		t.Skip(path)
	}
	issues, err := LoadIssuesJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	tz := "America/New_York"
	targetFR := float64(19*3600 + 15*60)
	weeks := []string{"2026-10-05", "2026-10-12", "2026-10-19", "2026-10-26", "2026-11-02", "2026-11-09", "2026-11-16"}
	fmt.Println("Scanning created_at weeks for FR calendar median near Pylon 19h15m:")
	for _, week := range weeks {
		cache, counts, err := ReconcileFromPylonCache(issues, week, tz)
		if err != nil {
			t.Fatal(err)
		}
		if counts.CacheFRCalendar < 1 {
			continue
		}
		delta := absDiff(cache.CalendarFR, targetFR)
		fmt.Printf("  %s n=%d cal_FR=%s pylon_sec=%s res_pylon=%s delta_FR=%.0fs\n",
			week, counts.CacheInWeek, FormatDuration(cache.CalendarFR), FormatDuration(cache.PylonFRSeconds),
			FormatDuration(cache.PylonResSeconds), delta)
	}
}