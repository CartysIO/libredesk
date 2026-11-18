package report

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestGetWorkforceReportScalars(t *testing.T) {
	db := testutil.NewDB(t, "report_workforce")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}

	var contactID, inboxID, agentID, policyID, convID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'Test', 'User') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&agentID, `INSERT INTO users (type, first_name, last_name, email) VALUES ('agent', 'Agent', 'One', 'agent@example.com') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&policyID, `INSERT INTO sla_policies (name, description, first_response_time, resolution_time, next_response_time)
		VALUES ('Default', '', 3600, 86400, 3600) RETURNING id`); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	created := now.Add(-2 * time.Hour)
	metFr := created.Add(30 * time.Minute)
	if err := db.Get(&convID, `INSERT INTO conversations (contact_id, inbox_id, status_id, assigned_user_id, created_at, resolved_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), $3, $4, NULL) RETURNING id`,
		contactID, inboxID, agentID, created); err != nil {
		t.Fatal(err)
	}
	var appliedID int
	if err := db.Get(&appliedID, `INSERT INTO applied_slas (
			conversation_id, sla_policy_id, status, created_at,
			first_response_deadline_at, first_response_met_at
		) VALUES ($1, $2, 'pending', $3, $4, $5) RETURNING id`,
		convID, policyID, created, metFr, metFr); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sla_events (applied_sla_id, sla_policy_id, type, status, deadline_at, created_at)
		VALUES ($1, $2, 'next_response', 'met', $3, $3)`,
		appliedID, policyID, now); err != nil {
		t.Fatal(err)
	}

	raw, err := m.GetWorkforceReport(IssuesRangeInput{Days: 7, AppTimezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Scalars struct {
			MedianFirstResponseSec float64 `json:"median_first_response_sec"`
			ActiveBreached         int     `json:"active_breached_count"`
		} `json:"scalars"`
		Charts struct {
			AssigneeOverview []struct {
				AssigneeName string `json:"assignee_name"`
				Total        int    `json:"total"`
			} `json:"assignee_overview"`
		} `json:"charts"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Scalars.MedianFirstResponseSec < 1700 || payload.Scalars.MedianFirstResponseSec > 1900 {
		t.Fatalf("median_first_response_sec = %v, want ~1800", payload.Scalars.MedianFirstResponseSec)
	}
	if len(payload.Charts.AssigneeOverview) != 1 || payload.Charts.AssigneeOverview[0].Total != 1 {
		t.Fatalf("assignee_overview = %+v", payload.Charts.AssigneeOverview)
	}

	// Active breached count is global: SLA applied outside the report window still counts when still open.
	oldCreated := now.Add(-90 * 24 * time.Hour)
	var conv2ID int
	if err := db.Get(&conv2ID, `INSERT INTO conversations (contact_id, inbox_id, status_id, created_at, resolved_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), $3, NULL) RETURNING id`,
		contactID, inboxID, oldCreated); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO applied_slas (
			conversation_id, sla_policy_id, status, created_at,
			first_response_deadline_at, first_response_breached_at
		) VALUES ($1, $2, 'pending', $3, $4, $4)`,
		conv2ID, policyID, oldCreated, oldCreated.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	rawNarrow, err := m.GetWorkforceReport(IssuesRangeInput{Days: 7, AppTimezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	var narrow struct {
		Scalars struct {
			ActiveBreached int `json:"active_breached_count"`
		} `json:"scalars"`
	}
	if err := json.Unmarshal(rawNarrow, &narrow); err != nil {
		t.Fatal(err)
	}
	if narrow.Scalars.ActiveBreached < 1 {
		t.Fatalf("active_breached_count = %d, want >= 1", narrow.Scalars.ActiveBreached)
	}
}

func TestGetWorkforceReportFirstResponseUsesCalendarTime(t *testing.T) {
	db := testutil.NewDB(t, "report_workforce_fr_calendar")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}

	var contactID, inboxID, policyID, convID int
	db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'Test', 'User') RETURNING id`)
	db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`)
	db.Get(&policyID, `INSERT INTO sla_policies (name, description, first_response_time, resolution_time, next_response_time)
		VALUES ('Default', '', 3600, 86400, 3600) RETURNING id`)

	now := time.Now().UTC()
	created := now.Add(-2 * time.Hour)
	metFr := created.Add(30 * time.Minute)
	db.Get(&convID, `INSERT INTO conversations (contact_id, inbox_id, status_id, created_at, resolved_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), $3, NULL) RETURNING id`,
		contactID, inboxID, created)
	const pylonFR = 518400 // stored from Pylon import but not used for Workforce FR charts
	_, err = db.Exec(`INSERT INTO applied_slas (
			conversation_id, sla_policy_id, status, created_at,
			first_response_deadline_at, first_response_met_at, first_response_working_sec
		) VALUES ($1, $2, 'pending', $3, $4, $5, $6)`,
		convID, policyID, created, metFr, metFr, pylonFR)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := m.GetWorkforceReport(IssuesRangeInput{Days: 7, AppTimezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Scalars struct {
			MedianFirstResponseSec float64 `json:"median_first_response_sec"`
		} `json:"scalars"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	wantFR := metFr.Sub(created).Seconds()
	if payload.Scalars.MedianFirstResponseSec != wantFR {
		t.Fatalf("median_first_response_sec = %v, want %v (calendar elapsed, not Pylon first_response_seconds)",
			payload.Scalars.MedianFirstResponseSec, wantFR)
	}
}

func TestGetWorkforceReportExcludesFourHourAutoReplyFR(t *testing.T) {
	db := testutil.NewDB(t, "report_workforce_fr_no_auto_ack")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}

	var contactID, inboxID, policyID int
	db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'Test', 'User') RETURNING id`)
	db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`)
	db.Get(&policyID, `INSERT INTO sla_policies (name, description, first_response_time, resolution_time, next_response_time)
		VALUES ('Default', '', 3600, 86400, 3600) RETURNING id`)

	now := time.Now().UTC()
	created := now.Add(-2 * time.Hour)
	metAuto := created.Add(4 * time.Hour)      // excluded band
	metHuman := created.Add(19*time.Hour + 15*time.Minute)

	var convAuto, convHuman int
	db.Get(&convAuto, `INSERT INTO conversations (contact_id, inbox_id, status_id, created_at, meta)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), $3, '{"pylon_number": 4005}') RETURNING id`,
		contactID, inboxID, created)
	db.Get(&convHuman, `INSERT INTO conversations (contact_id, inbox_id, status_id, created_at, meta)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), $3, '{"pylon_number": 4007}') RETURNING id`,
		contactID, inboxID, created)

	for _, pair := range []struct {
		convID int
		met    time.Time
	}{
		{convAuto, metAuto},
		{convHuman, metHuman},
	} {
		_, err = db.Exec(`INSERT INTO applied_slas (
				conversation_id, sla_policy_id, status, created_at,
				first_response_deadline_at, first_response_met_at
			) VALUES ($1, $2, 'met', $3, $4, $4)`,
			pair.convID, policyID, created, pair.met)
		if err != nil {
			t.Fatal(err)
		}
	}

	raw, err := m.GetWorkforceReport(IssuesRangeInput{Days: 7, AppTimezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Scalars struct {
			MedianFirstResponseSec float64 `json:"median_first_response_sec"`
		} `json:"scalars"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	want := metHuman.Sub(created).Seconds()
	if payload.Scalars.MedianFirstResponseSec != want {
		t.Fatalf("median_first_response_sec = %v, want %v (4h auto-reply excluded)", payload.Scalars.MedianFirstResponseSec, want)
	}
}

func TestGetWorkforceReportResolutionUsesPylonWorkingSeconds(t *testing.T) {
	db := testutil.NewDB(t, "report_workforce_res_working_charts")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}

	var contactID, inboxID, policyID int
	db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'Test', 'User') RETURNING id`)
	db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`)
	db.Get(&policyID, `INSERT INTO sla_policies (name, description, first_response_time, resolution_time, next_response_time)
		VALUES ('Default', '', 3600, 86400, 3600) RETURNING id`)

	now := time.Now().UTC()
	created := now.Add(-2 * time.Hour)
	metFr := created.Add(30 * time.Minute)
	resolved := created.Add(72 * time.Hour)
	const pylonRes = 518400 // 3d+ bucket when used for resolution
	var convID int
	db.Get(&convID, `INSERT INTO conversations (contact_id, inbox_id, status_id, created_at, resolved_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), $3, $4) RETURNING id`,
		contactID, inboxID, created, resolved)
	_, err = db.Exec(`INSERT INTO applied_slas (
			conversation_id, sla_policy_id, status, created_at,
			first_response_deadline_at, first_response_met_at,
			resolution_deadline_at, resolution_met_at, resolution_working_sec
		) VALUES ($1, $2, 'met', $3, $4, $5, $6, $7, $8)`,
		convID, policyID, created, metFr, metFr, resolved, resolved, pylonRes)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := m.GetWorkforceReport(IssuesRangeInput{Days: 7, AppTimezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Charts struct {
			ResolutionBuckets []struct {
				Label string `json:"label"`
				Count int    `json:"count"`
			} `json:"resolution_buckets"`
			MedianSLAByWeek []struct {
				MedianResolutionSec float64 `json:"median_resolution_sec"`
			} `json:"median_sla_by_week"`
			AssigneeOverview []struct {
				MedianResolutionSec float64 `json:"median_resolution_sec"`
			} `json:"assignee_overview"`
		} `json:"charts"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	var bucket3d int
	for _, b := range payload.Charts.ResolutionBuckets {
		if b.Label == "3d+" {
			bucket3d = b.Count
		}
	}
	if bucket3d != 1 {
		t.Fatalf("3d+ resolution bucket count = %d, want 1", bucket3d)
	}
	if len(payload.Charts.MedianSLAByWeek) == 0 || payload.Charts.MedianSLAByWeek[0].MedianResolutionSec != float64(pylonRes) {
		t.Fatalf("median_sla_by_week = %+v", payload.Charts.MedianSLAByWeek)
	}
	if len(payload.Charts.AssigneeOverview) != 1 || payload.Charts.AssigneeOverview[0].MedianResolutionSec != float64(pylonRes) {
		t.Fatalf("assignee_overview = %+v", payload.Charts.AssigneeOverview)
	}
}