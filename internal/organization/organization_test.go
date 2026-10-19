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

	contact := seedContact(t, db, "ada")
	if _, err := mgr.AddContactOrganization(contact, org.ID, false); err != nil {
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

func contactDefault(t *testing.T, db *sqlx.DB, contactID int) null.Int {
	t.Helper()
	var org null.Int
	if err := db.Get(&org, `SELECT organization_id FROM users WHERE id = $1`, contactID); err != nil {
		t.Fatal(err)
	}
	return org
}

func conversationOrg(t *testing.T, db *sqlx.DB, id int) null.Int {
	t.Helper()
	var org null.Int
	if err := db.Get(&org, `SELECT organization_id FROM conversations WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	return org
}

func TestAddContactOrganization(t *testing.T) {
	mgr, db := newTestManager(t)
	acme, _ := mgr.Create("Acme", "team_1")
	globex, _ := mgr.Create("Globex", "team_2")
	initech, _ := mgr.Create("Initech", "team_3")
	contact := seedContact(t, db, "ada")
	open := seedConversation(t, db, contact, null.Int{})
	old := seedConversation(t, db, contact, null.IntFrom(initech.ID))

	// The first organization becomes the default, and conversations without an organization get it.
	change, err := mgr.AddContactOrganization(contact, acme.ID, false)
	if err != nil || change.Added != 1 || change.DefaultChanged != 1 {
		t.Fatalf("first add = %+v, %v; want added and default changed", change, err)
	}
	var openUUID string
	db.Get(&openUUID, `SELECT uuid FROM conversations WHERE id = $1`, open)
	if len(change.Filed) != 1 || change.Filed[0] != openUUID {
		t.Fatalf("filed = %v, want only the conversation without organization (%s)", change.Filed, openUUID)
	}
	if got := contactDefault(t, db, contact); got.Int != acme.ID {
		t.Fatalf("default = %v, want %d", got, acme.ID)
	}
	if got := conversationOrg(t, db, open); got.Int != acme.ID {
		t.Fatalf("conversation without organization = %v, want %d", got, acme.ID)
	}
	if got := conversationOrg(t, db, old); got.Int != initech.ID {
		t.Fatalf("conversation with an organization changed to %v, want it kept at %d", got, initech.ID)
	}

	// A second organization is added without changing the default.
	change, err = mgr.AddContactOrganization(contact, globex.ID, false)
	if err != nil || change.Added != 1 || change.DefaultChanged != 0 {
		t.Fatalf("second add = %+v, %v; want added, default kept", change, err)
	}
	if len(change.Filed) != 0 {
		t.Fatalf("second add filed %v, want none", change.Filed)
	}
	if got := contactDefault(t, db, contact); got.Int != acme.ID {
		t.Fatalf("default = %v, want it kept at %d", got, acme.ID)
	}

	// Adding again is a no-op; with makeDefault it only changes the default.
	change, err = mgr.AddContactOrganization(contact, globex.ID, true)
	if err != nil || change.Added != 0 || change.DefaultChanged != 1 {
		t.Fatalf("make default = %+v, %v; want only the default changed", change, err)
	}
	change, err = mgr.AddContactOrganization(contact, globex.ID, true)
	if err != nil || change.Added != 0 || change.DefaultChanged != 0 {
		t.Fatalf("repeat = %+v, %v; want no change", change, err)
	}

	orgs, err := mgr.GetContactOrganizations(contact)
	if err != nil || len(orgs) != 2 || orgs[0].ID != globex.ID || !orgs[0].IsDefault || orgs[1].ID != acme.ID || orgs[1].IsDefault {
		t.Fatalf("GetContactOrganizations = %+v, %v; want Globex (default) then Acme", orgs, err)
	}
	for _, o := range []struct {
		id        int
		isDefault bool
	}{{acme.ID, false}, {globex.ID, true}} {
		contacts, err := mgr.GetContacts(o.id, 1, 10)
		if err != nil || len(contacts) != 1 || contacts[0].ID != contact || contacts[0].IsDefault != o.isDefault {
			t.Fatalf("GetContacts(%d) = %+v, %v", o.id, contacts, err)
		}
	}
	all, total, err := mgr.GetAll("", 1, 10)
	if err != nil || total != 3 {
		t.Fatalf("GetAll = %+v, total %d, %v", all, total, err)
	}
	for _, o := range all {
		want := 1
		if o.ID == initech.ID {
			want = 0
		}
		if o.ContactCount != want {
			t.Fatalf("%s has contact_count %d, want %d", o.Name, o.ContactCount, want)
		}
	}
	pastEnd, total, err := mgr.GetAll("", 5, 10)
	if err != nil || len(pastEnd) != 0 || total != 3 {
		t.Fatalf("page past the end = %d rows, total %d, %v; want 0 rows, total 3", len(pastEnd), total, err)
	}
	byExt, _, _ := mgr.GetAll("team_2", 1, 10)
	if len(byExt) != 1 || byExt[0].ID != globex.ID {
		t.Fatalf("GetAll by external ID = %+v", byExt)
	}
	if _, err := mgr.GetContacts(999999, 1, 10); errType(err) != envelope.NotFoundError {
		t.Fatalf("contacts of a missing organization: err = %v, want not found", err)
	}

	if _, err := mgr.AddContactOrganization(contact, 999999, false); errType(err) != envelope.NotFoundError {
		t.Fatalf("unknown organization: err = %v, want not found", err)
	}
	if _, err := mgr.AddContactOrganization(999999, acme.ID, false); errType(err) != envelope.NotFoundError {
		t.Fatalf("unknown contact: err = %v, want not found", err)
	}
	var agent int
	db.Get(&agent, `INSERT INTO users (type, first_name, email) VALUES ('agent', 'Agent', 'agent@example.com') RETURNING id`)
	if _, err := mgr.AddContactOrganization(agent, acme.ID, false); errType(err) != envelope.NotFoundError {
		t.Fatalf("agent: err = %v, want not found", err)
	}
}

func TestRemoveContactOrganization(t *testing.T) {
	mgr, db := newTestManager(t)
	acme, _ := mgr.Create("Acme", "team_1")
	globex, _ := mgr.Create("Globex", "team_2")
	contact := seedContact(t, db, "ada")
	conv := seedConversation(t, db, contact, null.Int{})
	mgr.AddContactOrganization(contact, acme.ID, false)
	mgr.AddContactOrganization(contact, globex.ID, false)

	// Removing a membership that is not the default keeps the default.
	change, err := mgr.RemoveContactOrganization(contact, globex.ID)
	if err != nil || change.DefaultChanged != 0 {
		t.Fatalf("remove other = %+v, %v", change, err)
	}
	mgr.AddContactOrganization(contact, globex.ID, false)

	// Removing the default makes the other membership the default. The conversation keeps its organization.
	change, err = mgr.RemoveContactOrganization(contact, acme.ID)
	if err != nil || change.DefaultChanged != 1 {
		t.Fatalf("remove default = %+v, %v", change, err)
	}
	if got := contactDefault(t, db, contact); got.Int != globex.ID {
		t.Fatalf("default = %v, want %d", got, globex.ID)
	}
	if got := conversationOrg(t, db, conv); got.Int != acme.ID {
		t.Fatalf("conversation organization = %v, want it kept at %d", got, acme.ID)
	}

	// Removing the last membership leaves no default.
	if _, err := mgr.RemoveContactOrganization(contact, globex.ID); err != nil {
		t.Fatal(err)
	}
	if got := contactDefault(t, db, contact); got.Valid {
		t.Fatalf("default = %v, want NULL", got)
	}
	if _, err := mgr.RemoveContactOrganization(contact, globex.ID); errType(err) != envelope.NotFoundError {
		t.Fatalf("removing a missing membership: err = %v, want not found", err)
	}
}

func TestSetConversationOrganization(t *testing.T) {
	mgr, db := newTestManager(t)
	acme, _ := mgr.Create("Acme", "team_1")
	globex, _ := mgr.Create("Globex", "team_2")
	other, _ := mgr.Create("Other", "team_3")
	contact := seedContact(t, db, "ada")
	mgr.AddContactOrganization(contact, acme.ID, false)
	mgr.AddContactOrganization(contact, globex.ID, false)
	conv := seedConversation(t, db, contact, null.IntFrom(acme.ID))
	var uuid string
	db.Get(&uuid, `SELECT uuid FROM conversations WHERE id = $1`, conv)

	if err := mgr.SetConversationOrganization(uuid, null.IntFrom(globex.ID)); err != nil {
		t.Fatalf("set to a contact organization: %v", err)
	}
	if got := conversationOrg(t, db, conv); got.Int != globex.ID {
		t.Fatalf("conversation organization = %v, want %d", got, globex.ID)
	}
	if err := mgr.SetConversationOrganization(uuid, null.IntFrom(other.ID)); errType(err) != envelope.InputError {
		t.Fatalf("organization the contact is not in: err = %v, want input error", err)
	}
	if got := conversationOrg(t, db, conv); got.Int != globex.ID {
		t.Fatalf("rejected change still changed the organization to %v", got)
	}
	if err := mgr.SetConversationOrganization(uuid, null.Int{}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got := conversationOrg(t, db, conv); got.Valid {
		t.Fatalf("conversation organization = %v, want NULL", got)
	}
}

func TestDeleteReassignsDefault(t *testing.T) {
	mgr, db := newTestManager(t)
	acme, _ := mgr.Create("Acme", "team_1")
	globex, _ := mgr.Create("Globex", "team_2")
	both := seedContact(t, db, "ada")
	only := seedContact(t, db, "bob")
	conv := seedConversation(t, db, only, null.Int{})
	mgr.AddContactOrganization(both, acme.ID, false)
	mgr.AddContactOrganization(both, globex.ID, false)
	mgr.AddContactOrganization(only, acme.ID, false)

	changed, err := mgr.Delete(acme.ID)
	if err != nil || len(changed) != 2 {
		t.Fatalf("Delete = %+v, %v; want 2 contacts with a new default", changed, err)
	}
	if got := contactDefault(t, db, both); got.Int != globex.ID {
		t.Fatalf("contact in two organizations: default = %v, want %d", got, globex.ID)
	}
	if got := contactDefault(t, db, only); got.Valid {
		t.Fatalf("contact in one organization: default = %v, want NULL", got)
	}
	if got := conversationOrg(t, db, conv); got.Valid {
		t.Fatalf("conversation organization = %v, want NULL", got)
	}
	orgs, _ := mgr.GetContactOrganizations(both)
	if len(orgs) != 1 || orgs[0].ID != globex.ID {
		t.Fatalf("memberships after delete = %+v", orgs)
	}
	if _, err := mgr.Delete(acme.ID); errType(err) != envelope.NotFoundError {
		t.Fatalf("second delete: err = %v, want not found", err)
	}
}
