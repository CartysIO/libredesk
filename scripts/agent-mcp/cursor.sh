#!/bin/sh
# Starts the Libredesk MCP for Cursor. Reads the agent API key from Doppler
# project libredesk, config stg (override with LIBREDESK_AGENT_DOPPLER_CONFIG).
# Stdout is the MCP protocol, so Doppler is silenced.
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

NODE=""
if command -v node >/dev/null 2>&1; then
  NODE=$(command -v node)
else
  for candidate in "$HOME/.nvm/versions/node/v22.14.0/bin/node" /opt/homebrew/bin/node /usr/local/bin/node; do
    if [ -x "$candidate" ]; then
      NODE=$candidate
      break
    fi
  done
fi
if [ -z "$NODE" ]; then
  echo "libredesk agent MCP: node was not found" >&2
  exit 1
fi

DOPPLER=""
if command -v doppler >/dev/null 2>&1; then
  DOPPLER=$(command -v doppler)
elif [ -x /usr/local/bin/doppler ]; then
  DOPPLER=/usr/local/bin/doppler
elif [ -x /opt/homebrew/bin/doppler ]; then
  DOPPLER=/opt/homebrew/bin/doppler
fi
if [ -z "$DOPPLER" ]; then
  echo "libredesk agent MCP: doppler was not found" >&2
  exit 1
fi

CONFIG="${LIBREDESK_AGENT_DOPPLER_CONFIG:-stg}"
exec "$DOPPLER" run --silent --project libredesk --config "$CONFIG" \
  --only-secrets LIBREDESK_AGENT_API_KEY,LIBREDESK_AGENT_API_SECRET \
  -- "$NODE" "$ROOT/scripts/agent-mcp/server.js"
