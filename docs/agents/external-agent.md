# External support agents (Grok, scripts)

LibreDesk exposes a **REST API** at `/api/v1`. Remote agents (for example xAI Grok) integrate with HTTP and an **agent API key**. The repo also ships an **MCP server** (`scripts/agent-mcp/`) that calls the same endpoints — useful for Cursor, not required for Grok.

**Canonical REST mapping:** the table in [Endpoints](#endpoints-used-by-scriptsagent-mcp) is the single source of truth. `scripts/agent-mcp/README.md` lists MCP tool names only.

## Setup in Admin

1. **Agents →** create a dedicated agent (for example `Clyr Support Bot`) with a **custom role** (not Admin).
2. Grant permissions the bot actually needs:

   **Typical autonomous bot (assigned queue only)**

   - `conversations:read`, `conversations:read_assigned`
   - `conversations:update_status`, `conversations:update_user_assignee`, `conversations:update_priority`, `conversations:update_tags`
   - `messages:read`, `messages:write`, `messages:write_private`

   **Add when the bot triages the global unassigned inbox**

   - `conversations:read_unassigned`

   **Add only when the bot must see every conversation (broader blast radius if the key leaks)**

   - `conversations:read_all`

3. On that agent, **Generate API key**. Copy key and secret; the secret is shown once.
4. Set the instance URL (production: `https://clyr-support.up.railway.app` or your `app.root_url`).

Store key and secret in the external agent's secret store (not in Slack). Revoke and reissue if a key is exposed.

## Security

- **Least privilege:** omit `conversations:read_all` unless the bot needs the full desk. A stolen key then cannot list every thread.
- **Separate keys:** use a **staging** API key in Cursor/Doppler for MCP; keep the **production** key only in Grok (or your bot runtime).
- **No admin:** do not grant `users:manage`, settings write, or conversation delete unless you intend a destructive bot.
- **Customer mail:** `POST …/messages` with `private: false` sends real outbound mail on email inboxes. Smoke-test on staging first.
- **Replies and status:** with `keep_status: false` (the API default), a public reply can move the conversation to **On customer**. Bots should usually send `keep_status: true` unless that transition is intentional.
- **Permission errors:** missing grants return `403` with `Permission denied` in the JSON envelope — fix the role, not the client.

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
| `send_reply` | POST | `/api/v1/conversations/{uuid}/messages` | `{ "message": "<p>…</p>", "private": false, "sender_type": "agent", "keep_status": true }` (recommended for bots) |
| `add_note` | POST | same | `{ "private": true, ... }` |
| `set_status` | PUT | `/api/v1/conversations/{uuid}/status` | `{ "status": "Resolved", "snoozed_until": "4h" }` (duration when Snoozed) |
| `assign` | PUT | `/api/v1/conversations/{uuid}/assignee/user` | `{ "assignee_id": 123 }` |
| `unassign` | PUT | `/api/v1/conversations/{uuid}/assignee/user/remove` | `{}` |
| `set_priority` | PUT | `/api/v1/conversations/{uuid}/priority` | `{ "priority": "<label from Admin → Priorities>" }` |
| `update_tags` | POST | `/api/v1/conversations/{uuid}/tags` | `{ "tags": ["escalated"], "action": "add_tags" }` |

The MCP server wraps reply and note text as HTML in `<p>` with basic escaping. Grok may send HTML directly when calling REST.

### Status values

`Open`, `On you`, `On customer`, `Snoozed`, `Resolved`, `Closed`.

### Tag actions

`add_tags`, `remove_tags`, `set_tags` (default `add_tags`).

## Cutover notes

- Decide whether Grok checks LibreDesk only or Pylon for legacy threads.
- Test with `GET` on a few conversations before automating replies on production mail.

## MCP (optional)

For Cursor, enable `libredesk` in `.cursor/mcp.json` and Doppler secrets `LIBREDESK_AGENT_API_KEY`, `LIBREDESK_AGENT_API_SECRET`. Prefer **staging** keys in Doppler. See `scripts/agent-mcp/README.md`.
