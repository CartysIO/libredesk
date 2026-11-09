package report

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestGetIssuesReportScalars(t *testing.T) {
	db := testutil.NewDB(t, "report_issues")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}

	var orgID int
	if err := db.Get(&orgID, `INSERT INTO organizations (name, external_id) VALUES ('Acme', 'team-1') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	var contactID, inboxID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'Test', 'User') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, organization_id, created_at, resolved_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), $3, $4, NULL)`,
		contactID, inboxID, orgID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, organization_id, created_at, resolved_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'resolved' LIMIT 1), $3, $4, $4)`,
		contactID, inboxID, orgID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	raw, err := m.GetIssuesReport(7, "UTC", "")
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Scalars struct {
			AllTimeAccounts int `json:"all_time_accounts"`
			AllTimeIssues   int `json:"all_time_issues"`
			IssuesCreated   int `json:"issues_created"`
			IssuesSolved    int `json:"issues_solved"`
			IssuesUnsolved  int `json:"issues_unsolved"`
		} `json:"scalars"`
		TopAccounts []struct {
			AccountName string `json:"account_name"`
			Total       int    `json:"total"`
		} `json:"top_accounts"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Scalars.AllTimeAccounts != 1 || payload.Scalars.AllTimeIssues != 2 {
		t.Fatalf("all-time counts = %+v", payload.Scalars)
	}
	if payload.Scalars.IssuesCreated != 2 || payload.Scalars.IssuesUnsolved != 1 || payload.Scalars.IssuesSolved != 1 {
		t.Fatalf("range counts = %+v", payload.Scalars)
	}
	if len(payload.TopAccounts) != 1 || payload.TopAccounts[0].Total != 2 {
		t.Fatalf("top accounts = %+v", payload.TopAccounts)
	}
}

func TestIssuesTopAccountsOpenClosedByResolvedAt(t *testing.T) {
	db := testutil.NewDB(t, "report_issues_open_closed")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	var orgID, contactID, inboxID int
	if err := db.Get(&orgID, `INSERT INTO organizations (name, external_id) VALUES ('Beta', 'team-2') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'T', 'U') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('I', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	openStatusID, resolvedStatusID := 0, 0
	if err := db.Get(&openStatusID, `SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&resolvedStatusID, `SELECT id FROM conversation_statuses WHERE category = 'resolved' LIMIT 1`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, organization_id, created_at, resolved_at)
		VALUES ($1, $2, $3, $4, $5, NULL)`, contactID, inboxID, openStatusID, orgID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, organization_id, created_at, resolved_at)
		VALUES ($1, $2, $3, $4, $5, $5)`, contactID, inboxID, resolvedStatusID, orgID, now); err != nil {
		t.Fatal(err)
	}

	raw, err := m.GetIssuesReport(7, "UTC", "")
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		TopAccounts []struct {
			Open   int `json:"open"`
			Closed int `json:"closed"`
		} `json:"top_accounts"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.TopAccounts) != 1 {
		t.Fatalf("top accounts len = %d", len(payload.TopAccounts))
	}
	row := payload.TopAccounts[0]
	if row.Open != 1 || row.Closed != 1 {
		t.Fatalf("open/closed = %+v, want open=1 closed=1", row)
	}
}
