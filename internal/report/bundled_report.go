package report

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/abhinavxd/libredesk/internal/envelope"
)

// loadBundledReport runs scalar and chart queries in one read-only transaction and returns merged JSON.
func (m *Manager) loadBundledReport(
	scalarSQL, chartSQL string,
	scalarArgs, chartArgs []any,
	scalarLogLabel, chartLogLabel string,
) (json.RawMessage, error) {
	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		m.lo.Error("error starting db txn", "error", err)
		return nil, m.reportErr()
	}
	defer tx.Rollback()

	var (
		scalars   json.RawMessage
		charts    json.RawMessage
		scalarObj map[string]any
		chartObj  map[string]any
	)

	if err := tx.Get(&scalars, scalarSQL, scalarArgs...); err != nil {
		m.lo.Error("error fetching "+scalarLogLabel, "error", err)
		return nil, m.reportErr()
	}
	if err := json.Unmarshal(scalars, &scalarObj); err != nil {
		m.lo.Error("error unmarshaling "+scalarLogLabel, "error", err)
		return nil, m.reportErr()
	}

	if err := tx.Get(&charts, chartSQL, chartArgs...); err != nil {
		m.lo.Error("error fetching "+chartLogLabel, "error", err)
		return nil, m.reportErr()
	}
	if err := json.Unmarshal(charts, &chartObj); err != nil {
		m.lo.Error("error unmarshaling "+chartLogLabel, "error", err)
		return nil, m.reportErr()
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing db txn", "error", err)
		return nil, m.reportErr()
	}

	out, err := json.Marshal(map[string]any{
		"scalars": scalarObj,
		"charts":  chartObj,
	})
	if err != nil {
		m.lo.Error("error marshaling bundled report", "error", err)
		return nil, m.reportErr()
	}
	return out, nil
}

func (m *Manager) reportErr() error {
	return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
}