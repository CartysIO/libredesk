package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
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

// contactOrganizationReq is the body for setting a contact's organization. A null ID removes it.
type contactOrganizationReq struct {
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
}

// orgSyncResp counts what a sync did. Errors lists the items that failed; the other items are still synced.
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
// A contact gets the organization only when it has none, so links set by chat or by an agent stay.
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

// syncOrganizationContact creates or updates the contact like a widget JWT sign-in does, and links it to orgID when it has no organization.
// It returns true when it set the link.
func syncOrganizationContact(app *App, orgID int, c orgSyncContact) (bool, error) {
	extID := strings.TrimSpace(c.ExternalUserID)
	email := strings.ToLower(strings.TrimSpace(c.Email))
	if extID == "" || len(extID) > maxExternalUserIDLength {
		return false, fmt.Errorf("external_user_id is required and has at most %d characters", maxExternalUserIDLength)
	}
	if !stringutil.ValidEmail(email) {
		return false, fmt.Errorf("invalid email")
	}
	firstName := strings.TrimSpace(c.FirstName)
	if firstName == "" {
		firstName = strings.SplitN(email, "@", 2)[0]
	}
	user := umodels.User{
		FirstName:      firstName,
		LastName:       strings.TrimSpace(c.LastName),
		Email:          null.StringFrom(email),
		ExternalUserID: null.StringFrom(extID),
	}
	if err := app.user.ResolveContact(&user, umodels.ContactSync); err != nil {
		return false, err
	}
	contact, err := app.user.GetContactOrVisitor(user.ID, "")
	if err != nil {
		return false, err
	}
	if contact.OrganizationID.Valid {
		return false, nil
	}
	id := null.IntFrom(orgID)
	if err := app.organization.SetContactOrganization(user.ID, id); err != nil {
		return false, err
	}
	broadcastContactOrganization(app, user.ID, id)
	return true, nil
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
	if err := app.organization.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
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

// handleSetContactOrganization sets or removes a contact's organization.
func handleSetContactOrganization(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		req   contactOrganizationReq
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundUser"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	if err := app.organization.SetContactOrganization(id, req.OrganizationID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	broadcastContactOrganization(app, id, req.OrganizationID)
	c, err := app.user.GetContactOrVisitor(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(c)
}

// broadcastContactOrganization tells open conversations of the contact about its new organization.
func broadcastContactOrganization(app *App, contactID int, orgID null.Int) {
	update := map[string]any{"organization_id": nil, "organization_name": nil, "organization_external_id": nil}
	if orgID.Valid {
		org, err := app.organization.Get(orgID.Int)
		if err != nil {
			return
		}
		update = map[string]any{"organization_id": org.ID, "organization_name": org.Name, "organization_external_id": org.ExternalID}
	}
	app.conversation.BroadcastContactUpdate(contactID, update)
}

// linkContactOrganization creates or renames the organization with this external ID and makes it the contact's organization.
func linkContactOrganization(app *App, contactID int, externalID, name string) error {
	org, err := app.organization.UpsertByExternalID(externalID, name)
	if err != nil {
		return err
	}
	contact, err := app.user.GetContactOrVisitor(contactID, "")
	if err != nil {
		return err
	}
	if contact.OrganizationID.Valid && contact.OrganizationID.Int == org.ID {
		return nil
	}
	orgID := null.IntFrom(org.ID)
	if err := app.organization.SetContactOrganization(contactID, orgID); err != nil {
		return err
	}
	broadcastContactOrganization(app, contactID, orgID)
	return nil
}

func organizationIDParam(r *fastglue.Request) (int, bool) {
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	return id, err == nil && id > 0
}
