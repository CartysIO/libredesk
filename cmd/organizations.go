package main

import (
	"fmt"
	"strconv"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	omodels "github.com/abhinavxd/libredesk/internal/organization/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

// organizationReq is the body for creating or updating an organization.
type organizationReq struct {
	Name       string `json:"name"`
	ExternalID string `json:"external_id"`
}

// contactOrganizationReq is the body for adding a contact to an organization or making it the default.
type contactOrganizationReq struct {
	OrganizationID int `json:"organization_id"`
}

// conversationOrganizationReq is the body for filing a conversation under an organization. A null ID removes it.
type conversationOrganizationReq struct {
	OrganizationID null.Int `json:"organization_id"`
}

const (
	maxSyncOrganizations = 100
	maxSyncContacts      = 1000
)

// orgSyncReq is the body for syncing organizations and their contacts from an external system.
type orgSyncReq struct {
	Organizations []orgSyncOrganization `json:"organizations"`
}

type orgSyncOrganization struct {
	ExternalID string           `json:"external_id"`
	Name       string           `json:"name"`
	Contacts   []orgSyncContact `json:"contacts"`
}

type orgSyncContact struct {
	ExternalUserID string `json:"external_user_id"`
	Email          string `json:"email"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	// Default makes this organization the contact's default, which new email conversations get.
	Default bool `json:"default"`
}

// orgSyncResp counts what a sync did. Linked counts new memberships and Kept existing ones. Errors lists the items that failed; the other items are still synced.
type orgSyncResp struct {
	Organizations int            `json:"organizations"`
	Linked        int            `json:"linked"`
	Kept          int            `json:"kept"`
	Errors        []orgSyncError `json:"errors"`
}

type orgSyncError struct {
	ExternalID     string `json:"external_id"`
	ExternalUserID string `json:"external_user_id,omitempty"`
	Error          string `json:"error"`
}

// handleSyncOrganizations creates or renames organizations by external ID and creates or updates their contacts by external user ID.
// Each contact is added to the organization. It never removes a membership, so links set by chat or by an agent stay.
func handleSyncOrganizations(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req orgSyncReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	contacts := 0
	for _, o := range req.Organizations {
		contacts += len(o.Contacts)
	}
	if len(req.Organizations) > maxSyncOrganizations || contacts > maxSyncContacts {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, fmt.Sprintf("at most %d organizations and %d contacts per request", maxSyncOrganizations, maxSyncContacts), nil, envelope.InputError)
	}

	resp := orgSyncResp{Errors: []orgSyncError{}}
	for _, o := range req.Organizations {
		org, err := app.organization.UpsertByExternalID(o.ExternalID, o.Name)
		if err != nil {
			resp.Errors = append(resp.Errors, orgSyncError{ExternalID: o.ExternalID, Error: err.Error()})
			continue
		}
		resp.Organizations++
		for _, c := range o.Contacts {
			linked, err := syncOrganizationContact(app, org.ID, c)
			if err != nil {
				resp.Errors = append(resp.Errors, orgSyncError{ExternalID: o.ExternalID, ExternalUserID: c.ExternalUserID, Error: err.Error()})
				continue
			}
			if linked {
				resp.Linked++
			} else {
				resp.Kept++
			}
		}
	}
	return r.SendEnvelope(resp)
}

// syncOrganizationContact creates or updates the contact like a widget JWT sign-in does, and adds it to orgID.
// It returns true when the membership is new. Its errors are envelope errors, safe to return to the caller.
func syncOrganizationContact(app *App, orgID int, c orgSyncContact) (bool, error) {
	extID := strings.TrimSpace(c.ExternalUserID)
	email := strings.ToLower(strings.TrimSpace(c.Email))
	firstName := strings.TrimSpace(c.FirstName)
	lastName := strings.TrimSpace(c.LastName)
	if extID == "" || len(extID) > maxExternalUserIDLength {
		return false, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.required", "name", "external_user_id"), nil)
	}
	if len(email) > maxEmailLength || !stringutil.ValidEmail(email) {
		return false, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidEmail"), nil)
	}
	if firstName == "" {
		firstName = strings.SplitN(email, "@", 2)[0]
	}
	if len(firstName) > maxNameLength || len(lastName) > maxNameLength {
		return false, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.maxLength", "max", strconv.Itoa(maxNameLength)), nil)
	}
	user := umodels.User{
		FirstName:      firstName,
		LastName:       lastName,
		Email:          null.StringFrom(email),
		ExternalUserID: null.StringFrom(extID),
	}
	if err := app.user.ResolveContact(&user, umodels.ContactSync); err != nil {
		app.lo.Error("error resolving contact in organization sync", "external_user_id", extID, "error", err)
		return false, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	change, err := app.organization.AddContactOrganization(user.ID, orgID, c.Default)
	if err != nil {
		return false, err
	}
	if change.Added > 0 || change.DefaultChanged > 0 {
		broadcastContactOrganizations(app, user.ID, change.Filed)
	}
	return change.Added > 0, nil
}

// handleGetOrganizations returns organizations, filtered by the `q` search text.
func handleGetOrganizations(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		query = string(r.RequestCtx.QueryArgs().Peek("q"))
	)
	page, pageSize := getPagination(r)
	orgs, total, err := app.organization.GetAll(query, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(envelope.PageResults{
		Results:    orgs,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetOrganization returns one organization.
func handleGetOrganization(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, ok := organizationIDParam(r)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundOrganization"), nil, envelope.InputError)
	}
	org, err := app.organization.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(org)
}

// handleCreateOrganization creates an organization.
func handleCreateOrganization(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req organizationReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	org, err := app.organization.Create(req.Name, req.ExternalID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(org)
}

// handleUpdateOrganization updates an organization.
func handleUpdateOrganization(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req organizationReq
	)
	id, ok := organizationIDParam(r)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundOrganization"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	org, err := app.organization.Update(id, req.Name, req.ExternalID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(org)
}

// handleDeleteOrganization deletes an organization.
func handleDeleteOrganization(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, ok := organizationIDParam(r)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundOrganization"), nil, envelope.InputError)
	}
	changed, err := app.organization.Delete(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	for _, c := range changed {
		broadcastContactOrganizations(app, c.ContactID, nil)
	}
	return r.SendEnvelope(true)
}

// handleGetOrganizationContacts returns the contacts of an organization.
func handleGetOrganizationContacts(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, ok := organizationIDParam(r)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundOrganization"), nil, envelope.InputError)
	}
	page, pageSize := getPagination(r)
	contacts, err := app.organization.GetContacts(id, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(contacts)
}

// handleGetContactOrganizations returns the organizations of a contact, the default one first.
func handleGetContactOrganizations(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, ok := intParam(r, "id")
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundUser"), nil, envelope.InputError)
	}
	orgs, err := app.organization.GetContactOrganizations(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(orgs)
}

// handleAddContactOrganization adds a contact to an organization. It becomes the default when the contact has none.
func handleAddContactOrganization(r *fastglue.Request) error {
	return updateContactOrganization(r, false)
}

// handleSetContactOrganization makes an organization the contact's default, and adds the contact to it if needed.
func handleSetContactOrganization(r *fastglue.Request) error {
	return updateContactOrganization(r, true)
}

func updateContactOrganization(r *fastglue.Request, makeDefault bool) error {
	var (
		app = r.Context.(*App)
		req contactOrganizationReq
	)
	id, ok := intParam(r, "id")
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundUser"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	if req.OrganizationID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "organization_id"), nil, envelope.InputError)
	}
	change, err := app.organization.AddContactOrganization(id, req.OrganizationID, makeDefault)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return sendContactOrganizations(r, app, id, change.Filed)
}

// handleRemoveContactOrganization removes a contact from an organization.
func handleRemoveContactOrganization(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, ok := intParam(r, "id")
	orgID, orgOK := intParam(r, "org_id")
	if !ok || !orgOK {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundOrganization"), nil, envelope.InputError)
	}
	if _, err := app.organization.RemoveContactOrganization(id, orgID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return sendContactOrganizations(r, app, id, nil)
}

// sendContactOrganizations tells open conversations about the change and returns the contact's organizations.
func sendContactOrganizations(r *fastglue.Request, app *App, contactID int, filed []string) error {
	orgs := broadcastContactOrganizations(app, contactID, filed)
	if orgs == nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	return r.SendEnvelope(orgs)
}

// broadcastContactOrganizations tells open conversations of the contact about its organizations and its default
// organization. The filed conversations just got the default organization and are told so too.
// It returns the organizations, or nil when they could not be read.
func broadcastContactOrganizations(app *App, contactID int, filed []string) []omodels.ContactOrganization {
	orgs, err := app.organization.GetContactOrganizations(contactID)
	if err != nil {
		return nil
	}
	update := map[string]any{"organizations": orgs, "organization_id": nil, "organization_name": nil, "organization_external_id": nil}
	for _, o := range orgs {
		if o.IsDefault {
			update["organization_id"] = o.ID
			update["organization_name"] = o.Name
			update["organization_external_id"] = o.ExternalID
		}
	}
	app.conversation.BroadcastContactUpdate(contactID, update)
	for _, uuid := range filed {
		app.conversation.BroadcastConversationUpdate(uuid, map[string]any{
			"organization_id":          update["organization_id"],
			"organization_name":        update["organization_name"],
			"organization_external_id": update["organization_external_id"],
		})
	}
	return orgs
}

// handleSetConversationOrganization files a conversation under one of its contact's organizations, or under none.
func handleSetConversationOrganization(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   conversationOrganizationReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if _, err := enforceConversationAccess(app, uuid, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.organization.SetConversationOrganization(uuid, req.OrganizationID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	update := map[string]any{"organization_id": nil, "organization_name": nil, "organization_external_id": nil}
	if req.OrganizationID.Valid {
		if org, err := app.organization.Get(req.OrganizationID.Int); err == nil {
			update = map[string]any{"organization_id": org.ID, "organization_name": org.Name, "organization_external_id": org.ExternalID}
		}
	}
	app.conversation.BroadcastConversationUpdate(uuid, update)
	return r.SendEnvelope(update)
}

// linkContactOrganization creates or renames the organization with this external ID and adds the contact to it.
// It returns the organization ID. The contact's default organization changes only when it has none.
func linkContactOrganization(app *App, contactID int, externalID, name string) (int, error) {
	org, err := app.organization.UpsertByExternalID(externalID, name)
	if err != nil {
		return 0, err
	}
	change, err := app.organization.AddContactOrganization(contactID, org.ID, false)
	if err != nil {
		return 0, err
	}
	if change.Added > 0 || change.DefaultChanged > 0 {
		broadcastContactOrganizations(app, contactID, change.Filed)
	}
	return org.ID, nil
}

func intParam(r *fastglue.Request, name string) (int, bool) {
	v, _ := r.RequestCtx.UserValue(name).(string)
	id, err := strconv.Atoi(v)
	return id, err == nil && id > 0
}

func organizationIDParam(r *fastglue.Request) (int, bool) {
	return intParam(r, "id")
}
