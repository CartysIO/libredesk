package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/pylonimport"
)

// runPylonSLAReconcile prints Pylon Workforce parity analysis from --pylon-cache issues.json.
func runPylonSLAReconcile() {
	week := ko.String("pylon-reconcile-week")
	if week == "" {
		log.Fatal("--pylon-reconcile-week required (YYYY-MM-DD week label, e.g. 2026-11-09)")
	}
	tz := ko.String("pylon-reconcile-timezone")
	if tz == "" {
		tz = "America/New_York"
	}
	cachePath := ko.String("pylon-cache")
	if cachePath == "" {
		cachePath = "pylon-cache"
	}
	issuesPath := cachePath
	if !strings.HasSuffix(issuesPath, ".json") {
		issuesPath = cachePath + "/issues.json"
	}
	issues, err := pylonimport.LoadIssuesJSON(issuesPath)
	if err != nil {
		log.Fatalf("load issues: %v", err)
	}

	ref := pylonimport.PylonUIReference{
		WeekLabel:            week,
		Timezone:             tz,
		MedianFirstResponseS: parsePylonDuration(ko.String("pylon-reconcile-fr-target")),
		MedianResolutionS:    parsePylonDuration(ko.String("pylon-reconcile-res-target")),
		FRToleranceS:         pylonimport.PylonUIReferenceToleranceSec,
		ResToleranceS:        pylonimport.PylonUIReferenceToleranceSec,
	}
	targets := pylonimport.PylonChartTargets{
		MedianFirstResponseSec: ref.MedianFirstResponseS,
		MedianResolutionSec:    ref.MedianResolutionS,
	}
	if ref.MedianFirstResponseS > 0 {
		targets.HaveFR = true
	}
	if ref.MedianResolutionS > 0 {
		targets.HaveRes = true
	}

	cache, counts, err := pylonimport.ReconcileFromPylonCache(issues, week, tz)
	if err != nil {
		log.Fatal(err)
	}
	report := pylonimport.SLAReconcileWeek{
		Week:         week,
		Timezone:     tz,
		PylonTargets: targets,
		PylonCache:   cache,
		Counts:       counts,
	}
	pylonimport.PrintReconcileReport(report)
	pylonimport.ReconcileWeekSweep(issues, week, tz)

	rows, err := pylonimport.WeekIssuesFromCache(issues, week, tz)
	if err != nil {
		log.Fatal(err)
	}
	pylonimport.PrintWeekIssueTable(rows, week)

	locked := pylonimport.LockedPylonResolutionHypothesis()
	r, err := pylonimport.EvaluateHypothesis(issues, ref, locked)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nLocked resolution hypothesis: %s\n  median_res=%s res_n=%d match=%v\n",
		locked.Name, pylonimport.FormatDuration(r.MedianResS), r.ResN, r.MatchesRes)

	if ko.Bool("pylon-reconcile-discover") {
		grid := pylonimport.DefaultPylonHypothesisGrid()
		matches, err := pylonimport.SearchPylonHypotheses(issues, ref, grid)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\nDiscover: %d hypotheses matched (±2m)\n", len(matches))
		for i, m := range matches {
			if i >= 20 {
				break
			}
			fmt.Printf("  %s fr_n=%d res_n=%d FR=%s RES=%s\n",
				m.Hypothesis.Name, m.FRN, m.ResN,
				pylonimport.FormatDuration(m.MedianFRS), pylonimport.FormatDuration(m.MedianResS))
		}
	}
}

func parsePylonDuration(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if sec, err := strconv.ParseFloat(s, 64); err == nil {
		return sec
	}
	// "19h15m" style
	var h, m int
	if _, err := fmt.Sscanf(s, "%dh%dm", &h, &m); err == nil {
		return float64(h*3600 + m*60)
	}
	return 0
}
