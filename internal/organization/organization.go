// Package organization manages organizations: the customer companies that contacts belong to.
// It is a Clyr fork addition. A contact can belong to several organizations; one of them is its
// default (users.organization_id), which new email conversations get. A conversation keeps its
// organization until an agent changes it.
package organization

import (
	"database/sql"
	"embed"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/organization/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

const (
	maxNameLength       = 255
	maxExternalIDLength = 128
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles organization operations.
type Manager struct {
	db   *sqlx.DB
	lo   *logf.Logger
	i18n *i18n.I18n
	q    queries
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

type queries struct {
	CountOrganizations           *sqlx.Stmt `query:"count-organizations"`
	GetOrganizations             *sqlx.Stmt `query:"get-organizations"`
	GetOrganization              *sqlx.Stmt `query:"get-organization"`
	GetOrganizationByExternalID  *sqlx.Stmt `query:"get-organization-by-external-id"`
	InsertOrganization           *sqlx.Stmt `query:"insert-organization"`
	UpdateOrganization           *sqlx.Stmt `query:"update-organization"`
	UpsertOrganizationByExternal *sqlx.Stmt `query:"upsert-organization-by-external-id"`
	DeleteOrganization           *sqlx.Stmt `query:"delete-organization"`
	GetOrganizationContacts      *sqlx.Stmt `query:"get-organization-contacts"`
	GetContactOrganizations      *sqlx.Stmt `query:"get-contact-organizations"`
	AddContactOrganization       *sqlx.Stmt `query:"add-contact-organization"`
	RemoveContactOrganization    *sqlx.Stmt `query:"remove-contact-organization"`
	ReassignDefaultOrganization  *sqlx.Stmt `query:"reassign-default-organization"`
	SetConversationOrganization  *sqlx.Stmt `query:"set-conversation-organization"`
	SetOrganizationFields        *sqlx.Stmt `query:"set-organization-fields"`
	AssignOrganizationByDomain   *sqlx.Stmt `query:"assign-organization-by-domain"`
}

// New creates and returns a new Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{q: q, db: opts.DB, lo: opts.Lo, i18n: opts.I18n}, nil
}

// GetAll returns organizations whose name contains query or whose external ID equals it, ordered by name.
func (m *Manager) GetAll(query string, page, pageSize int) ([]models.Organization, int, error) {
	var (
		orgs  = make([]models.Organization, 0)
		total int
	)
	query = strings.TrimSpace(query)
	// Count separately, so a page past the last one still reports the total.
	if err := m.q.CountOrganizations.Get(&total, query); err != nil {
		m.lo.Error("error counting organizations", "error", err)
		return orgs, 0, m.generalError()
	}
	if err := m.q.GetOrganizations.Select(&orgs, query, pageSize, dbutil.PageOffset(page, pageSize)); err != nil {
		m.lo.Error("error fetching organizations", "error", err)
		return orgs, 0, m.generalError()
	}
	return orgs, total, nil
}

// Get returns an organization by ID.
func (m *Manager) Get(id int) (models.Organization, error) {
	var org models.Organization
	if err := m.q.GetOrganization.Get(&org, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return org, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundOrganization"), nil)
		}
		m.lo.Error("error fetching organization", "id", id, "error", err)
		return org, m.generalError()
	}
	return org, nil
}

// GetByExternalID returns an organization by its external ID.
func (m *Manager) GetByExternalID(externalID string) (models.Organization, error) {
	var org models.Organization
	if err := m.q.GetOrganizationByExternalID.Get(&org, externalID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return org, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundOrganization"), nil)
		}
		m.lo.Error("error fetching organization by external id", "external_id", externalID, "error", err)
		return org, m.generalError()
	}
	return org, nil
}

// Create creates an organization. An empty external ID is stored as NULL.
func (m *Manager) Create(name, externalID string) (models.Organization, error) {
	var org models.Organization
	name, extID, err := m.validate(name, externalID)
	if err != nil {
		return org, err
	}
	if err := m.q.InsertOrganization.Get(&org, name, extID); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return org, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsOrganization"), nil)
		}
		m.lo.Error("error inserting organization", "error", err)
		return org, m.generalError()
	}
	return org, nil
}

// Update changes an organization's name and external ID.
func (m *Manager) Update(id int, name, externalID string) (models.Organization, error) {
	var org models.Organization
	name, extID, err := m.validate(name, externalID)
	if err != nil {
		return org, err
	}
	if err := m.q.UpdateOrganization.Get(&org, id, name, extID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return org, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundOrganization"), nil)
		}
		if dbutil.IsUniqueViolationError(err) {
			return org, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsOrganization"), nil)
		}
		m.lo.Error("error updating organization", "id", id, "error", err)
		return org, m.generalError()
	}
	return m.Get(org.ID)
}

// UpsertByExternalID creates the organization with this external ID, or renames it when name is not empty.
// A new organization without a name is named after its external ID.
func (m *Manager) UpsertByExternalID(externalID, name string) (models.Organization, error) {
	var org models.Organization
	externalID = strings.TrimSpace(externalID)
	name = strings.TrimSpace(name)
	if externalID == "" {
		return org, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.required", "name", "external_id"), nil)
	}
	if utf8.RuneCountInString(externalID) > maxExternalIDLength {
		return org, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.maxLength", "max", "128"), nil)
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return org, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.maxLength", "max", "255"), nil)
	}
	if err := m.q.UpsertOrganizationByExternal.Get(&org, externalID, name); err != nil {
		m.lo.Error("error upserting organization", "external_id", externalID, "error", err)
		return org, m.generalError()
	}
	return org, nil
}

// Delete deletes an organization. Its contacts lose the membership; a contact that had it as default
// gets its earliest other organization. Conversations filed under it keep existing without an organization.
// It returns the contacts whose default organization changed.
func (m *Manager) Delete(id int) ([]models.DefaultChange, error) {
	var changed []models.DefaultChange
	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return nil, m.generalError()
	}
	defer tx.Rollback()
	if err := tx.Stmtx(m.q.ReassignDefaultOrganization).Select(&changed, id); err != nil {
		m.lo.Error("error reassigning default organization", "id", id, "error", err)
		return nil, m.generalError()
	}
	res, err := tx.Stmtx(m.q.DeleteOrganization).Exec(id)
	if err != nil {
		m.lo.Error("error deleting organization", "id", id, "error", err)
		return nil, m.generalError()
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundOrganization"), nil)
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing organization delete", "id", id, "error", err)
		return nil, m.generalError()
	}
	return changed, nil
}

// GetContacts returns the contacts of an organization, ordered by first name.
func (m *Manager) GetContacts(id, page, pageSize int) ([]models.Contact, error) {
	var contacts = make([]models.Contact, 0)
	if _, err := m.Get(id); err != nil {
		return contacts, err
	}
	if err := m.q.GetOrganizationContacts.Select(&contacts, id, pageSize, dbutil.PageOffset(page, pageSize)); err != nil {
		m.lo.Error("error fetching organization contacts", "id", id, "error", err)
		return contacts, m.generalError()
	}
	return contacts, nil
}

// GetContactOrganizations returns the organizations of a contact, the default one first.
func (m *Manager) GetContactOrganizations(contactID int) ([]models.ContactOrganization, error) {
	var orgs = make([]models.ContactOrganization, 0)
	if err := m.q.GetContactOrganizations.Select(&orgs, contactID); err != nil {
		m.lo.Error("error fetching contact organizations", "contact_id", contactID, "error", err)
		return orgs, m.generalError()
	}
	return orgs, nil
}

// AddContactOrganization adds the contact to the organization in one statement. With makeDefault, or when the
// contact has no default, the organization becomes its default. Conversations of the contact that have no
// organization get the default.
func (m *Manager) AddContactOrganization(contactID, orgID int, makeDefault bool) (models.MembershipChange, error) {
	var res struct {
		Found int `db:"found"`
		models.MembershipChange
	}
	if err := m.q.AddContactOrganization.Get(&res, contactID, orgID, makeDefault); err != nil {
		if dbutil.IsForeignKeyError(err) {
			return res.MembershipChange, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundOrganization"), nil)
		}
		m.lo.Error("error adding contact organization", "contact_id", contactID, "organization_id", orgID, "error", err)
		return res.MembershipChange, m.generalError()
	}
	if res.Found == 0 {
		return res.MembershipChange, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundUser"), nil)
	}
	return res.MembershipChange, nil
}

// RemoveContactOrganization removes the contact from the organization. When it was the default, the earliest
// other membership becomes the default. Conversations keep their organization.
func (m *Manager) RemoveContactOrganization(contactID, orgID int) (models.MembershipChange, error) {
	var res struct {
		Removed int `db:"removed"`
		models.MembershipChange
	}
	if err := m.q.RemoveContactOrganization.Get(&res, contactID, orgID); err != nil {
		m.lo.Error("error removing contact organization", "contact_id", contactID, "organization_id", orgID, "error", err)
		return res.MembershipChange, m.generalError()
	}
	if res.Removed == 0 {
		return res.MembershipChange, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundOrganization"), nil)
	}
	return res.MembershipChange, nil
}

// SetConversationOrganization files the conversation under one of its contact's organizations, or under none
// when orgID is not valid. An organization the contact does not belong to is rejected.
func (m *Manager) SetConversationOrganization(conversationUUID string, orgID null.Int) error {
	var id int
	if err := m.q.SetConversationOrganization.Get(&id, conversationUUID, orgID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return envelope.NewError(envelope.InputError, m.i18n.T("organization.notContactOrganization"), nil)
		}
		m.lo.Error("error setting conversation organization", "uuid", conversationUUID, "organization_id", orgID, "error", err)
		return m.generalError()
	}
	return nil
}

func (m *Manager) validate(name, externalID string) (string, null.String, error) {
	name = strings.TrimSpace(name)
	externalID = strings.TrimSpace(externalID)
	if name == "" {
		return "", null.String{}, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.required", "name", "{globals.terms.name}"), nil)
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return "", null.String{}, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.maxLength", "max", "255"), nil)
	}
	if utf8.RuneCountInString(externalID) > maxExternalIDLength {
		return "", null.String{}, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.maxLength", "max", "128"), nil)
	}
	return name, null.NewString(externalID, externalID != ""), nil
}

func (m *Manager) generalError() error {
	return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
}
