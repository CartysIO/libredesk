package main

import (
	"errors"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/clerk"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type clerkLoginRequest struct {
	Token string `json:"token"`
}

// handleClerkLogin signs an agent in with a Clerk session token from the login page.
func handleClerkLogin(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		ip  = realip.FromRequest(r.RequestCtx)
		req clerkLoginRequest
	)
	if app.clerk == nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("globals.messages.badRequest"), nil, envelope.NotFoundError)
	}
	if err := r.Decode(&req, "json"); err != nil || req.Token == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	rootURL, err := app.setting.GetAppRootURL()
	if err != nil {
		app.lo.Error("error getting app root url for clerk login", "error", err)
		return clerkLoginFailed(r)
	}

	clerkUserID, err := app.clerk.VerifySession(r.RequestCtx, req.Token, rootURL)
	if err != nil {
		app.lo.Warn("clerk session token rejected", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("auth.oidcLoginFailed"), nil, envelope.PermissionError)
	}

	email, err := app.clerk.PrimaryEmail(r.RequestCtx, clerkUserID)
	if err != nil {
		if errors.Is(err, clerk.ErrNoVerifiedMail) {
			app.lo.Warn("clerk user has no verified primary email", "clerk_user_id", clerkUserID)
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("auth.oidcNoAccount"), nil, envelope.PermissionError)
		}
		app.lo.Error("error fetching clerk user", "clerk_user_id", clerkUserID, "error", err)
		return clerkLoginFailed(r)
	}

	user, err := app.user.GetAgent(0, email)
	if err != nil {
		if e, ok := err.(envelope.Error); ok && e.ErrorType == envelope.NotFoundError {
			app.lo.Warn("no agent account matching clerk email", "clerk_user_id", clerkUserID, "email", email)
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("auth.oidcNoAccount"), nil, envelope.PermissionError)
		}
		return clerkLoginFailed(r)
	}
	// Only agents can log in; GetAgent also resolves ai_assistant identity users.
	if user.Type != models.UserTypeAgent {
		app.lo.Warn("clerk login rejected for non-agent user", "user_id", user.ID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("auth.oidcNoAccount"), nil, envelope.PermissionError)
	}
	if !user.Enabled {
		app.lo.Warn("clerk login rejected for disabled account", "user_id", user.ID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("user.accountDisabled"), nil, envelope.PermissionError)
	}

	if err := app.auth.SaveSession(amodels.User{
		ID:        user.ID,
		Email:     user.Email.String,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, r); err != nil {
		app.lo.Error("error saving session for clerk login", "user_id", user.ID, "error", err)
		return clerkLoginFailed(r)
	}
	if err := app.auth.SetCSRFCookie(r); err != nil {
		app.lo.Error("error setting csrf cookie", "error", err)
		return clerkLoginFailed(r)
	}

	if err := app.user.UpdateLastLoginAt(user.ID); err != nil {
		app.lo.Error("error updating last login at for clerk login", "user_id", user.ID, "error", err)
	}
	app.user.InvalidateAgentCache(user.ID)

	if err := app.activityLog.Login(user.ID, user.Email.String, ip); err != nil {
		app.lo.Error("error creating login activity log", "error", err)
	}

	app.lo.Info("clerk login successful", "user_id", user.ID, "email", user.Email.String)
	return r.SendEnvelope(user)
}

func clerkLoginFailed(r *fastglue.Request) error {
	app := r.Context.(*App)
	return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("auth.oidcLoginFailed"), nil, envelope.GeneralError)
}
