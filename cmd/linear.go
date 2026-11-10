package main

import (
	"errors"
	"log"
	"strconv"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/linear"
	"github.com/jmoiron/sqlx"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// maxLinearWebhookBody caps the Linear webhook request body.
const maxLinearWebhookBody = 1 << 20

// initLinear returns the Linear integration, or nil when the [linear] keys are not set.
func initLinear(db *sqlx.DB, convs *conversation.Manager, systemUserID int) *linear.Manager {
	cfg := linear.Config{
		APIKey:        ko.String("linear.api_key"),
		WebhookSecret: ko.String("linear.webhook_secret"),
		APIURL:        ko.String("linear.api_url"),
	}
	if !cfg.Enabled() {
		return nil
	}
	m, err := linear.New(cfg, db, initLogger("linear"), convs, systemUserID)
	if err != nil {
		log.Fatalf("error initializing linear: %v", err)
	}
	return m
}

type linearLinkReq struct {
	// Ref is an issue URL or identifier, for example ENG-123.
	Ref string `json:"ref"`
}

// linearConversationID checks the agent may see the conversation and returns its ID.
func linearConversationID(r *fastglue.Request) (int, error) {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return 0, err
	}
	conv, err := enforceConversationAccess(app, r.RequestCtx.UserValue("uuid").(string), user)
	if err != nil {
		return 0, err
	}
	return conv.ID, nil
}

// handleGetLinearIssues returns the Linear issues linked to a conversation, and whether Linear is set up.
func handleGetLinearIssues(r *fastglue.Request) error {
	app := r.Context.(*App)
	convID, err := linearConversationID(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if app.linear == nil {
		return r.SendEnvelope(map[string]any{"enabled": false, "issues": []linear.Link{}})
	}
	links, err := app.linear.Links(convID)
	if err != nil {
		app.lo.Error("error fetching linear issues", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}
	return r.SendEnvelope(map[string]any{"enabled": true, "issues": links})
}

// handleLinkLinearIssue links a Linear issue, given by URL or identifier, to a conversation.
func handleLinkLinearIssue(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	if app.linear == nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("linear.notConfigured"), nil, envelope.NotFoundError)
	}
	var req linearLinkReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	convID, err := linearConversationID(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	link, err := app.linear.Link(r.RequestCtx, convID, req.Ref, auser.ID)
	switch {
	case errors.Is(err, linear.ErrInvalidRef):
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("linear.invalidRef"), nil, envelope.InputError)
	case errors.Is(err, linear.ErrIssueNotFound):
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("linear.issueNotFound"), nil, envelope.NotFoundError)
	case err != nil:
		app.lo.Error("error linking linear issue", "ref", req.Ref, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, app.i18n.T("linear.lookupFailed"), nil, envelope.GeneralError)
	}
	return r.SendEnvelope(link)
}

// handleUnlinkLinearIssue removes a linked Linear issue from a conversation.
func handleUnlinkLinearIssue(r *fastglue.Request) error {
	app := r.Context.(*App)
	if app.linear == nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("linear.notConfigured"), nil, envelope.NotFoundError)
	}
	linkID, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if linkID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	convID, err := linearConversationID(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.linear.Unlink(convID, linkID); err != nil {
		app.lo.Error("error unlinking linear issue", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}
	return r.SendEnvelope(true)
}

// handleLinearWebhook receives Linear's Comment and Issue webhooks. Linear signs each request.
func handleLinearWebhook(r *fastglue.Request) error {
	app := r.Context.(*App)
	if app.linear == nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "linear is not configured", nil, envelope.NotFoundError)
	}
	body := r.RequestCtx.PostBody()
	if len(body) > maxLinearWebhookBody {
		return r.SendErrorEnvelope(fasthttp.StatusRequestEntityTooLarge, "request too large", nil, envelope.InputError)
	}
	sig := string(r.RequestCtx.Request.Header.Peek("Linear-Signature"))
	err := app.linear.HandleWebhook(body, sig, time.Now())
	switch {
	case errors.Is(err, linear.ErrBadSignature):
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "invalid signature", nil, envelope.PermissionError)
	case errors.Is(err, linear.ErrBadPayload):
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "bad payload", nil, envelope.InputError)
	case err != nil:
		// A non-2xx answer makes Linear send the webhook again.
		app.lo.Error("error handling linear webhook", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "error", nil, envelope.GeneralError)
	}
	return r.SendEnvelope(true)
}
