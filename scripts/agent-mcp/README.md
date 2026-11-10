# Libredesk agent MCP

Stdio MCP server that lets an agent read the desk and leave private notes. It calls the existing `/api/v1` API with one agent's API key. It cannot send a customer-visible reply.

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
