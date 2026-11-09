package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// GetIssuesReport returns Pylon-style issue analytics for the selected day range.
func (m *Manager) GetIssuesReport(days int, timezone, accountSearch string) (json.RawMessage, error) {
	days = clampDays(days)
	tz := stringutil.NormalizeTimezone(timezone)
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

	scalarQuery := fmt.Sprintf(
		m.q.GetIssuesScalars,
		days, days,
		days, days,
		days, days,
		days, days,
		days, days,
	)
	if err := tx.Get(&scalars, scalarQuery, tz); err != nil {
		m.lo.Error("error fetching issues scalars", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := json.Unmarshal(scalars, &scalarObj); err != nil {
		m.lo.Error("error unmarshaling issues scalars", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	chartQuery := fmt.Sprintf(m.q.GetIssuesCharts, days, days)
	if err := tx.Get(&charts, chartQuery, tz); err != nil {
		m.lo.Error("error fetching issues charts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := json.Unmarshal(charts, &chartObj); err != nil {
		m.lo.Error("error unmarshaling issues charts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	topQuery := fmt.Sprintf(m.q.GetIssuesTopAccounts, days, days)
	if err := tx.Get(&topAccounts, topQuery, tz, accountSearch); err != nil {
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
