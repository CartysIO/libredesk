package conversation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx/types"
	"github.com/zerodha/logf"
)

// TestConversationOrganization checks the Clyr fork's organization fields: a new conversation copies its
// contact's organization, reads return it, and list filters select by organization ID or external ID.
func TestConversationOrganization(t *testing.T) {
	db := testutil.NewDB(t, "conversation_organization")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), settingsStore: utcSettings{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}

	var acme, inbox, withOrg, withoutOrg int
	db.MustExec(`INSERT INTO organizations (name, external_id) VALUES ('Globex', 'team_2')`)
	if err := db.Get(&acme, `INSERT INTO organizations (name, external_id) VALUES ('Acme', 'team_1') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inbox, `INSERT INTO inboxes (name, channel) VALUES ('Chat', 'livechat') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&withOrg, `INSERT INTO users (type, first_name, last_name, email, organization_id) VALUES ('contact', 'Ada', '', 'ada@example.com', $1) RETURNING id`, acme); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&withoutOrg, `INSERT INTO users (type, first_name, last_name, email) VALUES ('contact', 'Bob', '', 'bob@example.com') RETURNING id`); err != nil {
		t.Fatal(err)
	}

	insert := func(contactID int, subject string) string {
		var id int
		var uuid string
		if err := m.q.InsertConversation.QueryRow(contactID, models.StatusOpen, inbox, "", time.Now(), subject, "", false, []byte("{}"), []byte("{}"), time.Time{}, 0, "").Scan(&id, &uuid); err != nil {
			t.Fatalf("inserting conversation: %v", err)
		}
		return uuid
	}
	acmeUUID := insert(withOrg, "acme")
	insert(withoutOrg, "none")

	conv, err := m.GetConversation(0, acmeUUID, "")
	if err != nil {
		t.Fatal(err)
	}
	if conv.OrganizationID.Int != acme || conv.OrganizationName.String != "Acme" {
		t.Fatalf("conversation organization = %v %v, want %d Acme", conv.OrganizationID, conv.OrganizationName, acme)
	}
	if conv.Contact.OrganizationID.Int != acme || conv.Contact.OrganizationExternalID.String != "team_1" || conv.Contact.OrganizationName.String != "Acme" {
		t.Fatalf("contact organization = %+v", conv.Contact)
	}

	filter := func(field, op, value string) string {
		b, _ := json.Marshal([]map[string]string{{"model": "conversations", "field": field, "operator": op, "value": value}})
		return string(b)
	}
	for _, tt := range []struct {
		name, filters string
		want          []string
	}{
		{"by id", filter("organization_id", "equals", jsonInt(acme)), []string{"acme"}},
		{"by external id", filter("organization_external_id", "equals", "team_1"), []string{"acme"}},
		{"not external id", filter("organization_external_id", "not equals", "team_1"), []string{"none"}},
		{"unknown external id", filter("organization_external_id", "equals", "nope"), nil},
		{"set", filter("organization_external_id", "set", ""), []string{"acme"}},
		{"not set", filter("organization_external_id", "not set", ""), []string{"none"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			list, err := m.GetAllConversationsList(0, "", "", tt.filters, 1, 50)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, c := range list {
				got = append(got, c.Subject.String)
				if c.Subject.String == "acme" && c.OrganizationName.String != "Acme" {
					t.Fatalf("list item organization name = %v", c.OrganizationName)
				}
			}
			if len(got) != len(tt.want) || (len(got) == 1 && got[0] != tt.want[0]) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
	if _, err := m.GetAllConversationsList(0, "", "", filter("organization_external_id", "contains", "x"), 1, 50); err == nil {
		t.Fatal("unsupported operator was accepted")
	}
}

// utcSettings has no app timezone, so date filters use UTC.
type utcSettings struct{ settingsStore }

func (utcSettings) Get(string) (types.JSONText, error) { return types.JSONText(`""`), nil }

func jsonInt(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
