package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
)

const defaultIssuesTopAccountsDays = 120

// IssuesTopAccountsSort is server-side ordering for the issues top-accounts table.
type IssuesTopAccountsSort struct {
	By  string
	Dir string
}

func (s IssuesTopAccountsSort) orderExpr() (string, error) {
	by := strings.ToLower(strings.TrimSpace(s.By))
	dir := strings.ToLower(strings.TrimSpace(s.Dir))
	if by == "" {
		by = "total"
	}
	if dir == "" {
		dir = "desc"
	}

	cols := map[string]string{
		"account_name":       "account_name",
		"open":               "open",
		"closed":             "closed",
		"total":              "total",
		"avg_resolution_sec": "avg_resolution_sec",
		"avg_touches":        "avg_touches",
	}
	col, ok := cols[by]
	if !ok {
		return "", fmt.Errorf("invalid accounts_sort_by %q", by)
	}
	switch dir {
	case "asc", "desc":
	default:
		return "", fmt.Errorf("invalid accounts_sort_dir %q", dir)
	}
	return fmt.Sprintf("%s %s", col, strings.ToUpper(dir)), nil
}

// GetIssuesTopAccounts returns ranked organization stats for the issues report table.
func (m *Manager) GetIssuesTopAccounts(in IssuesRangeInput, accountSearch string, sort IssuesTopAccountsSort) (json.RawMessage, error) {
	if strings.TrimSpace(in.StartDate) == "" && strings.TrimSpace(in.EndDate) == "" && in.Days == 0 {
		in.Days = defaultIssuesTopAccountsDays
	}
	bounds, err := ResolveIssuesRangeBounds(in)
	if err != nil {
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	orderExpr, err := sort.orderExpr()
	if err != nil {
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	topAccountsQuery := fmt.Sprintf(m.q.GetIssuesTopAccounts, orderExpr)
	accountSearch = strings.TrimSpace(accountSearch)

	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		m.lo.Error("error starting db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	var topAccounts json.RawMessage
	if err := tx.Get(&topAccounts, topAccountsQuery, bounds.StartsAt, bounds.EndsAt, accountSearch); err != nil {
		m.lo.Error("error fetching issues top accounts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var accountsList []any
	if err := json.Unmarshal(topAccounts, &accountsList); err != nil {
		m.lo.Error("error unmarshaling issues top accounts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	out, err := json.Marshal(accountsList)
	if err != nil {
		m.lo.Error("error marshaling issues top accounts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return out, nil
}