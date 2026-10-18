package organization

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func newTestManager(t *testing.T) (*Manager, *sqlx.DB) {
	t.Helper()
	db := testutil.NewDB(t, "organization")
	lo := logf.New(logf.Opts{})
	mgr, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatalf("creating organization manager: %v", err)
	}
	return mgr, db
}

func seedContact(t *testing.T, db *sqlx.DB, name string) int {
	t.Helper()
	var id int
	if err := db.Get(&id, `INSERT INTO users (type, first_name, email) VALUES ('contact', $1, $1 || '@example.com') RETURNING id`, name); err != nil {
		t.Fatalf("seeding contact: %v", err)
	}
	return id
}

func seedConversation(t *testing.T, db *sqlx.DB, contactID int, orgID null.Int) int {
	t.Helper()
	var inboxID, id int
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'livechat') RETURNING id`); err != nil {
		t.Fatalf("seeding inbox: %v", err)
	}
	if err := db.Get(&id, `INSERT INTO conversations (contact_id, inbox_id, status_id, organization_id) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1), $3) RETURNING id`, contactID, inboxID, orgID); err != nil {
		t.Fatalf("seeding conversation: %v", err)
	}
	return id
}

func errType(err error) string {
	if e, ok := err.(envelope.Error); ok {
		return e.ErrorType
	}
	return ""
}

func TestCreateUpdateAndValidate(t *testing.T) {
	mgr, db := newTestManager(t)

	org, err := mgr.Create("  Acme  ", " team_1 ")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if org.Name != "Acme" || org.ExternalID.String != "team_1" {
		t.Fatalf("Create did not trim: %+v", org)
	}
	noExt, err := mgr.Create("No external ID", "")
	if err != nil || noExt.ExternalID.Valid {
		t.Fatalf("Create without external ID = %+v, %v; want NULL external ID", noExt, err)
	}
	if _, err := mgr.Create("Another without ID", ""); err != nil {
		t.Fatalf("a second organization without external ID must be allowed: %v", err)
	}
	if _, err := mgr.Create("Copy", "team_1"); errType(err) != envelope.ConflictError {
		t.Fatalf("duplicate external ID: err = %v, want conflict", err)
	}
	if _, err := mgr.Create(" ", "x"); errType(err) != envelope.InputError {
		t.Fatalf("empty name: err = %v, want input error", err)
	}

	var contact int
	if err := db.Get(&contact, `INSERT INTO users (type, first_name, organization_id) VALUES ('contact', 'Ada', $1) RETURNING id`, org.ID); err != nil {
		t.Fatal(err)
	}
	updated, err := mgr.Update(org.ID, "Acme Inc", "team_1")
	if err != nil || updated.Name != "Acme Inc" || updated.ContactCount != 1 {
		t.Fatalf("Update = %+v, %v; want the new name and contact_count 1", updated, err)
	}
	if _, err := mgr.Update(noExt.ID, "No external ID", "team_1"); errType(err) != envelope.ConflictError {
		t.Fatalf("update to a used external ID: err = %v, want conflict", err)
	}
	if _, err := mgr.Update(999999, "Missing", ""); errType(err) != envelope.NotFoundError {
		t.Fatalf("update of a missing organization: err = %v, want not found", err)
	}
	got, err := mgr.GetByExternalID("team_1")
	if err != nil || got.ID != org.ID {
		t.Fatalf("GetByExternalID = %+v, %v", got, err)
	}
	if _, err := mgr.Get(999999); errType(err) != envelope.NotFoundError {
		t.Fatalf("Get of a missing organization: err = %v, want not found", err)
	}
}

func TestUpsertByExternalID(t *testing.T) {
	mgr, _ := newTestManager(t)

	created, err := mgr.UpsertByExternalID("team_9", "")
	if err != nil || created.Name != "team_9" {
		t.Fatalf("new organization without name = %+v, %v; want the external ID as name", created, err)
	}
	renamed, err := mgr.UpsertByExternalID("team_9", "Globex")
	if err != nil || renamed.ID != created.ID || renamed.Name != "Globex" {
		t.Fatalf("rename = %+v, %v", renamed, err)
	}
	kept, err := mgr.UpsertByExternalID("team_9", "")
	if err != nil || kept.Name != "Globex" || !kept.UpdatedAt.Equal(renamed.UpdatedAt) {
		t.Fatalf("empty name must keep name and updated_at: %+v, %v", kept, err)
	}
	if _, err := mgr.UpsertByExternalID(" ", "Name"); errType(err) != envelope.InputError {
		t.Fatalf("empty external ID: err = %v, want input error", err)
	}
}

func TestSetContactOrganization(t *testing.T) {
	mgr, db := newTestManager(t)
	acme, _ := mgr.Create("Acme", "team_1")
	globex, _ := mgr.Create("Globex", "team_2")
	contact := seedContact(t, db, "ada")
	open := seedConversation(t, db, contact, null.Int{})
	old := seedConversation(t, db, contact, null.IntFrom(globex.ID))

	if err := mgr.SetContactOrganization(contact, null.IntFrom(acme.ID)); err != nil {
		t.Fatalf("SetContactOrganization: %v", err)
	}
	var contactOrg null.Int
	db.Get(&contactOrg, `SELECT organization_id FROM users WHERE id = $1`, contact)
	if contactOrg.Int != acme.ID {
		t.Fatalf("contact organization = %v, want %d", contactOrg, acme.ID)
	}
	var openOrg, oldOrg null.Int
	db.Get(&openOrg, `SELECT organization_id FROM conversations WHERE id = $1`, open)
	db.Get(&oldOrg, `SELECT organization_id FROM conversations WHERE id = $1`, old)
	if openOrg.Int != acme.ID {
		t.Fatalf("conversation without organization = %v, want %d", openOrg, acme.ID)
	}
	if oldOrg.Int != globex.ID {
		t.Fatalf("conversation with an organization changed to %v, want it kept at %d", oldOrg, globex.ID)
	}

	contacts, err := mgr.GetContacts(acme.ID, 1, 10)
	if err != nil || len(contacts) != 1 || contacts[0].ID != contact {
		t.Fatalf("GetContacts = %+v, %v", contacts, err)
	}
	all, total, err := mgr.GetAll("acm", 1, 10)
	if err != nil || total != 1 || all[0].ContactCount != 1 {
		t.Fatalf("GetAll search = %+v, total %d, %v", all, total, err)
	}
	pastEnd, total, err := mgr.GetAll("", 5, 10)
	if err != nil || len(pastEnd) != 0 || total != 2 {
		t.Fatalf("page past the end = %d rows, total %d, %v; want 0 rows, total 2", len(pastEnd), total, err)
	}
	if _, err := mgr.GetContacts(999999, 1, 10); errType(err) != envelope.NotFoundError {
		t.Fatalf("contacts of a missing organization: err = %v, want not found", err)
	}
	byExt, _, _ := mgr.GetAll("team_2", 1, 10)
	if len(byExt) != 1 || byExt[0].ID != globex.ID {
		t.Fatalf("GetAll by external ID = %+v", byExt)
	}

	if err := mgr.SetContactOrganization(contact, null.IntFrom(999999)); errType(err) != envelope.NotFoundError {
		t.Fatalf("unknown organization: err = %v, want not found", err)
	}
	if err := mgr.SetContactOrganization(999999, null.IntFrom(acme.ID)); errType(err) != envelope.NotFoundError {
		t.Fatalf("unknown contact: err = %v, want not found", err)
	}
	var agent int
	db.Get(&agent, `INSERT INTO users (type, first_name, email) VALUES ('agent', 'Agent', 'agent@example.com') RETURNING id`)
	if err := mgr.SetContactOrganization(agent, null.IntFrom(acme.ID)); errType(err) != envelope.NotFoundError {
		t.Fatalf("agent: err = %v, want not found", err)
	}

	if err := mgr.SetContactOrganization(contact, null.Int{}); err != nil {
		t.Fatalf("removing organization: %v", err)
	}
	db.Get(&contactOrg, `SELECT organization_id FROM users WHERE id = $1`, contact)
	db.Get(&openOrg, `SELECT organization_id FROM conversations WHERE id = $1`, open)
	if contactOrg.Valid || openOrg.Int != acme.ID {
		t.Fatalf("after removal: contact org %v (want NULL), conversation org %v (want kept %d)", contactOrg, openOrg, acme.ID)
	}
}

func TestSetContactOrganizationIfNone(t *testing.T) {
	mgr, db := newTestManager(t)
	acme, _ := mgr.Create("Acme", "team_1")
	globex, _ := mgr.Create("Globex", "team_2")
	contact := seedContact(t, db, "ada")
	open := seedConversation(t, db, contact, null.Int{})

	linked, err := mgr.SetContactOrganizationIfNone(contact, acme.ID)
	if err != nil || !linked {
		t.Fatalf("first link = %v, %v; want true", linked, err)
	}
	linked, err = mgr.SetContactOrganizationIfNone(contact, globex.ID)
	if err != nil || linked {
		t.Fatalf("second link = %v, %v; want false (contact already has one)", linked, err)
	}
	var contactOrg, openOrg null.Int
	db.Get(&contactOrg, `SELECT organization_id FROM users WHERE id = $1`, contact)
	db.Get(&openOrg, `SELECT organization_id FROM conversations WHERE id = $1`, open)
	if contactOrg.Int != acme.ID || openOrg.Int != acme.ID {
		t.Fatalf("contact org %v, conversation org %v; want both %d", contactOrg, openOrg, acme.ID)
	}

	if linked, err := mgr.SetContactOrganizationIfNone(999999, acme.ID); err != nil || linked {
		t.Fatalf("unknown contact = %v, %v; want false, nil", linked, err)
	}
	other := seedContact(t, db, "bob")
	if _, err := mgr.SetContactOrganizationIfNone(other, 999999); errType(err) != envelope.NotFoundError {
		t.Fatalf("unknown organization: err = %v, want not found", err)
	}
}

func TestDeleteKeepsContactsAndConversations(t *testing.T) {
	mgr, db := newTestManager(t)
	acme, _ := mgr.Create("Acme", "team_1")
	contact := seedContact(t, db, "bob")
	conv := seedConversation(t, db, contact, null.Int{})
	if err := mgr.SetContactOrganization(contact, null.IntFrom(acme.ID)); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Delete(acme.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var contactOrg, convOrg null.Int
	if err := db.Get(&contactOrg, `SELECT organization_id FROM users WHERE id = $1`, contact); err != nil {
		t.Fatalf("contact is gone: %v", err)
	}
	if err := db.Get(&convOrg, `SELECT organization_id FROM conversations WHERE id = $1`, conv); err != nil {
		t.Fatalf("conversation is gone: %v", err)
	}
	if contactOrg.Valid || convOrg.Valid {
		t.Fatalf("organization IDs not cleared: contact %v, conversation %v", contactOrg, convOrg)
	}
	if err := mgr.Delete(acme.ID); errType(err) != envelope.NotFoundError {
		t.Fatalf("second delete: err = %v, want not found", err)
	}
}
