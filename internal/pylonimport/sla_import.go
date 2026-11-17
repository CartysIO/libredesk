package pylonimport

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type slaPolicyRow struct {
	FirstResponse string `db:"first_response_time"`
	Resolution    string `db:"resolution_time"`
}

// simulateImportedSLA inserts a settled applied_slas row so reports and overview match native conversations.
// Deadlines use simple ParseDuration on the policy; business-hours alignment with sla.GetDeadlines is follow-up work.
func (im *Importer) simulateImportedSLA(tx *sqlx.Tx, convID int, is Issue, createdAt time.Time, statusCategory string) error {
	if im.Opt.SLAPolicyID <= 0 {
		return nil
	}
	var has bool
	if err := tx.Get(&has, `SELECT EXISTS (SELECT 1 FROM applied_slas WHERE conversation_id = $1)`, convID); err != nil {
		return err
	}
	if has {
		return nil
	}
	var pol slaPolicyRow
	if err := tx.Get(&pol, `SELECT first_response_time, resolution_time FROM sla_policies WHERE id = $1`, im.Opt.SLAPolicyID); err != nil {
		return fmt.Errorf("sla policy %d: %w", im.Opt.SLAPolicyID, err)
	}
	frDeadline, err := addPolicyDuration(createdAt, pol.FirstResponse)
	if err != nil {
		return err
	}
	resDeadline, err := addPolicyDuration(createdAt, pol.Resolution)
	if err != nil {
		return err
	}

	var appliedID int
	if err := tx.Get(&appliedID, `INSERT INTO applied_slas (
			created_at, updated_at, conversation_id, sla_policy_id,
			first_response_deadline_at, resolution_deadline_at, status)
		VALUES ($1, $1, $2, $3, $4, $5, 'pending')
		RETURNING id`,
		createdAt, convID, im.Opt.SLAPolicyID,
		nullTime(frDeadline), nullTime(resDeadline),
	); err != nil {
		return fmt.Errorf("insert applied_slas: %w", err)
	}

	var (
		frMet, frBreach     sql.NullTime
		resMet, resBreach   sql.NullTime
		firstReply, resolved *time.Time
	)
	if is.FirstResponseTime != nil {
		firstReply = is.FirstResponseTime
	}
	if statusCategory == "resolved" && is.ResolutionTime != nil {
		resolved = is.ResolutionTime
	}
	stampOutcome(firstReply, frDeadline, &frMet, &frBreach)
	stampOutcome(resolved, resDeadline, &resMet, &resBreach)

	status := appliedSLAStatus(frMet, frBreach, resMet, resBreach, statusCategory)

	if _, err := tx.Exec(`UPDATE applied_slas SET
			first_response_met_at = $2,
			first_response_breached_at = $3,
			resolution_met_at = $4,
			resolution_breached_at = $5,
			status = $6::applied_sla_status,
			updated_at = $1
		WHERE id = $7`,
		createdAt, frMet, frBreach, resMet, resBreach, status, appliedID,
	); err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE conversations SET
			sla_policy_id = $2,
			next_sla_deadline_at = CASE WHEN $3 = 'resolved' THEN NULL ELSE LEAST($4::timestamptz, $5::timestamptz) END,
			updated_at = GREATEST(updated_at, $1)
		WHERE id = $6`,
		createdAt, im.Opt.SLAPolicyID, statusCategory,
		nullTimePtr(frDeadline), nullTimePtr(resDeadline), convID,
	)
	return err
}

func addPolicyDuration(start time.Time, dur string) (time.Time, error) {
	dur = strings.TrimSpace(dur)
	if dur == "" {
		return time.Time{}, nil
	}
	d, err := time.ParseDuration(dur)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse SLA duration %q: %w", dur, err)
	}
	return start.Add(d), nil
}

func stampOutcome(actual *time.Time, deadline time.Time, met, breach *sql.NullTime) {
	if actual == nil || actual.IsZero() {
		return
	}
	t := actual.UTC()
	if deadline.IsZero() || !t.After(deadline) {
		*met = sql.NullTime{Time: t, Valid: true}
		return
	}
	*breach = sql.NullTime{Time: t, Valid: true}
}

func appliedSLAStatus(frMet, frBreach, resMet, resBreach sql.NullTime, statusCategory string) string {
	if statusCategory != "resolved" {
		if !frMet.Valid && !frBreach.Valid {
			return "pending"
		}
		if frBreach.Valid {
			return "partially_met"
		}
		return "pending"
	}
	if !frBreach.Valid && !resBreach.Valid {
		return "met"
	}
	if !frMet.Valid && !resMet.Valid {
		return "breached"
	}
	return "partially_met"
}

func nullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func nullTimePtr(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}