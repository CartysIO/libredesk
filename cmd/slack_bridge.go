package main

import (
	"log"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/media"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/slackbridge"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/abhinavxd/libredesk/internal/webhook"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/jmoiron/sqlx"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// maxSlackEventBody caps the Slack Events API request body.
const maxSlackEventBody = 1 << 20

// initSlackBridge returns the Slack bridge, or nil when the [slack] keys are not set.
func initSlackBridge(db *sqlx.DB, users *user.Manager, mediaStore *media.Manager, settings *setting.Manager) *slackbridge.Manager {
	cfg := slackbridge.Config{
		BotToken:      ko.String("slack.bot_token"),
		SigningSecret: ko.String("slack.signing_secret"),
		ChannelID:     ko.String("slack.channel_id"),
		AIBotID:       ko.String("slack.ai_bot_id"),
		AIAgentEmail:  ko.String("slack.ai_agent_email"),
		APIURL:        ko.String("slack.api_url"),
		QueueSize:     ko.Int("slack.queue_size"),
	}
	if !cfg.Enabled() {
		return nil
	}
	m, err := slackbridge.New(cfg, db, initLogger("slack_bridge"), users, mediaStore, settings)
	if err != nil {
		log.Fatalf("error initializing slack bridge: %v", err)
	}
	return m
}

// eventFanout sends conversation events to the webhooks and to the Slack bridge.
type eventFanout struct {
	webhook *webhook.Manager
	bridge  *slackbridge.Manager
}

func (f eventFanout) TriggerEvent(event wmodels.WebhookEvent, data any) {
	f.webhook.TriggerEvent(event, data)
	if f.bridge != nil {
		f.bridge.TriggerEvent(event, data)
	}
}

func (f eventFanout) TriggerWebhook(webhookID int, event wmodels.WebhookEvent, data any) {
	f.webhook.TriggerWebhook(webhookID, event, data)
}

// handleSlackEvents receives Slack Events API requests. It checks the signature, answers the
// url_verification challenge, and queues events; Slack needs an answer within 3 seconds.
func handleSlackEvents(r *fastglue.Request) error {
	app := r.Context.(*App)
	if app.slackBridge == nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "slack bridge is not configured", nil, envelope.NotFoundError)
	}
	body := r.RequestCtx.PostBody()
	if len(body) > maxSlackEventBody {
		return r.SendErrorEnvelope(fasthttp.StatusRequestEntityTooLarge, "request too large", nil, envelope.InputError)
	}
	body = append([]byte(nil), body...)
	ts := string(r.RequestCtx.Request.Header.Peek("X-Slack-Request-Timestamp"))
	sig := string(r.RequestCtx.Request.Header.Peek("X-Slack-Signature"))
	if !slackbridge.VerifySignature(app.slackBridge.SigningSecret(), ts, body, sig, time.Now()) {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "invalid signature", nil, envelope.PermissionError)
	}
	challenge, err := app.slackBridge.HandleSlackEvent(body)
	if err != nil {
		app.lo.Warn("bad slack event", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "bad event", nil, envelope.InputError)
	}
	r.RequestCtx.SetContentType("text/plain")
	r.RequestCtx.SetStatusCode(fasthttp.StatusOK)
	r.RequestCtx.SetBodyString(challenge)
	return nil
}
