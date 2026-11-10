package migrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC9_CLYR6 removes the Closed status (Clyr fork). Resolved does the same job and also sends
// CSAT and tells the AI agent, so Closed conversations move to Resolved, every stored reference to
// Closed (automation rules, macros, views) points to Resolved, and the Closed row is deleted.
// It also turns off the live-chat "prevent reply to closed conversation" settings. It is idempotent,
// and on a fresh install (no Closed status, no inboxes) it changes nothing.
func V2_9_0_RC9_CLYR6(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// The UI no longer shows the "prevent reply to closed conversation" switches, so turn them off.
	// Otherwise a status an admin later names "Closed" would block chat replies with no way to undo it.
	for _, audience := range []string{"visitors", "users"} {
		if _, err := tx.Exec(`UPDATE inboxes
			SET config = jsonb_set(config, ARRAY[$1::text, 'prevent_reply_to_closed_conversation'], 'false'::jsonb)
			WHERE channel = 'livechat' AND config->($1::text)->'prevent_reply_to_closed_conversation' = 'true'::jsonb`, audience); err != nil {
			return err
		}
	}

	var closedID, resolvedID sql.NullInt64
	if err := tx.QueryRow(`SELECT
		(SELECT id FROM conversation_statuses WHERE name = 'Closed'),
		(SELECT id FROM conversation_statuses WHERE name = 'Resolved')`).Scan(&closedID, &resolvedID); err != nil {
		return err
	}
	if !closedID.Valid {
		return tx.Commit()
	}
	if !resolvedID.Valid {
		return fmt.Errorf("cannot remove the Closed status: there is no Resolved status to move its conversations to")
	}

	// A closed conversation keeps its close time as its resolve time.
	if _, err := tx.Exec(`UPDATE conversations SET
		status_id        = $2,
		resolved_at      = COALESCE(resolved_at, closed_at, updated_at),
		last_resolved_at = COALESCE(last_resolved_at, closed_at, resolved_at, updated_at)
		WHERE status_id = $1`, closedID.Int64, resolvedID.Int64); err != nil {
		return err
	}

	from, to := strconv.FormatInt(closedID.Int64, 10), strconv.FormatInt(resolvedID.Int64, 10)
	for _, col := range []struct{ table, column string }{
		{"automation_rules", "rules"},
		{"macros", "actions"},
		{"views", "filters"},
	} {
		if err := repointClosedJSON(tx, col.table, col.column, from, to); err != nil {
			return fmt.Errorf("repointing Closed in %s: %w", col.table, err)
		}
	}

	if _, err := tx.Exec(`DELETE FROM conversation_statuses WHERE id = $1`, closedID.Int64); err != nil {
		return err
	}
	return tx.Commit()
}

// repointClosedJSON rewrites the Closed status id (from) to the Resolved id (to) in one JSONB column.
func repointClosedJSON(tx *sqlx.Tx, table, column, from, to string) error {
	var rows []struct {
		ID   int    `db:"id"`
		JSON []byte `db:"j"`
	}
	if err := tx.Select(&rows, fmt.Sprintf(`SELECT id, %s AS j FROM %s WHERE %s IS NOT NULL`, column, table, column)); err != nil {
		return err
	}
	for _, r := range rows {
		dec := json.NewDecoder(strings.NewReader(string(r.JSON)))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return fmt.Errorf("row %d: %w", r.ID, err)
		}
		if !repointClosedRefs(v, from, to) {
			continue
		}
		out, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(fmt.Sprintf(`UPDATE %s SET %s = $1 WHERE id = $2`, table, column), out, r.ID); err != nil {
			return err
		}
	}
	return nil
}

// repointClosedRefs walks a rule, macro or view JSON value and rewrites the status references in it:
//   - set_status actions: {"type":"set_status","value":["<id>"]}
//   - status conditions in automation rules: {"field":"status"|"previous_status","value":"<id>" or "<id>,<id>"}
//   - view filters: {"field":"status_id","value":"<id>" or "[\"<id>\"]"} and
//     {"model":"conversation_statuses","field":"name","value":"Closed"}
//
// It reports whether anything changed.
func repointClosedRefs(v any, from, to string) bool {
	changed := false
	switch n := v.(type) {
	case []any:
		for _, e := range n {
			changed = repointClosedRefs(e, from, to) || changed
		}
	case map[string]any:
		if n["type"] == "set_status" {
			if vals, ok := n["value"].([]any); ok {
				for i, s := range vals {
					if s == from {
						vals[i] = to
						changed = true
					}
				}
			}
		}
		if s, ok := n["value"].(string); ok {
			field, _ := n["field"].(string)
			model, _ := n["model"].(string)
			fieldType, _ := n["field_type"].(string)
			var ns string
			switch {
			case model == "conversation_statuses" && field == "name":
				ns = replaceToken(s, "Closed", "Resolved")
			case model == "conversation_statuses":
				ns = s
			case (field == "status" || field == "previous_status") && (fieldType == "" || fieldType == "conversation"):
				ns = replaceToken(s, from, to)
			case field == "status_id" && (model == "" || model == "conversations"):
				ns = replaceToken(s, from, to)
			default:
				ns = s
			}
			if ns != s {
				n["value"] = ns
				changed = true
			}
		}
		for _, e := range n {
			changed = repointClosedRefs(e, from, to) || changed
		}
	}
	return changed
}

// replaceToken replaces from with to in a single value, a JSON string array ("in" operator) or a
// comma-separated list ("contains" operator). Only whole tokens match, so id 4 never touches 14.
func replaceToken(s, from, to string) string {
	if strings.TrimSpace(s) == from {
		return to
	}
	if strings.HasPrefix(strings.TrimSpace(s), "[") {
		var arr []string
		if json.Unmarshal([]byte(s), &arr) != nil {
			return s
		}
		hit := false
		for i := range arr {
			if arr[i] == from {
				arr[i] = to
				hit = true
			}
		}
		if !hit {
			return s
		}
		out, _ := json.Marshal(arr)
		return string(out)
	}
	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		hit := false
		for i, p := range parts {
			if strings.TrimSpace(p) == from {
				parts[i] = to
				hit = true
			}
		}
		if hit {
			return strings.Join(parts, ",")
		}
	}
	return s
}
