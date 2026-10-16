package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
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
