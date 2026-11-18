package pylonimport

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"time"
)

// SLAReconcileWeek compares median SLA durations for one Workforce week bucket.
type SLAReconcileWeek struct {
	Week           string
	Timezone       string
	PylonTargets   PylonChartTargets
	PylonCache     CacheMedians
	LibredeskDB    *DBMedians
	Counts         ReconcileCounts
}

// PylonChartTargets is the reference from Pylon UI (optional).
type PylonChartTargets struct {
	MedianFirstResponseSec float64
	MedianResolutionSec    float64
	HaveFR                 bool
	HaveRes                bool
}

// CacheMedians are medians computed from issues.json (Pylon API export).
type CacheMedians struct {
	PylonFRSeconds   float64
	PylonResSeconds  float64
	CalendarFR       float64
	CalendarRes      float64
	CalendarFRBreach float64
}

// DBMedians mirror Workforce SQL on applied_slas (optional).
type DBMedians struct {
	WorkingFR          float64
	WorkingRes         float64
	CalendarFR         float64
	CalendarRes        float64
	CalendarResWorking float64
}

// ReconcileCounts describe how many issues contributed to each median.
type ReconcileCounts struct {
	CacheInWeek          int
	CacheFRPylonSec      int
	CacheFRCalendar      int
	CacheResPylonSec     int
	CacheResCalendar     int
	DBInWeek             int
	DBFRWorking          int
	DBFRCalendar         int
	DBResWorking         int
	DBResCalendar        int
}

// LoadIssuesJSON reads issues.json from a Pylon import cache directory or file path.
func LoadIssuesJSON(path string) ([]Issue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Railway SSH may prefix stdout with a one-line status banner.
	if i := bytesIndexByte(data, '['); i > 0 {
		data = data[i:]
	}
	var issues []Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

func bytesIndexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}

// pgWeekLabel matches Workforce SQL:
// TO_CHAR(DATE_TRUNC('week', ts AT TIME ZONE tz), 'YYYY-MM-DD').
func pgWeekLabel(ts time.Time, tz string) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", err
	}
	local := ts.In(loc)
	dow := int(local.Weekday())
	// PostgreSQL DATE_TRUNC('week'): week starts Monday.
	daysFromMonday := (dow + 6) % 7
	monday := time.Date(local.Year(), local.Month(), local.Day()-daysFromMonday, 0, 0, 0, 0, loc)
	return monday.Format("2006-01-02"), nil
}

func medianFloat64(values []float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	sort.Float64s(values)
	n := len(values)
	if n%2 == 1 {
		return values[n/2]
	}
	return (values[n/2-1] + values[n/2]) / 2
}

func calendarSec(start, end time.Time) float64 {
	if end.Before(start) {
		return math.NaN()
	}
	return end.Sub(start).Seconds()
}

// Bucket issues by Postgres week label on created_at, first_response, or resolution event.
type SLABucket string

const (
	BucketByCreatedAt    SLABucket = "created_at"
	BucketByFirstReply   SLABucket = "first_response"
	BucketByResolution   SLABucket = "resolution"
)

// ReconcileFromPylonCache computes medians for issues whose created_at falls in weekStart (YYYY-MM-DD label).
func ReconcileFromPylonCache(issues []Issue, weekLabel, tz string) (CacheMedians, ReconcileCounts, error) {
	return reconcileFromPylonCacheBucket(issues, weekLabel, tz, BucketByCreatedAt)
}

func reconcileFromPylonCacheBucket(issues []Issue, weekLabel, tz string, bucket SLABucket) (CacheMedians, ReconcileCounts, error) {
	var (
		frPylon, frCal, frBreach []float64
		resPylon, resCal         []float64
		inWeek                   int
	)
	for _, is := range issues {
		var anchor time.Time
		switch bucket {
		case BucketByCreatedAt:
			anchor = is.CreatedAt
		case BucketByFirstReply:
			if is.FirstResponseTime != nil {
				anchor = *is.FirstResponseTime
			} else if is.FirstResponseBreachTime != nil {
				anchor = *is.FirstResponseBreachTime
			} else {
				continue
			}
		case BucketByResolution:
			if is.ResolutionTime != nil {
				anchor = *is.ResolutionTime
			} else {
				continue
			}
		default:
			return CacheMedians{}, ReconcileCounts{}, fmt.Errorf("unknown bucket %q", bucket)
		}
		wk, err := pgWeekLabel(anchor, tz)
		if err != nil {
			return CacheMedians{}, ReconcileCounts{}, err
		}
		if wk != weekLabel {
			continue
		}
		inWeek++

		if is.FirstResponseSeconds != nil {
			frPylon = append(frPylon, float64(*is.FirstResponseSeconds))
		}
		frEnd := is.FirstResponseTime
		if frEnd == nil {
			frEnd = is.FirstResponseBreachTime
		}
		if frEnd != nil {
			frCal = append(frCal, calendarSec(is.CreatedAt, *frEnd))
		}
		if is.FirstResponseTime != nil || is.FirstResponseBreachTime != nil {
			end := is.FirstResponseTime
			if end == nil {
				end = is.FirstResponseBreachTime
			}
			frBreach = append(frBreach, calendarSec(is.CreatedAt, *end))
		}

		if is.ResolutionSeconds != nil {
			resPylon = append(resPylon, float64(*is.ResolutionSeconds))
		}
		resEnd := is.ResolutionTime
		if resEnd != nil {
			resCal = append(resCal, calendarSec(is.CreatedAt, *resEnd))
		}
	}

	return CacheMedians{
			PylonFRSeconds:   medianFloat64(frPylon),
			PylonResSeconds:  medianFloat64(resPylon),
			CalendarFR:       medianFloat64(frCal),
			CalendarRes:      medianFloat64(resCal),
			CalendarFRBreach: medianFloat64(frBreach),
		}, ReconcileCounts{
			CacheInWeek:      inWeek,
			CacheFRPylonSec:  len(frPylon),
			CacheFRCalendar:  len(frCal),
			CacheResPylonSec: len(resPylon),
			CacheResCalendar: len(resCal),
		}, nil
}

// FormatDuration formats seconds like the Workforce UI (e.g. 9h 12m).
func FormatDuration(sec float64) string {
	if math.IsNaN(sec) || sec < 0 {
		return "—"
	}
	sec = math.Round(sec)
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// ReconcileWeekSweep tries every bucket and prints medians for one week (debug parity search).
func ReconcileWeekSweep(issues []Issue, weekLabel, tz string) {
	targetFR := 19*3600 + 15*60
	targetRes := 22*3600 + 53*60
	for _, bucket := range []SLABucket{BucketByCreatedAt, BucketByFirstReply, BucketByResolution} {
		cache, counts, err := reconcileFromPylonCacheBucket(issues, weekLabel, tz, bucket)
		if err != nil {
			fmt.Printf("%s: error: %v\n", bucket, err)
			continue
		}
		fmt.Printf("\n--- bucket %s (n=%d) ---\n", bucket, counts.CacheInWeek)
		fmt.Printf("  FR pylon seconds: %s | calendar: %s\n", FormatDuration(cache.PylonFRSeconds), FormatDuration(cache.CalendarFR))
		fmt.Printf("  RES pylon seconds: %s | calendar: %s\n", FormatDuration(cache.PylonResSeconds), FormatDuration(cache.CalendarRes))
		matchFR := absDiff(cache.PylonFRSeconds, float64(targetFR)) < 1800 || absDiff(cache.CalendarFR, float64(targetFR)) < 1800
		matchRes := absDiff(cache.PylonResSeconds, float64(targetRes)) < 600 || absDiff(cache.CalendarRes, float64(targetRes)) < 600
		if matchFR {
			fmt.Println("  >>> possible FR match to Pylon UI (within tolerance)")
		}
		if matchRes {
			fmt.Println("  >>> possible RES match to Pylon UI (within tolerance)")
		}
	}
}

func absDiff(a, b float64) float64 {
	if math.IsNaN(a) {
		return math.MaxFloat64
	}
	d := a - b
	if d < 0 {
		return -d
	}
	return d
}

// PrintReconcileReport writes a human-readable comparison to stdout.
func PrintReconcileReport(r SLAReconcileWeek) {
	fmt.Printf("SLA reconcile — week label %s (%s)\n\n", r.Week, r.Timezone)

	fmt.Println("Pylon UI (reference):")
	if r.PylonTargets.HaveFR {
		fmt.Printf("  first response median: %s (%.0fs)\n", FormatDuration(r.PylonTargets.MedianFirstResponseSec), r.PylonTargets.MedianFirstResponseSec)
	} else {
		fmt.Println("  first response median: (not set)")
	}
	if r.PylonTargets.HaveRes {
		fmt.Printf("  resolution median:     %s (%.0fs)\n", FormatDuration(r.PylonTargets.MedianResolutionSec), r.PylonTargets.MedianResolutionSec)
	} else {
		fmt.Println("  resolution median: (not set)")
	}

	fmt.Printf("\nPylon cache issues.json (n=%d in week):\n", r.Counts.CacheInWeek)
	fmt.Printf("  median first_response_seconds:     %s (n=%d)\n", FormatDuration(r.PylonCache.PylonFRSeconds), r.Counts.CacheFRPylonSec)
	fmt.Printf("  median calendar FR (time−created):  %s (n=%d)\n", FormatDuration(r.PylonCache.CalendarFR), r.Counts.CacheFRCalendar)
	fmt.Printf("  median resolution_seconds:         %s (n=%d)\n", FormatDuration(r.PylonCache.PylonResSeconds), r.Counts.CacheResPylonSec)
	fmt.Printf("  median calendar res (time−created): %s (n=%d)\n", FormatDuration(r.PylonCache.CalendarRes), r.Counts.CacheResCalendar)

	if r.LibredeskDB != nil {
		fmt.Printf("\nLibredesk DB (Workforce universe, n=%d in week):\n", r.Counts.DBInWeek)
		fmt.Printf("  median first_response_working_sec:   %s (n=%d)\n", FormatDuration(r.LibredeskDB.WorkingFR), r.Counts.DBFRWorking)
		fmt.Printf("  median calendar FR (met−created):  %s (n=%d)\n", FormatDuration(r.LibredeskDB.CalendarFR), r.Counts.DBFRCalendar)
		fmt.Printf("  median calendar res (met−created): %s (n=%d)\n", FormatDuration(r.LibredeskDB.CalendarRes), r.Counts.DBResCalendar)
		fmt.Printf("  median resolution (COALESCE working): %s (n=%d)\n", FormatDuration(r.LibredeskDB.CalendarResWorking), r.Counts.DBResWorking)
	}
}