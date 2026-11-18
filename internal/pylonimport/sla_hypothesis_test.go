package pylonimport

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// nov9Workforce2026 is the Pylon Workforce hover observed for week label 2026-11-09 (America/New_York).
func nov9Workforce2026() PylonUIReference {
	return PylonUIReference{
		WeekLabel:            "2026-11-09",
		Timezone:             "America/New_York",
		MedianFirstResponseS: 19*3600 + 15*60, // 69300
		MedianResolutionS:    22*3600 + 53*60, // 82380
		FRToleranceS:         PylonUIReferenceToleranceSec,
		ResToleranceS:        PylonUIReferenceToleranceSec,
	}
}

func loadIssuesForHypothesisTest(t *testing.T) []Issue {
	t.Helper()
	if p := os.Getenv("PYLON_ISSUES_JSON"); p != "" {
		issues, err := LoadIssuesJSON(p)
		if err != nil {
			t.Fatal(err)
		}
		return issues
	}
	cache := filepath.Join("..", "..", ".scratch", "pylon-issues.json")
	if _, err := os.Stat(cache); err == nil {
		issues, err := LoadIssuesJSON(cache)
		if err != nil {
			t.Fatal(err)
		}
		return issues
	}
	issues, err := LoadIssuesJSON(filepath.Join("testdata", "pylon_hypothesis_week.json"))
	if err != nil {
		t.Fatal(err)
	}
	return issues
}

func TestLockedPylonResolutionSpecFixtureAlways(t *testing.T) {
	ref := nov9Workforce2026()
	issues, err := LoadIssuesJSON(filepath.Join("testdata", "pylon_hypothesis_week.json"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := EvaluateHypothesis(issues, ref, LockedPylonResolutionHypothesis())
	if err != nil {
		t.Fatal(err)
	}
	if r.ResN != 2 || !r.MatchesRes {
		t.Fatalf("fixture locked resolution: res_n=%d median=%s match=%v", r.ResN, FormatDuration(r.MedianResS), r.MatchesRes)
	}
}

func TestPylonHypothesisResolutionAPIClosedCreatedWeek(t *testing.T) {
	// Documented winner from production cache: resolution_seconds, created_at week, closed-only is NOT required.
	ref := nov9Workforce2026()
	issues, err := LoadIssuesJSON(filepath.Join("testdata", "pylon_hypothesis_week.json"))
	if err != nil {
		t.Fatal(err)
	}
	hyp := SLAHypothesis{
		Name:      "fixture: created_at + res_api (subset)",
		Bucket:    BucketByCreatedAt,
		Universe:  UniverseHasResSeconds,
		FRSource:  FRCalendarFromFirstResponse,
		ResSource: ResAPIResolutionSeconds,
	}
	r, err := EvaluateHypothesis(issues, ref, hyp)
	if err != nil {
		t.Fatal(err)
	}
	if r.ResN != 2 {
		t.Fatalf("res n = %d, want 2", r.ResN)
	}
	// Fixture medians differ from full week; this test only checks machinery.
	if mathIsNaN(r.MedianResS) {
		t.Fatal("expected resolution median")
	}
}

func TestDiscoverPylonWorkforceHypotheses(t *testing.T) {
	if os.Getenv("PYLON_HYPOTHESIS_DISCOVER") != "1" {
		t.Skip("set PYLON_HYPOTHESIS_DISCOVER=1 with .scratch/pylon-issues.json for full grid search")
	}
	issues := loadIssuesForHypothesisTest(t)
	ref := nov9Workforce2026()
	usingFixture := len(issues) <= 10
	if usingFixture {
		ref.FRToleranceS = 3600
		ref.ResToleranceS = 3600
	}

	grid := DefaultPylonHypothesisGrid()
	results, err := SearchPylonHypotheses(issues, ref, grid)
	if err != nil {
		t.Fatal(err)
	}

	var both, frOnly, resOnly []SLAHypothesisResult
	for _, r := range results {
		switch {
		case r.MatchesBoth:
			both = append(both, r)
		case r.MatchesFR:
			frOnly = append(frOnly, r)
		case r.MatchesRes:
			resOnly = append(resOnly, r)
		}
	}
	sort.Slice(both, func(i, j int) bool { return both[i].FRDeltaS+both[i].ResDeltaS < both[j].FRDeltaS+both[j].ResDeltaS })
	sort.Slice(frOnly, func(i, j int) bool { return frOnly[i].FRDeltaS < frOnly[j].FRDeltaS })
	sort.Slice(resOnly, func(i, j int) bool { return resOnly[i].ResDeltaS < resOnly[j].ResDeltaS })

	printHypothesisResults("MATCH FR+RES", both, 15)
	printHypothesisResults("MATCH FR only", frOnly, 10)
	printHypothesisResults("MATCH RES only", resOnly, 10)

	// Locked expectation from real cache analysis: at least one rule reproduces resolution.
	if !usingFixture {
		if len(resOnly) == 0 && len(both) == 0 {
			t.Fatal("expected at least one hypothesis to match Pylon resolution 22h53m on full cache")
		}
		foundRes := false
		for _, r := range append(both, resOnly...) {
			if r.Hypothesis.Bucket == BucketByCreatedAt &&
				r.Hypothesis.ResSource == ResAPIResolutionSeconds &&
				r.Hypothesis.Universe == UniverseHasResSeconds &&
				r.MatchesRes {
				foundRes = true
				break
			}
		}
		if !foundRes {
			t.Log("WARN: expected created_at + has_res_seconds + api_res_sec; check grid or tolerance")
		}
	}
}

func TestLockedPylonResolutionSpec(t *testing.T) {
	issues := loadIssuesForHypothesisTest(t)
	if len(issues) < 20 {
		t.Skip("needs full pylon cache")
	}
	ref := nov9Workforce2026()
	r, err := EvaluateHypothesis(issues, ref, LockedPylonResolutionHypothesis())
	if err != nil {
		t.Fatal(err)
	}
	if r.ResN != 9 || !r.MatchesRes {
		t.Fatalf("locked resolution: res_n=%d median=%s match=%v", r.ResN, FormatDuration(r.MedianResS), r.MatchesRes)
	}
}

func TestPylonHypothesisKnownResolutionRule(t *testing.T) {
	issues := loadIssuesForHypothesisTest(t)
	if len(issues) < 20 {
		t.Skip("needs full pylon cache for resolution lock-in")
	}
	ref := nov9Workforce2026()
	hyp := SLAHypothesis{
		Name:      "pylon_resolution_rule",
		Bucket:    BucketByCreatedAt,
		Universe:  UniverseHasResSeconds,
		FRSource:  FRCalendarFromFirstResponse, // FR not asserted here
		ResSource: ResAPIResolutionSeconds,
	}
	r, err := EvaluateHypothesis(issues, ref, hyp)
	if err != nil {
		t.Fatal(err)
	}
	if !r.MatchesRes {
		t.Fatalf("resolution rule failed: got %s (%ds), want %s (±%.0fs), n=%d",
			FormatDuration(r.MedianResS), int(r.MedianResS),
			FormatDuration(ref.MedianResolutionS), ref.ResToleranceS, r.ResN)
	}
	if r.ResN != 9 {
		t.Fatalf("res n = %d, want 9 (Pylon issues with resolution_seconds that week)", r.ResN)
	}
}

// TestPylonHypothesisNonDegenerateFR searches FR rules that match Pylon while using at least 3 samples
// (avoids false positives where the median is a single issue, e.g. #4007 at 19h15m).
func TestPylonHypothesisNonDegenerateFR(t *testing.T) {
	issues := loadIssuesForHypothesisTest(t)
	if len(issues) < 20 {
		t.Skip("needs full pylon cache")
	}
	ref := nov9Workforce2026()
	const minN = 3
	var best SLAHypothesisResult
	for _, hyp := range DefaultPylonHypothesisGrid() {
		r, err := EvaluateHypothesis(issues, ref, hyp)
		if err != nil {
			t.Fatal(err)
		}
		if r.FRN < minN {
			continue
		}
		if r.FRDeltaS < ref.FRToleranceS && (best.FRN == 0 || r.FRDeltaS < best.FRDeltaS) {
			best = r
		}
	}
	if best.FRN == 0 {
		t.Log("no non-degenerate hypothesis (fr_n>=3) matched 19h15m within 2m — see closest candidates:")
		for _, h := range []SLAHypothesis{
			{Name: "cal_no_4h", Bucket: BucketByCreatedAt, Universe: UniverseAllInWeek, FRSource: FRCalendarExcludingAutoReply4h, ResSource: ResAPIResolutionSeconds},
			{Name: "cal_gt_6h", Bucket: BucketByCreatedAt, Universe: UniverseAllInWeek, FRSource: FRCalendarFromFirstResponse, ResSource: ResAPIResolutionSeconds, MinFRSeconds: 6 * 3600},
			{Name: "cal_gt_6h_closed", Bucket: BucketByCreatedAt, Universe: UniverseClosedOnly, FRSource: FRCalendarFromFirstResponse, ResSource: ResAPIResolutionSeconds, MinFRSeconds: 6 * 3600},
		} {
			r, _ := EvaluateHypothesis(issues, ref, h)
			t.Logf("  %s: FR=%s n=%d Δ=%.0fs", h.Name, FormatDuration(r.MedianFRS), r.FRN, r.FRDeltaS)
		}
		return
	}
	t.Logf("non-degenerate FR match: %s FR=%s fr_n=%d Δ=%.0fs", best.Hypothesis.Name, FormatDuration(best.MedianFRS), best.FRN, best.FRDeltaS)
}

func TestPylonHypothesisDegenerateFRAPIClosed(t *testing.T) {
	// Documents why api_fr_closed can "match" 19h15m: only one closed issue has first_response_seconds that week.
	issues := loadIssuesForHypothesisTest(t)
	if len(issues) < 20 {
		t.Skip("needs full pylon cache")
	}
	ref := nov9Workforce2026()
	hyp := SLAHypothesis{
		Name:      "closed+api_fr_sec",
		Bucket:    BucketByCreatedAt,
		Universe:  UniverseClosedWithFRSeconds,
		FRSource:  FRAPIFirstResponseSeconds,
		ResSource: ResAPIResolutionSeconds,
	}
	r, err := EvaluateHypothesis(issues, ref, hyp)
	if err != nil {
		t.Fatal(err)
	}
	if r.FRN != 1 {
		t.Fatalf("expected fr_n=1 (only #4007), got %d", r.FRN)
	}
	if deltaSec(r.MedianFRS, ref.MedianFirstResponseS) > ref.FRToleranceS {
		t.Fatalf("median %s (%.0fs), want %s", FormatDuration(r.MedianFRS), r.MedianFRS, FormatDuration(ref.MedianFirstResponseS))
	}
}

func TestPylonHypothesisFirstResponseCandidates(t *testing.T) {
	issues := loadIssuesForHypothesisTest(t)
	if len(issues) < 20 {
		t.Skip("needs full pylon cache")
	}
	ref := nov9Workforce2026()

	// FR is the open problem: no hypothesis in the default grid matched 19h15m in discovery.
	// These tests document the closest rules and guard regressions when we add one.
	candidates := []SLAHypothesis{
		{
			Name:         "cal_fr_gt_6h",
			Bucket:       BucketByCreatedAt,
			Universe:     UniverseAllInWeek,
			FRSource:     FRCalendarFromFirstResponse,
			ResSource:    ResAPIResolutionSeconds,
			MinFRSeconds: 6 * 3600,
		},
		{
			Name:      "cal_fr_no_4h_band",
			Bucket:    BucketByCreatedAt,
			Universe:  UniverseAllInWeek,
			FRSource:  FRCalendarExcludingAutoReply4h,
			ResSource: ResAPIResolutionSeconds,
		},
		{
			Name:      "api_fr_sec",
			Bucket:    BucketByCreatedAt,
			Universe:  UniverseHasFRSeconds,
			FRSource:  FRAPIFirstResponseSeconds,
			ResSource: ResAPIResolutionSeconds,
		},
		{
			Name:      "closed+api_fr",
			Bucket:    BucketByCreatedAt,
			Universe:  UniverseClosedWithFRSeconds,
			FRSource:  FRAPIFirstResponseSeconds,
			ResSource: ResAPIResolutionSeconds,
		},
		{
			Name:      "fr_event_week_cal",
			Bucket:    BucketByFirstReply,
			Universe:  UniverseAllInWeek,
			FRSource:  FRCalendarFromFirstResponse,
			ResSource: ResAPIResolutionSeconds,
		},
	}

	var best SLAHypothesisResult
	for _, h := range candidates {
		r, err := EvaluateHypothesis(issues, ref, h)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("candidate %s: FR=%s (n=%d Δ=%.0fs) RES=%s (n=%d Δ=%.0fs)",
			h.Name, FormatDuration(r.MedianFRS), r.FRN, r.FRDeltaS,
			FormatDuration(r.MedianResS), r.ResN, r.ResDeltaS)
		if r.FRN > 0 && (best.FRN == 0 || r.FRDeltaS < best.FRDeltaS) {
			best = r
		}
	}

	if best.FRN == 0 {
		t.Fatal("no FR candidate produced values")
	}
	// Known gap: best delta is still >> 2m until Pylon FR spec is found.
	if best.FRN == 1 {
		t.Log("closest FR match is degenerate (fr_n=1); see TestPylonHypothesisDegenerateFRAPIClosed")
	}
	if best.FRDeltaS < ref.FRToleranceS && best.FRN > 1 {
		t.Logf("FOUND FR match: %s", best.Hypothesis.Name)
	} else {
		t.Logf("No candidate within %.0fs of Pylon FR; closest %s at Δ=%.0fs (%s vs %s)",
			ref.FRToleranceS, best.Hypothesis.Name, best.FRDeltaS,
			FormatDuration(best.MedianFRS), FormatDuration(ref.MedianFirstResponseS))
	}
}

func printHypothesisResults(title string, list []SLAHypothesisResult, limit int) {
	if len(list) == 0 {
		fmt.Printf("\n%s: (none)\n", title)
		return
	}
	fmt.Printf("\n%s (%d hits, showing %d):\n", title, len(list), min(limit, len(list)))
	for i, r := range list {
		if i >= limit {
			break
		}
		fmt.Printf("  [%s] week_issues=%d fr_n=%d res_n=%d FR=%s (Δ%.0fs) RES=%s (Δ%.0fs)\n",
			r.Hypothesis.Name, r.WeekIssues, r.FRN, r.ResN,
			FormatDuration(r.MedianFRS), r.FRDeltaS,
			FormatDuration(r.MedianResS), r.ResDeltaS)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func mathIsNaN(f float64) bool {
	return f != f
}
