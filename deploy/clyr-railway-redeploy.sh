#!/usr/bin/env bash
# Redeploy Libredesk in Railway after GHCR publishes ghcr.io/cartysio/libredesk:clyr.
# Requires: railway CLI, logged in to the Clyr workspace.
set -euo pipefail

PROJECT_ID="86f4a5dc-54b6-43b3-ad04-f88f0fd77916"
SERVICE="Libredesk"

redeploy_env() {
  local env="$1"
  echo "==> Redeploying ${SERVICE} (${env}) from ghcr.io/cartysio/libredesk:clyr"
  railway link -p "${PROJECT_ID}" -e "${env}" -s "${SERVICE}" >/dev/null
  railway redeploy --from-source -y
}

redeploy_env staging
redeploy_env production

echo "Done. Staging: https://libredesk-staging-6deb.up.railway.app"
echo "      Production: https://clyr-support.up.railway.app"
