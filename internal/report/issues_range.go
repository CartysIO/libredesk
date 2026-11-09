package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// IssuesRangeBounds is the inclusive-start, exclusive-end window for issues report queries.
type IssuesRangeBounds struct {
	StartsAt time.Time
	EndsAt   time.Time
}

// IssuesRangeInput selects either a rolling day window or an explicit calendar range (YYYY-MM-DD).
type IssuesRangeInput struct {
	Days        int
	StartDate   string
	EndDate     string
	AppTimezone string
}

const maxCustomRangeDays = 365

// ResolveIssuesRangeBounds maps preset days or custom dates to query timestamps in the app timezone.
func ResolveIssuesRangeBounds(in IssuesRangeInput) (IssuesRangeBounds, error) {
	tz := stringutil.NormalizeTimezone(in.AppTimezone)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return IssuesRangeBounds{}, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	startS := trimDate(in.StartDate)
	endS := trimDate(in.EndDate)
	if startS != "" || endS != "" {
		if startS == "" || endS == "" {
			return IssuesRangeBounds{}, fmt.Errorf("start_date and end_date are required together")
		}
		startDay, err := parseISODate(startS)
		if err != nil {
			return IssuesRangeBounds{}, fmt.Errorf("start_date: %w", err)
		}
		endDay, err := parseISODate(endS)
		if err != nil {
			return IssuesRangeBounds{}, fmt.Errorf("end_date: %w", err)
		}
		if endDay.Before(startDay) {
			return IssuesRangeBounds{}, fmt.Errorf("end_date must be on or after start_date")
		}
		span := endDay.Sub(startDay).Hours()/24 + 1
		if span > maxCustomRangeDays {
			return IssuesRangeBounds{}, fmt.Errorf("date range cannot exceed %d days", maxCustomRangeDays)
		}
		startsAt := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, loc)
		endExclusive := time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
		return IssuesRangeBounds{StartsAt: startsAt.UTC(), EndsAt: endExclusive.UTC()}, nil
	}

	days := clampDays(in.Days)
	if days == 0 {
		now := time.Now().In(loc)
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		return IssuesRangeBounds{StartsAt: start.UTC(), EndsAt: now.UTC()}, nil
	}
	endsAt := time.Now().UTC()
	startsAt := endsAt.Add(-time.Duration(days) * 24 * time.Hour)
	return IssuesRangeBounds{StartsAt: startsAt, EndsAt: endsAt}, nil
}

func trimDate(s string) string {
	return strings.TrimSpace(s)
}

func parseISODate(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
