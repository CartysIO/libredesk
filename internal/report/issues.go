package report

import (
	"encoding/json"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// GetIssuesReport returns Pylon-style issue analytics (scalars and charts) for the selected day range.
func (m *Manager) GetIssuesReport(in IssuesRangeInput) (json.RawMessage, error) {
	bounds, err := ResolveIssuesRangeBounds(in)
	if err != nil {
		return nil, envelope.NewError(envelope.InputError, err.Error(), nil)
	}
	tz := stringutil.NormalizeTimezone(in.AppTimezone)
	return m.loadBundledReport(
		m.q.GetIssuesScalars,
		m.q.GetIssuesCharts,
		[]any{bounds.StartsAt, bounds.EndsAt},
		[]any{tz, bounds.StartsAt, bounds.EndsAt},
		"issues scalars",
		"issues charts",
	)
}