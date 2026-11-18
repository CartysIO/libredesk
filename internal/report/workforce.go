package report

import (
	"encoding/json"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// GetWorkforceReport returns SLA and agent workload analytics for the selected day range.
func (m *Manager) GetWorkforceReport(in IssuesRangeInput) (json.RawMessage, error) {
	bounds, err := ResolveIssuesRangeBounds(in)
	if err != nil {
		return nil, envelope.NewError(envelope.InputError, err.Error(), nil)
	}
	tz := stringutil.NormalizeTimezone(in.AppTimezone)
	return m.loadBundledReport(
		m.q.GetWorkforceScalars,
		m.q.GetWorkforceCharts,
		[]any{bounds.StartsAt, bounds.EndsAt},
		[]any{tz, bounds.StartsAt, bounds.EndsAt},
		"workforce scalars",
		"workforce charts",
	)
}