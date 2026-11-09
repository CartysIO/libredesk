package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// GetIssuesReport returns Pylon-style issue analytics for the selected day range.
func (m *Manager) GetIssuesReport(in IssuesRangeInput, accountSearch string) (json.RawMessage, error) {
	bounds, err := ResolveIssuesRangeBounds(in)
	if err != nil {
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	tz := stringutil.NormalizeTimezone(in.AppTimezone)
	accountSearch = strings.TrimSpace(accountSearch)

	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		m.lo.Error("error starting db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	var (
		scalars      json.RawMessage
		charts       json.RawMessage
		topAccounts  json.RawMessage
		scalarObj    map[string]any
		chartObj     map[string]any
		accountsList []any
	)

	if err := tx.Get(&scalars, m.q.GetIssuesScalars, bounds.StartsAt, bounds.EndsAt); err != nil {
		m.lo.Error("error fetching issues scalars", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := json.Unmarshal(scalars, &scalarObj); err != nil {
		m.lo.Error("error unmarshaling issues scalars", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Get(&charts, m.q.GetIssuesCharts, tz, bounds.StartsAt, bounds.EndsAt); err != nil {
		m.lo.Error("error fetching issues charts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := json.Unmarshal(charts, &chartObj); err != nil {
		m.lo.Error("error unmarshaling issues charts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Get(&topAccounts, m.q.GetIssuesTopAccounts, bounds.StartsAt, bounds.EndsAt, accountSearch); err != nil {
		m.lo.Error("error fetching issues top accounts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := json.Unmarshal(topAccounts, &accountsList); err != nil {
		m.lo.Error("error unmarshaling issues top accounts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	out, err := json.Marshal(map[string]any{
		"scalars":      scalarObj,
		"charts":       chartObj,
		"top_accounts": accountsList,
	})
	if err != nil {
		m.lo.Error("error marshaling issues report", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return out, nil
}
