# Libredesk agent MCP

Stdio MCP server for external agents: read the desk, search, manage conversations, send **customer replies**, and leave private notes. It calls `/api/v1` with one agent's API key — the same integration surface Grok should use over HTTP. See [external agent setup](../../docs/agents/external-agent.md).

## Tools

MCP tool names match the [external agent REST guide](../../docs/agents/external-agent.md#endpoints-used-by-scriptsagent-mcp) (methods, paths, and bodies live there only).

## Secrets

Doppler project `libredesk`, config `stg`:

- `LIBREDESK_AGENT_API_KEY`
- `LIBREDESK_AGENT_API_SECRET`

Optional `LIBREDESK_AGENT_URL`. When it is unset the server uses staging, `https://libredesk-staging-6deb.up.railway.app`. Use staging keys in Doppler by default; only point at production when you accept the risk of customer-visible `send_reply`.

Cursor starts it through `.cursor/mcp.json`, which runs `cursor.sh`. That script loads only those two secrets.

## Check

```
cd scripts/agent-mcp && npm test
```
