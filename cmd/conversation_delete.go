package main

import (
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleDeleteConversation permanently deletes a conversation with its messages and notes.
func handleDeleteConversation(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conversation, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.lo.Info("deleting conversation", "uuid", uuid, "actor_id", auser.ID)
	if err := app.conversation.DeleteConversation(uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	logConversationsDeleted(r, []cmodels.DeletedConversation{{ID: conversation.ID, UUID: uuid, ReferenceNumber: conversation.ReferenceNumber}}, conversation.Contact.Email.String, false)
	return r.SendEnvelope(true)
}

// handleMarkConversationSpam blocks the conversation's contact, so new messages from them are dropped,
// and deletes the conversation with the contact's other unresolved conversations.
func handleMarkConversationSpam(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	canBlock, err := app.authz.Enforce(user, "contacts", "block")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !canBlock {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
	}
	conversation, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.lo.Info("marking conversation as spam", "uuid", uuid, "contact_id", conversation.ContactID, "actor_id", auser.ID)
	if err := app.user.ToggleEnabled(conversation.ContactID, conversation.Contact.Type, false); err != nil {
		return sendErrorEnvelope(r, err)
	}
	deleted, err := app.conversation.DeleteSpamConversations(uuid, conversation.ContactID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	logConversationsDeleted(r, deleted, conversation.Contact.Email.String, true)

	uuids := make([]string, 0, len(deleted))
	for _, d := range deleted {
		uuids = append(uuids, d.UUID)
	}
	return r.SendEnvelope(map[string]any{"deleted_uuids": uuids})
}

// logConversationsDeleted writes one activity log entry per deleted conversation.
func logConversationsDeleted(r *fastglue.Request, deleted []cmodels.DeletedConversation, contactEmail string, spam bool) {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		ip    = realip.FromRequest(r.RequestCtx)
	)
	for _, d := range deleted {
		if err := app.activityLog.ConversationDeleted(auser.ID, auser.Email, ip, d.ID, d.ReferenceNumber, contactEmail, spam); err != nil {
			app.lo.Error("error creating conversation deleted activity log", "uuid", d.UUID, "error", err)
		}
	}
}
