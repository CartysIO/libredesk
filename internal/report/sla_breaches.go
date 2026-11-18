package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
)

const (
	defaultSlaBreachesDays     = 90
	slaBreachesDefaultPageSize = 50
)

// SlaBreachesDefaultPageSize is the default page size for the SLA breaches report API.
func SlaBreachesDefaultPageSize() int {
	return slaBreachesDefaultPageSize
}

// SlaBreachesSort is server-side ordering for the SLA breaches table.
type SlaBreachesSort struct {
	By  string
	Dir string
}

func (s SlaBreachesSort) orderExpr() (string, error) {
	by := strings.ToLower(strings.TrimSpace(s.By))
	dir := strings.ToLower(strings.TrimSpace(s.Dir))
	if by == "" {
		by = "created_at"
	}
	if dir == "" {
		dir = "desc"
	}

	cols := map[string]string{
		"reference_number":      "reference_number",
		"created_at":            "conversation_created_at",
		"subject":               "subject",
		"first_response_sec":    "first_response_sec",
		"resolution_sec":        "resolution_sec",
		"max_next_response_sec": "max_next_response_sec",
		"organization_name":     "organization_name",
	}
	col, ok := cols[by]
	if !ok {
		return "", fmt.Errorf("invalid sort_by %q", by)
	}
	switch dir {
	case "asc", "desc":
	default:
		return "", fmt.Errorf("invalid sort_dir %q", dir)
	}
	dirSQL := strings.ToUpper(dir)
	nulls := " NULLS LAST"
	if col == "organization_name" || col == "subject" || col == "reference_number" {
		nulls = ""
	}
	return fmt.Sprintf("%s %s%s, conversation_created_at DESC, reference_number", col, dirSQL, nulls), nil
}

// SlaBreachesPage is a paginated SLA breaches result.
type SlaBreachesPage struct {
	Rows  json.RawMessage `json:"rows"`
	Total int             `json:"total"`
}

// GetSlaBreaches returns conversations whose latest applied SLA breached first response or resolution.
func (m *Manager) GetSlaBreaches(in IssuesRangeInput, search string, sort SlaBreachesSort, page, pageSize int) (SlaBreachesPage, error) {
	if strings.TrimSpace(in.StartDate) == "" && strings.TrimSpace(in.EndDate) == "" && in.Days == 0 {
		in.Days = defaultSlaBreachesDays
	}
	bounds, err := ResolveIssuesRangeBounds(in)
	if err != nil {
		return SlaBreachesPage{}, envelope.NewError(envelope.InputError, err.Error(), nil)
	}
	orderExpr, err := sort.orderExpr()
	if err != nil {
		return SlaBreachesPage{}, envelope.NewError(envelope.InputError, err.Error(), nil)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = slaBreachesDefaultPageSize
	}
	offset := (page - 1) * pageSize

	search = normalizeConversationSearch(search)
	query := fmt.Sprintf(m.q.GetSlaBreaches, orderExpr)

	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		m.lo.Error("error starting db txn", "error", err)
		return SlaBreachesPage{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	var row struct {
		Total int             `db:"total"`
		Rows  json.RawMessage `db:"rows"`
	}
	if err := tx.Get(&row, query, bounds.StartsAt, bounds.EndsAt, search, pageSize, offset); err != nil {
		m.lo.Error("error fetching sla breaches", "error", err)
		return SlaBreachesPage{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if row.Rows == nil {
		row.Rows = json.RawMessage("[]")
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing db txn", "error", err)
		return SlaBreachesPage{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	return SlaBreachesPage{Rows: row.Rows, Total: row.Total}, nil
}

func normalizeConversationSearch(search string) string {
	search = strings.TrimSpace(search)
	if strings.HasPrefix(search, "#") {
		search = strings.TrimSpace(search[1:])
	}
	return search
}