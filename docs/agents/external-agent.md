# External support agents (Grok, scripts)

LibreDesk exposes a **REST API** at `/api/v1`. Remote agents (for example xAI Grok) integrate with HTTP and an **agent API key**. The repo also ships an **MCP server** (`scripts/agent-mcp/`) that calls the same endpoints — useful for Cursor, not required for Grok.

## Setup in Admin

1. **Agents →** create a dedicated agent (for example `Clyr Support Bot`) with a custom role.
2. Grant at least:
   - `conversations:read`, `conversations:read_all`, `conversations:read_unassigned`, `conversations:read_assigned`
   - `conversations:update_status`, `conversations:update_user_assignee`, `conversations:update_priority`, `conversations:update_tags`
   - `messages:read`, `messages:write`, `messages:write_private`
3. On that agent, **Generate API key**. Copy key and secret; the secret is shown once.
4. Set the instance URL (production: `https://clyr-support.up.railway.app` or your `app.root_url`).

Store key and secret in the external agent's secret store (not in Slack).

## Authentication

Every request:

```http
Authorization: token <api_key>:<api_secret>
Accept: application/json
Content-Type: application/json
```

Responses use `{ "status": "success", "data": ... }` or `{ "status": "error", "message": "..." }`.

## Suggested agent workflow

```text
triage
  GET /api/v1/conversations/unassigned
  or GET /api/v1/conversations/assigned (bot's queue)
investigate
  GET /api/v1/conversations/{uuid}
  GET /api/v1/conversations/{uuid}/messages
  GET /api/v1/search/conversations?query=...
act
  POST .../messages  (reply or note)
  PUT .../status, .../assignee/user, .../priority
  POST .../tags
```

## Endpoints used by `scripts/agent-mcp`

| Tool (MCP) | Method | Path | Body (JSON) |
|------------|--------|------|-------------|
| `list_unassigned` | GET | `/api/v1/conversations/unassigned` | — |
| `list_assigned_to_me` | GET | `/api/v1/conversations/assigned` | — |
| `list_conversations` | GET | `/api/v1/conversations/all` | query `filters` for status |
| `get_conversation` | GET | `/api/v1/conversations/{uuid}` | — |
| `get_messages` | GET | `/api/v1/conversations/{uuid}/messages` | — |
| `search` | GET | `/api/v1/search/conversations`, `/api/v1/search/messages` | query `query` |
| `search_contacts` | GET | `/api/v1/contacts/search` | query `query` |
| `send_reply` | POST | `/api/v1/conversations/{uuid}/messages` | `{ "message": "<p>…</p>", "private": false, "sender_type": "agent", "keep_status": false }` |
| `add_note` | POST | same | `{ "private": true, ... }` |
| `set_status` | PUT | `/api/v1/conversations/{uuid}/status` | `{ "status": "Resolved", "snoozed_until": "4h" }` (duration when Snoozed) |
| `assign` | PUT | `/api/v1/conversations/{uuid}/assignee/user` | `{ "assignee_id": 123 }` |
| `unassign` | PUT | `/api/v1/conversations/{uuid}/assignee/user/remove` | `{}` |
| `set_priority` | PUT | `/api/v1/conversations/{uuid}/priority` | `{ "priority": "High" }` |
| `update_tags` | POST | `/api/v1/conversations/{uuid}/tags` | `{ "tags": ["escalated"], "action": "add_tags" }` |

`message` is HTML. The MCP server wraps plain text in `<p>` with basic escaping.

### Status values

`Open`, `On you`, `On customer`, `Snoozed`, `Resolved`, `Closed`.

### Tag actions

`add_tags`, `remove_tags`, `set_tags` (default `add_tags`).

## Cutover notes

- Decide whether Grok checks LibreDesk only or Pylon for legacy threads.
- Customer replies should use `send_reply` / `private: false` only when the bot should email or message the contact.
- Test with `GET` on a few conversations before automating replies on production mail.

## MCP (optional)

For Cursor, enable `libredesk` in `.cursor/mcp.json` and Doppler secrets `LIBREDESK_AGENT_API_KEY`, `LIBREDESK_AGENT_API_SECRET`. See `scripts/agent-mcp/README.md`.
