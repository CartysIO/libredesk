package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestIssuesTopAccountsSortOrderExpr(t *testing.T) {
	expr, err := IssuesTopAccountsSort{}.orderExpr()
	if err != nil || expr != "total DESC" {
		t.Fatalf("default = %q, %v", expr, err)
	}
	expr, err = IssuesTopAccountsSort{By: "open", Dir: "asc"}.orderExpr()
	if err != nil || expr != "open ASC" {
		t.Fatalf("open asc = %q, %v", expr, err)
	}
	if _, err := (IssuesTopAccountsSort{By: "nope", Dir: "desc"}).orderExpr(); err == nil {
		t.Fatal("expected error for invalid sort column")
	}
}

func TestGetIssuesTopAccountsSortByOpen(t *testing.T) {
	db := testutil.NewDB(t, "report_issues_top_sort")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	var orgA, orgB, contactID, inboxID int
	if err := db.Get(&orgA, `INSERT INTO organizations (name, external_id) VALUES ('Alpha', 'team-a') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&orgB, `INSERT INTO organizations (name, external_id) VALUES ('Beta', 'team-b') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'Test', 'User') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
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
	insert := func(orgID int) {
		if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, organization_id, created_at, resolved_at)
			VALUES ($1, $2, $3, $4, $5, NULL)`, contactID, inboxID, openStatusID, orgID, now); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, organization_id, created_at, resolved_at)
			VALUES ($1, $2, $3, $4, $5, $5)`, contactID, inboxID, resolvedStatusID, orgID, now); err != nil {
			t.Fatal(err)
		}
	}
	insert(orgA)
	insert(orgB)
	insert(orgB)

	raw, err := m.GetIssuesTopAccounts(
		IssuesRangeInput{Days: 7, AppTimezone: "UTC"},
		"",
		IssuesTopAccountsSort{By: "open", Dir: "desc"},
	)
	if err != nil {
		t.Fatal(err)
	}
	var topAccounts []struct {
		AccountName string `json:"account_name"`
		Open        int    `json:"open"`
	}
	if err := json.Unmarshal(raw, &topAccounts); err != nil {
		t.Fatal(err)
	}
	if len(topAccounts) != 2 {
		t.Fatalf("len = %d", len(topAccounts))
	}
	if topAccounts[0].AccountName != "Beta" || topAccounts[0].Open != 2 {
		t.Fatalf("sort order = %+v", topAccounts)
	}
}

func TestGetIssuesTopAccountsInvalidSortInputError(t *testing.T) {
	db := testutil.NewDB(t, "report_issues_top_sort_bad")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.GetIssuesTopAccounts(
		IssuesRangeInput{Days: 7, AppTimezone: "UTC"},
		"",
		IssuesTopAccountsSort{By: "nope", Dir: "desc"},
	)
	var envErr envelope.Error
	if !errors.As(err, &envErr) || envErr.ErrorType != envelope.InputError {
		t.Fatalf("want InputError, got %v", err)
	}
}

func TestGetIssuesTopAccountsLimit(t *testing.T) {
	db := testutil.NewDB(t, "report_issues_top_limit")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	var contactID, inboxID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'Test', 'User') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	var openStatusID int
	if err := db.Get(&openStatusID, `SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i := 0; i < issuesTopAccountsLimit+2; i++ {
		var orgID int
		name := fmt.Sprintf("Org %02d", i)
		if err := db.Get(&orgID, `INSERT INTO organizations (name, external_id) VALUES ($1, $2) RETURNING id`, name, fmt.Sprintf("team-%d", i)); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, organization_id, created_at, resolved_at)
			VALUES ($1, $2, $3, $4, $5, NULL)`, contactID, inboxID, openStatusID, orgID, now); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := m.GetIssuesTopAccounts(IssuesRangeInput{Days: 7, AppTimezone: "UTC"}, "", IssuesTopAccountsSort{})
	if err != nil {
		t.Fatal(err)
	}
	var topAccounts []struct {
		AccountName string `json:"account_name"`
	}
	if err := json.Unmarshal(raw, &topAccounts); err != nil {
		t.Fatal(err)
	}
	if len(topAccounts) != issuesTopAccountsLimit {
		t.Fatalf("len = %d, want %d", len(topAccounts), issuesTopAccountsLimit)
	}
}