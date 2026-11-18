package pylonimport

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// PylonUIReferenceToleranceSec is the default ± match window when comparing to Pylon chart tooltips.
const PylonUIReferenceToleranceSec = 120.0

// PylonUIReference is a Workforce chart tooltip we want hypotheses to reproduce.
type PylonUIReference struct {
	WeekLabel            string
	Timezone             string
	MedianFirstResponseS float64
	MedianResolutionS    float64
	FRToleranceS         float64 // 0 = exact
	ResToleranceS        float64
}

// SLAValueSource picks which duration field to aggregate for FR or resolution.
type SLAValueSource int

const (
	FRCalendarFromFirstResponse SLAValueSource = iota
	FRAPIFirstResponseSeconds
	FRPreferAPIElseCalendar
	FRCalendarExcludingAutoReply4h
	FRAPIOnlyWhenClosed
	FRCalendarOnlyWhenResolved
	ResAPIResolutionSeconds
	ResCalendarFromResolution
	ResPreferAPIElseCalendar
)

// SLAUniverseFilter limits which issues enter the median for a week bucket.
type SLAUniverseFilter int

const (
	UniverseAllInWeek SLAUniverseFilter = iota
	UniverseClosedOnly
	UniverseHasFRSeconds
	UniverseHasResSeconds
	UniverseClosedWithFRSeconds
	UniverseClosedWithResSeconds
	UniverseExcludeWaiting
	UniverseEmailSourceOnly
)

// SLAHypothesis is one candidate Pylon Workforce rule (bucket + filter + sources).
type SLAHypothesis struct {
	Name         string
	Bucket       SLABucket
	Universe     SLAUniverseFilter
	FRSource     SLAValueSource
	ResSource    SLAValueSource
	MinFRSeconds float64 // optional lower bound on FR value to count (0 = none)
	MaxFRSeconds float64 // optional upper bound (0 = none)
}

// SLAHypothesisResult holds medians and counts for one hypothesis on one week.
type SLAHypothesisResult struct {
	Hypothesis  SLAHypothesis
	WeekIssues  int
	FRN         int
	ResN        int
	MedianFRS   float64
	MedianResS  float64
	MatchesFR   bool
	MatchesRes  bool
	MatchesBoth bool
	FRDeltaS    float64
	ResDeltaS   float64
}

// LockedPylonResolutionHypothesis is the Workforce resolution rule that matches Pylon cache (see sla_hypothesis_test.go).
func LockedPylonResolutionHypothesis() SLAHypothesis {
	return SLAHypothesis{
		Name:      "locked_resolution",
		Bucket:    BucketByCreatedAt,
		Universe:  UniverseHasResSeconds,
		FRSource:  FRCalendarFromFirstResponse,
		ResSource: ResAPIResolutionSeconds,
	}
}

// DefaultPylonHypothesisGrid is the search space for discovering Pylon-style rules.
// Used only by offline tests and --pylon-sla-reconcile --pylon-reconcile-discover (not production runtime).
func DefaultPylonHypothesisGrid() []SLAHypothesis {
	buckets := []SLABucket{BucketByCreatedAt, BucketByFirstReply, BucketByResolution}
	universes := []SLAUniverseFilter{
		UniverseAllInWeek,
		UniverseClosedOnly,
		UniverseHasFRSeconds,
		UniverseHasResSeconds,
		UniverseClosedWithFRSeconds,
		UniverseClosedWithResSeconds,
		UniverseExcludeWaiting,
		UniverseEmailSourceOnly,
	}
	frSources := []SLAValueSource{
		FRCalendarFromFirstResponse,
		FRAPIFirstResponseSeconds,
		FRPreferAPIElseCalendar,
		FRCalendarExcludingAutoReply4h,
		FRAPIOnlyWhenClosed,
	}
	resSources := []SLAValueSource{
		ResAPIResolutionSeconds,
		ResCalendarFromResolution,
		ResPreferAPIElseCalendar,
	}

	var out []SLAHypothesis
	for _, b := range buckets {
		for _, u := range universes {
			for _, fr := range frSources {
				for _, res := range resSources {
					name := fmt.Sprintf("bucket=%s universe=%s fr=%s res=%s",
						b, universeLabel(u), frSourceLabel(fr), resSourceLabel(res))
					out = append(out, SLAHypothesis{
						Name:      name,
						Bucket:    b,
						Universe:  u,
						FRSource:  fr,
						ResSource: res,
					})
				}
			}
		}
	}
	// Extra: exclude known 4h auto-ack band explicitly on created_at + calendar FR.
	out = append(out, SLAHypothesis{
		Name:         "created_at|all|calendar_fr>6h|res_api",
		Bucket:       BucketByCreatedAt,
		Universe:     UniverseAllInWeek,
		FRSource:     FRCalendarFromFirstResponse,
		ResSource:    ResAPIResolutionSeconds,
		MinFRSeconds: 6 * 3600,
	})
	out = append(out, SLAHypothesis{
		Name:         "created_at|closed|calendar_fr>6h|res_api",
		Bucket:       BucketByCreatedAt,
		Universe:     UniverseClosedOnly,
		FRSource:     FRCalendarFromFirstResponse,
		ResSource:    ResAPIResolutionSeconds,
		MinFRSeconds: 6 * 3600,
	})
	return out
}

// EvaluateHypothesis computes medians for issues matching weekLabel under hyp.
func EvaluateHypothesis(issues []Issue, ref PylonUIReference, hyp SLAHypothesis) (SLAHypothesisResult, error) {
	var frVals, resVals []float64
	weekIssues := 0
	for _, is := range issues {
		if !universeOK(is, hyp.Universe) {
			continue
		}
		anchor, ok := bucketAnchor(is, hyp.Bucket)
		if !ok {
			continue
		}
		wk, err := pgWeekLabel(anchor, ref.Timezone)
		if err != nil {
			return SLAHypothesisResult{}, err
		}
		if wk != ref.WeekLabel {
			continue
		}
		weekIssues++
		if v, ok := extractFR(is, hyp.FRSource); ok {
			includeFR := true
			if hyp.MinFRSeconds > 0 && v < hyp.MinFRSeconds {
				includeFR = false
			}
			if hyp.MaxFRSeconds > 0 && v > hyp.MaxFRSeconds {
				includeFR = false
			}
			if includeFR {
				frVals = append(frVals, v)
			}
		}
		if v, ok := extractRes(is, hyp.ResSource); ok {
			resVals = append(resVals, v)
		}
	}

	medFR := medianFloat64(frVals)
	medRes := medianFloat64(resVals)

	res := SLAHypothesisResult{
		Hypothesis: hyp,
		WeekIssues: weekIssues,
		FRN:        len(frVals),
		ResN:       len(resVals),
		MedianFRS:  medFR,
		MedianResS: medRes,
		FRDeltaS:   deltaSec(medFR, ref.MedianFirstResponseS),
		ResDeltaS:  deltaSec(medRes, ref.MedianResolutionS),
	}
	if ref.MedianFirstResponseS > 0 && !math.IsNaN(medFR) {
		res.MatchesFR = res.FRDeltaS <= ref.FRToleranceS
	}
	if ref.MedianResolutionS > 0 && !math.IsNaN(medRes) {
		res.MatchesRes = res.ResDeltaS <= ref.ResToleranceS
	}
	res.MatchesBoth = res.MatchesFR && res.MatchesRes
	return res, nil
}

// SearchPylonHypotheses runs every hypothesis and returns those that match the UI reference.
func SearchPylonHypotheses(issues []Issue, ref PylonUIReference, grid []SLAHypothesis) ([]SLAHypothesisResult, error) {
	var matches []SLAHypothesisResult
	for _, hyp := range grid {
		r, err := EvaluateHypothesis(issues, ref, hyp)
		if err != nil {
			return nil, err
		}
		if r.MatchesFR || r.MatchesRes {
			matches = append(matches, r)
		}
	}
	return matches, nil
}

func bucketAnchor(is Issue, bucket SLABucket) (time.Time, bool) {
	switch bucket {
	case BucketByCreatedAt:
		return is.CreatedAt, true
	case BucketByFirstReply:
		if is.FirstResponseTime != nil {
			return *is.FirstResponseTime, true
		}
		if is.FirstResponseBreachTime != nil {
			return *is.FirstResponseBreachTime, true
		}
		return time.Time{}, false
	case BucketByResolution:
		if is.ResolutionTime != nil {
			return *is.ResolutionTime, true
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}

func universeOK(is Issue, u SLAUniverseFilter) bool {
	switch u {
	case UniverseAllInWeek:
		return true
	case UniverseClosedOnly:
		return is.State == "closed"
	case UniverseHasFRSeconds:
		return is.FirstResponseSeconds != nil
	case UniverseHasResSeconds:
		return is.ResolutionSeconds != nil
	case UniverseClosedWithFRSeconds:
		return is.State == "closed" && is.FirstResponseSeconds != nil
	case UniverseClosedWithResSeconds:
		return is.State == "closed" && is.ResolutionSeconds != nil
	case UniverseExcludeWaiting:
		return !strings.HasPrefix(is.State, "waiting")
	case UniverseEmailSourceOnly:
		return is.Source == "email" || is.Source == "form"
	default:
		return true
	}
}

const autoReplyFRLow = 3.5 * 3600
const autoReplyFRHigh = 4.5 * 3600

func extractFR(is Issue, src SLAValueSource) (float64, bool) {
	switch src {
	case FRAPIFirstResponseSeconds, FRAPIOnlyWhenClosed:
		if is.FirstResponseSeconds == nil {
			return 0, false
		}
		if src == FRAPIOnlyWhenClosed && is.State != "closed" {
			return 0, false
		}
		return float64(*is.FirstResponseSeconds), true
	case FRPreferAPIElseCalendar:
		if is.FirstResponseSeconds != nil {
			return float64(*is.FirstResponseSeconds), true
		}
		return calendarFREnd(is)
	case FRCalendarExcludingAutoReply4h:
		v, ok := calendarFREnd(is)
		if !ok {
			return 0, false
		}
		if v >= autoReplyFRLow && v <= autoReplyFRHigh {
			return 0, false
		}
		return v, true
	case FRCalendarOnlyWhenResolved:
		if is.State != "closed" || is.ResolutionTime == nil {
			return 0, false
		}
		return calendarFREnd(is)
	default:
		return calendarFREnd(is)
	}
}

func calendarFREnd(is Issue) (float64, bool) {
	end := is.FirstResponseTime
	if end == nil {
		end = is.FirstResponseBreachTime
	}
	if end == nil {
		return 0, false
	}
	return calendarSec(is.CreatedAt, *end), true
}

func extractRes(is Issue, src SLAValueSource) (float64, bool) {
	switch src {
	case ResAPIResolutionSeconds:
		if is.ResolutionSeconds == nil {
			return 0, false
		}
		return float64(*is.ResolutionSeconds), true
	case ResPreferAPIElseCalendar:
		if is.ResolutionSeconds != nil {
			return float64(*is.ResolutionSeconds), true
		}
		if is.ResolutionTime == nil {
			return 0, false
		}
		return calendarSec(is.CreatedAt, *is.ResolutionTime), true
	default:
		if is.ResolutionTime == nil {
			return 0, false
		}
		return calendarSec(is.CreatedAt, *is.ResolutionTime), true
	}
}

func deltaSec(got, want float64) float64 {
	if math.IsNaN(got) || want <= 0 {
		return math.MaxFloat64
	}
	d := got - want
	if d < 0 {
		return -d
	}
	return d
}

func universeLabel(u SLAUniverseFilter) string {
	switch u {
	case UniverseAllInWeek:
		return "all"
	case UniverseClosedOnly:
		return "closed"
	case UniverseHasFRSeconds:
		return "has_fr_sec"
	case UniverseHasResSeconds:
		return "has_res_sec"
	case UniverseClosedWithFRSeconds:
		return "closed+fr_sec"
	case UniverseClosedWithResSeconds:
		return "closed+res_sec"
	case UniverseExcludeWaiting:
		return "not_waiting"
	case UniverseEmailSourceOnly:
		return "email"
	default:
		return "?"
	}
}

func frSourceLabel(s SLAValueSource) string {
	switch s {
	case FRAPIFirstResponseSeconds:
		return "api_fr_sec"
	case FRPreferAPIElseCalendar:
		return "api_or_cal_fr"
	case FRCalendarExcludingAutoReply4h:
		return "cal_fr_no_4h"
	case FRAPIOnlyWhenClosed:
		return "api_fr_closed"
	case FRCalendarOnlyWhenResolved:
		return "cal_fr_closed_res"
	default:
		return "cal_fr"
	}
}

func resSourceLabel(s SLAValueSource) string {
	switch s {
	case ResAPIResolutionSeconds:
		return "api_res_sec"
	case ResPreferAPIElseCalendar:
		return "api_or_cal_res"
	default:
		return "cal_res"
	}
}
