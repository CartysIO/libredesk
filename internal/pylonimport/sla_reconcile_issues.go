package pylonimport

import (
	"fmt"
	"sort"
	"time"
)

// WeekIssueRow is one Pylon issue in a Workforce week bucket with derived durations.
type WeekIssueRow struct {
	Number              int
	ID                  string
	State               string
	CreatedAt           time.Time
	FirstResponseAt     *time.Time
	CalendarFRSec       float64 // NaN if unknown
	PylonFRSeconds      *int
	ResolutionAt        *time.Time
	PylonResSeconds     *int
	CalendarResSec      float64
	HasFRSeconds        bool
	HasResSeconds       bool
}

// WeekIssuesFromCache lists issues in a Workforce week bucket (by Pylon created_at).
func WeekIssuesFromCache(issues []Issue, weekLabel, tz string) ([]WeekIssueRow, error) {
	return weekIssuesFromCache(issues, weekLabel, tz)
}

func weekIssuesFromCache(issues []Issue, weekLabel, tz string) ([]WeekIssueRow, error) {
	var rows []WeekIssueRow
	for _, is := range issues {
		wk, err := pgWeekLabel(is.CreatedAt, tz)
		if err != nil {
			return nil, err
		}
		if wk != weekLabel {
			continue
		}
		row := WeekIssueRow{
			Number:    is.Number,
			ID:        is.ID,
			State:     is.State,
			CreatedAt: is.CreatedAt,
			PylonFRSeconds: is.FirstResponseSeconds,
			PylonResSeconds: is.ResolutionSeconds,
		}
		if is.FirstResponseSeconds != nil {
			row.HasFRSeconds = true
		}
		if is.ResolutionSeconds != nil {
			row.HasResSeconds = true
		}
		if is.FirstResponseTime != nil {
			row.FirstResponseAt = is.FirstResponseTime
		}
		frEnd := is.FirstResponseTime
		if frEnd == nil {
			frEnd = is.FirstResponseBreachTime
		}
		if frEnd != nil {
			row.CalendarFRSec = calendarSec(is.CreatedAt, *frEnd)
		}
		if is.ResolutionTime != nil {
			row.ResolutionAt = is.ResolutionTime
			row.CalendarResSec = calendarSec(is.CreatedAt, *is.ResolutionTime)
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Number < rows[j].Number })
	return rows, nil
}

// PrintWeekIssueTable prints per-issue durations for parity debugging.
func PrintWeekIssueTable(rows []WeekIssueRow, weekLabel string) {
	fmt.Printf("\nPylon issues — week %s (by created_at), count=%d\n\n", weekLabel, len(rows))
	fmt.Printf("%-6s %-10s %-22s %-12s %-12s %-12s %-12s\n",
		"#", "state", "created_utc", "cal_FR", "api_FR_sec", "cal_RES", "api_RES_sec")
	for _, r := range rows {
		calFR := FormatDuration(r.CalendarFRSec)
		apiFR := "—"
		if r.PylonFRSeconds != nil {
			apiFR = FormatDuration(float64(*r.PylonFRSeconds))
		}
		calRes := "—"
		if !isNaN(r.CalendarResSec) && r.CalendarResSec > 0 {
			calRes = FormatDuration(r.CalendarResSec)
		}
		apiRes := "—"
		if r.PylonResSeconds != nil {
			apiRes = FormatDuration(float64(*r.PylonResSeconds))
		}
		fmt.Printf("%-6d %-10s %-22s %-12s %-12s %-12s %-12s\n",
			r.Number, r.State, r.CreatedAt.UTC().Format("2006-01-02 15:04"),
			calFR, apiFR, calRes, apiRes)
	}

	var calFR, apiFR, calRes, apiRes []float64
	for _, r := range rows {
		if !isNaN(r.CalendarFRSec) {
			calFR = append(calFR, r.CalendarFRSec)
		}
		if r.PylonFRSeconds != nil {
			apiFR = append(apiFR, float64(*r.PylonFRSeconds))
		}
		if !isNaN(r.CalendarResSec) && r.CalendarResSec > 0 {
			calRes = append(calRes, r.CalendarResSec)
		}
		if r.PylonResSeconds != nil {
			apiRes = append(apiRes, float64(*r.PylonResSeconds))
		}
	}
	fmt.Printf("\nMedians: cal_FR=%s (n=%d) api_FR=%s (n=%d) cal_RES=%s (n=%d) api_RES=%s (n=%d)\n",
		FormatDuration(medianFloat64(calFR)), len(calFR),
		FormatDuration(medianFloat64(apiFR)), len(apiFR),
		FormatDuration(medianFloat64(calRes)), len(calRes),
		FormatDuration(medianFloat64(apiRes)), len(apiRes))

	missing := 0
	for _, r := range rows {
		if !r.HasFRSeconds && !isNaN(r.CalendarFRSec) {
			missing++
		}
	}
	fmt.Printf("\nIssues with calendar FR but NO first_response_seconds in API: %d\n", missing)
}

func isNaN(f float64) bool {
	return f != f
}