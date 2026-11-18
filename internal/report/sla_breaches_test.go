package report

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestSlaBreachesSortOrderExpr(t *testing.T) {
	expr, err := SlaBreachesSort{}.orderExpr()
	if err != nil || expr != "conversation_created_at DESC NULLS LAST, conversation_created_at DESC, reference_number" {
		t.Fatalf("default = %q, %v", expr, err)
	}
}

func TestGetSlaBreachesIncludesBreachedConversation(t *testing.T) {
	db := testutil.NewDB(t, "report_sla_breaches")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}

	var contactID, inboxID, policyID, orgID, convID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'Test', 'User') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&policyID, `INSERT INTO sla_policies (name, description, first_response_time, resolution_time, next_response_time)
		VALUES ('Default', '', 3600, 86400, 3600) RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&orgID, `INSERT INTO organizations (name, external_id) VALUES ('Acme Corp', 'team-acme') RETURNING id`); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	created := now.Add(-2 * time.Hour)
	breachAt := created.Add(90 * time.Minute)
	if err := db.Get(&convID, `INSERT INTO conversations (contact_id, inbox_id, status_id, organization_id, subject, created_at, resolved_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), $3, 'Billing question', $4, NULL) RETURNING id`,
		contactID, inboxID, orgID, created); err != nil {
		t.Fatal(err)
	}
	var appliedID int
	if err := db.Get(&appliedID, `INSERT INTO applied_slas (
			conversation_id, sla_policy_id, status, created_at,
			first_response_deadline_at, first_response_breached_at,
			resolution_deadline_at, resolution_met_at
		) VALUES ($1, $2, 'pending', $3, $4, $4, $5, $5) RETURNING id`,
		convID, policyID, created, breachAt, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sla_events (applied_sla_id, sla_policy_id, type, status, deadline_at, met_at, created_at)
		VALUES ($1, $2, 'next_response', 'met', $3, $3, $3)`,
		appliedID, policyID, created.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}

	// Conversation without breach should not appear.
	if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, created_at, resolved_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), $3, NULL)`,
		contactID, inboxID, created); err != nil {
		t.Fatal(err)
	}

	page, err := m.GetSlaBreaches(IssuesRangeInput{Days: 7, AppTimezone: "UTC"}, "", SlaBreachesSort{}, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 {
		t.Fatalf("total = %d, want 1", page.Total)
	}
	var rows []struct {
		ReferenceNumber  string  `json:"reference_number"`
		OrganizationName string  `json:"organization_name"`
		FirstResponseSec float64 `json:"first_response_sec"`
	}
	if err := json.Unmarshal(page.Rows, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].OrganizationName != "Acme Corp" {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].FirstResponseSec < 5300 || rows[0].FirstResponseSec > 5500 {
		t.Fatalf("first_response_sec = %v", rows[0].FirstResponseSec)
	}
}

func TestGetSlaBreachesSearchBySubject(t *testing.T) {
	db := testutil.NewDB(t, "report_sla_breaches_search")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	var contactID, inboxID, policyID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'T', 'U') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&policyID, `INSERT INTO sla_policies (name, description, first_response_time, resolution_time, next_response_time)
		VALUES ('Default', '', 3600, 86400, 3600) RETURNING id`); err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC().Add(-time.Hour)
	var convID int
	if err := db.Get(&convID, `INSERT INTO conversations (contact_id, inbox_id, status_id, subject, created_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), 'UniqueWidgetFailure', $3) RETURNING id`,
		contactID, inboxID, created); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO applied_slas (conversation_id, sla_policy_id, status, created_at, first_response_deadline_at, resolution_breached_at)
		VALUES ($1, $2, 'pending', $3, $3, $3)`,
		convID, policyID, created); err != nil {
		t.Fatal(err)
	}

	page, err := m.GetSlaBreaches(IssuesRangeInput{Days: 7, AppTimezone: "UTC"}, "widget", SlaBreachesSort{}, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 {
		t.Fatalf("total = %d", page.Total)
	}
}

func TestGetSlaBreachesInvalidSortInputError(t *testing.T) {
	db := testutil.NewDB(t, "report_sla_breaches_bad_sort")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.GetSlaBreaches(IssuesRangeInput{Days: 7, AppTimezone: "UTC"}, "", SlaBreachesSort{By: "nope", Dir: "desc"}, 1, 50)
	var envErr envelope.Error
	if !errors.As(err, &envErr) || envErr.ErrorType != envelope.InputError {
		t.Fatalf("want InputError, got %v", err)
	}
}

func TestNormalizeConversationSearch(t *testing.T) {
	if got := normalizeConversationSearch("  #4068 "); got != "4068" {
		t.Fatalf("got %q", got)
	}
}

func TestGetSlaBreachesExcludesNextResponseOnly(t *testing.T) {
	db := testutil.NewDB(t, "report_sla_breaches_nr_only")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	var contactID, inboxID, policyID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'T', 'U') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&policyID, `INSERT INTO sla_policies (name, description, first_response_time, resolution_time, next_response_time)
		VALUES ('Default', '', 3600, 86400, 3600) RETURNING id`); err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC().Add(-time.Hour)
	var convID int
	if err := db.Get(&convID, `INSERT INTO conversations (contact_id, inbox_id, status_id, subject, created_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), 'NrOnly', $3) RETURNING id`,
		contactID, inboxID, created); err != nil {
		t.Fatal(err)
	}
	var appliedID int
	if err := db.Get(&appliedID, `INSERT INTO applied_slas (
			conversation_id, sla_policy_id, status, created_at,
			first_response_met_at, resolution_met_at
		) VALUES ($1, $2, 'met', $3, $3, $3) RETURNING id`,
		convID, policyID, created); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sla_events (applied_sla_id, sla_policy_id, type, status, deadline_at, breached_at, created_at)
		VALUES ($1, $2, 'next_response', 'breached', $3, $3, $3)`,
		appliedID, policyID, created.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}

	page, err := m.GetSlaBreaches(IssuesRangeInput{Days: 7, AppTimezone: "UTC"}, "", SlaBreachesSort{}, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 {
		t.Fatalf("total = %d, want 0 for next-response-only breach", page.Total)
	}
}

func TestGetSlaBreachesSearchByReferenceAndOrg(t *testing.T) {
	db := testutil.NewDB(t, "report_sla_breaches_ref_org")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	var contactID, inboxID, policyID, orgID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'T', 'U') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&policyID, `INSERT INTO sla_policies (name, description, first_response_time, resolution_time, next_response_time)
		VALUES ('Default', '', 3600, 86400, 3600) RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&orgID, `INSERT INTO organizations (name, external_id) VALUES ('ZetaSearchOrg', 'team-zeta') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC().Add(-time.Hour)
	var ref string
	if err := db.Get(&ref, `INSERT INTO conversations (contact_id, inbox_id, status_id, organization_id, subject, created_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), $3, 'RefOrg', $4) RETURNING reference_number`,
		contactID, inboxID, orgID, created); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO applied_slas (conversation_id, sla_policy_id, status, created_at, first_response_deadline_at, first_response_breached_at)
		VALUES ((SELECT id FROM conversations WHERE reference_number = $1), $2, 'pending', $3, $3, $3)`,
		ref, policyID, created); err != nil {
		t.Fatal(err)
	}

	byRef, err := m.GetSlaBreaches(IssuesRangeInput{Days: 7, AppTimezone: "UTC"}, ref, SlaBreachesSort{}, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if byRef.Total != 1 {
		t.Fatalf("ref search total = %d", byRef.Total)
	}
	byHash, err := m.GetSlaBreaches(IssuesRangeInput{Days: 7, AppTimezone: "UTC"}, "#"+ref, SlaBreachesSort{}, 1, 50)
	if err != nil || byHash.Total != 1 {
		t.Fatalf("# ref search total = %d, err %v", byHash.Total, err)
	}
	byOrg, err := m.GetSlaBreaches(IssuesRangeInput{Days: 7, AppTimezone: "UTC"}, "ZetaSearch", SlaBreachesSort{}, 1, 50)
	if err != nil || byOrg.Total != 1 {
		t.Fatalf("org search total = %d, err %v", byOrg.Total, err)
	}
}

func TestGetSlaBreachesCreatedAtDisplayTimezone(t *testing.T) {
	db := testutil.NewDB(t, "report_sla_breaches_tz")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	var contactID, inboxID, policyID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'T', 'U') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&policyID, `INSERT INTO sla_policies (name, description, first_response_time, resolution_time, next_response_time)
		VALUES ('Default', '', 3600, 86400, 3600) RETURNING id`); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 6, 15, 18, 30, 0, 0, time.UTC)
	var convID int
	if err := db.Get(&convID, `INSERT INTO conversations (contact_id, inbox_id, status_id, created_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), $3) RETURNING id`,
		contactID, inboxID, created); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO applied_slas (conversation_id, sla_policy_id, status, created_at, first_response_deadline_at, resolution_breached_at)
		VALUES ($1, $2, 'pending', $3, $3, $3)`,
		convID, policyID, created); err != nil {
		t.Fatal(err)
	}

	page, err := m.GetSlaBreaches(IssuesRangeInput{Days: 365, AppTimezone: "America/Chicago"}, "", SlaBreachesSort{}, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		CreatedAtDisplay string `json:"created_at_display"`
	}
	if err := json.Unmarshal(page.Rows, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || !strings.Contains(rows[0].CreatedAtDisplay, "Jun") {
		t.Fatalf("created_at_display = %q", rows[0].CreatedAtDisplay)
	}
}
