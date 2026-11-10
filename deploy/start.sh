#!/bin/sh
# Installs the schema on first boot, applies upgrades, then serves.
# All settings come from LIBREDESK_* environment variables (e.g. LIBREDESK_DB__HOST).
set -e

export LIBREDESK_APP__SERVER__ADDRESS="${LIBREDESK_APP__SERVER__ADDRESS:-0.0.0.0:${PORT:-9000}}"

./libredesk --install --idempotent-install --yes --config config.toml
./libredesk --upgrade --yes --config config.toml
exec ./libredesk --config config.toml
