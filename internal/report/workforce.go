package report

import (
	"context"
	"database/sql"
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

	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		m.lo.Error("error starting db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	var (
		scalars   json.RawMessage
		charts    json.RawMessage
		scalarObj map[string]any
		chartObj  map[string]any
	)

	if err := tx.Get(&scalars, m.q.GetWorkforceScalars, bounds.StartsAt, bounds.EndsAt); err != nil {
		m.lo.Error("error fetching workforce scalars", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := json.Unmarshal(scalars, &scalarObj); err != nil {
		m.lo.Error("error unmarshaling workforce scalars", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Get(&charts, m.q.GetWorkforceCharts, tz, bounds.StartsAt, bounds.EndsAt); err != nil {
		m.lo.Error("error fetching workforce charts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := json.Unmarshal(charts, &chartObj); err != nil {
		m.lo.Error("error unmarshaling workforce charts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	out, err := json.Marshal(map[string]any{
		"scalars": scalarObj,
		"charts":  chartObj,
	})
	if err != nil {
		m.lo.Error("error marshaling workforce report", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return out, nil
}