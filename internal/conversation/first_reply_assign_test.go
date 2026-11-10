package conversation

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/logf"
)

// TestShouldAssignOnFirstReply checks which sent messages assign an unassigned conversation to the sender.
func TestShouldAssignOnFirstReply(t *testing.T) {
	const systemUserID = 1
	agent := umodels.User{ID: 7, Type: umodels.UserTypeAgent}
	reply := func(mutate func(*models.Message)) models.Message {
		m := models.Message{Type: models.MessageOutgoing, SenderType: models.SenderTypeAgent, Meta: json.RawMessage(`{}`)}
		if mutate != nil {
			mutate(&m)
		}
		return m
	}
	tests := []struct {
		name     string
		message  models.Message
		sender   umodels.User
		assigned bool
		want     bool
	}{
		{"agent reply on unassigned", reply(nil), agent, false, true},
		{"reply sent with a chosen status", reply(func(m *models.Message) { m.Meta = json.RawMessage(`{"keep_status":true}`) }), agent, false, true},
		{"already assigned", reply(nil), agent, true, false},
		{"private note", reply(func(m *models.Message) { m.Private = true }), agent, false, false},
		{"automated reply", reply(func(m *models.Message) { m.Meta = json.RawMessage(`{"is_automated":true}`) }), agent, false, false},
		{"csat message", reply(func(m *models.Message) { m.Meta = json.RawMessage(`{"is_csat":true}`) }), agent, false, false},
		{"continuity email", reply(func(m *models.Message) { m.Meta = json.RawMessage(`{"continuity_email":true}`) }), agent, false, false},
		{"incoming message", reply(func(m *models.Message) { m.Type = models.MessageIncoming }), agent, false, false},
		{"system user", reply(nil), umodels.User{ID: systemUserID, Type: umodels.UserTypeAgent}, false, false},
		{"ai assistant", reply(nil), umodels.User{ID: 9, Type: umodels.UserTypeAIAssistant}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldAssignOnFirstReply(tt.message, tt.sender, systemUserID, tt.assigned); got != tt.want {
				t.Fatalf("shouldAssignOnFirstReply = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestClaimUnassignedConversationQuery checks that the guarded claim assigns only a conversation that is still
// unassigned and still in the team the caller saw, including a conversation with no team (team id 0).
func TestClaimUnassignedConversationQuery(t *testing.T) {
	db := testutil.NewDB(t, "conversation_claim_unassigned")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), settingsStore: utcSettings{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	var inbox, contact, ann, bob, teamA, teamB int
	if err := db.Get(&inbox, `INSERT INTO inboxes (name, channel) VALUES ('Mail', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&contact, `INSERT INTO users (type, first_name, last_name, email) VALUES ('contact', 'Ada', '', 'ada@example.com') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	for _, a := range []struct {
		id    *int
		email string
	}{{&ann, "ann@example.com"}, {&bob, "bob@example.com"}} {
		if err := db.Get(a.id, `INSERT INTO users (type, first_name, last_name, email) VALUES ('agent', 'A', '', $1) RETURNING id`, a.email); err != nil {
			t.Fatal(err)
		}
	}
	for _, tm := range []struct {
		id   *int
		name string
	}{{&teamA, "A"}, {&teamB, "B"}} {
		if err := db.Get(tm.id, `INSERT INTO teams (name, emoji, conversation_assignment_type) VALUES ($1, '', 'Manual') RETURNING id`, tm.name); err != nil {
			t.Fatal(err)
		}
	}
	create := func(team, user int) string {
		var id int
		var uuid string
		if err := m.q.InsertConversation.QueryRow(contact, models.StatusOpen, inbox, "", time.Now(), "s", "", false, []byte("{}"), []byte("{}"), time.Time{}, 0, "", 0).Scan(&id, &uuid); err != nil {
			t.Fatal(err)
		}
		db.MustExec(`UPDATE conversations SET assigned_team_id = NULLIF($2, 0), assigned_user_id = NULLIF($3, 0) WHERE id = $1`, id, team, user)
		return uuid
	}
	assignee := func(uuid string) int {
		var u sql.NullInt64
		if err := db.Get(&u, `SELECT assigned_user_id FROM conversations WHERE uuid = $1`, uuid); err != nil {
			t.Fatal(err)
		}
		return int(u.Int64)
	}

	tests := []struct {
		name      string
		team      int // team stored on the conversation
		user      int // assignee stored on the conversation
		sawTeam   int // team the caller passes
		wantClaim bool
	}{
		{"unassigned, no team", 0, 0, 0, true},
		{"unassigned, team", teamA, 0, teamA, true},
		{"assigned concurrently", 0, bob, 0, false},
		{"moved to a team concurrently", teamB, 0, 0, false},
		{"moved to another team concurrently", teamB, 0, teamA, false},
		{"team removed concurrently", 0, 0, teamA, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uuid := create(tt.team, tt.user)
			res, err := m.q.ClaimUnassignedConversation.Exec(uuid, ann, tt.sawTeam)
			if err != nil {
				t.Fatal(err)
			}
			n, _ := res.RowsAffected()
			if got := n == 1; got != tt.wantClaim {
				t.Fatalf("claimed = %v, want %v", got, tt.wantClaim)
			}
			want := tt.user
			if tt.wantClaim {
				want = ann
			}
			if got := assignee(uuid); got != want {
				t.Fatalf("assignee = %d, want %d", got, want)
			}
		})
	}
}
