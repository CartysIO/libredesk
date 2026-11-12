# Libredesk agent MCP

Stdio MCP server for external agents: read the desk, search, manage conversations, send **customer replies**, and leave private notes. It calls `/api/v1` with one agent's API key — the same integration surface Grok should use over HTTP. See [external agent setup](../../docs/agents/external-agent.md).

## Tools

| Tool | Purpose |
|------|---------|
| `list_unassigned` | Triage queue |
| `list_assigned_to_me` | Bot's own assignee queue |
| `list_conversations` | All conversations (optional status filter) |
| `get_conversation`, `get_messages` | Thread detail |
| `search`, `search_contacts` | Find by text |
| `send_reply` | Customer-visible message (`messages:write`) |
| `add_note` | Internal note (`messages:write_private`) |
| `set_status`, `assign`, `unassign`, `set_priority`, `update_tags` | Conversation management |

## Secrets

Doppler project `libredesk`, config `stg`:

- `LIBREDESK_AGENT_API_KEY`
- `LIBREDESK_AGENT_API_SECRET`

Optional `LIBREDESK_AGENT_URL`. When it is unset the server uses staging, `https://libredesk-staging-6deb.up.railway.app`. For production, set the URL and start with `LIBREDESK_AGENT_DOPPLER_CONFIG=prd`.

Cursor starts it through `.cursor/mcp.json`, which runs `cursor.sh`. That script loads only those two secrets.

## Check

```
cd scripts/agent-mcp && npm test
```
