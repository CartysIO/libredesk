# Clyr: GHCR image → Railway

Pushing to `clyr` builds and publishes `ghcr.io/cartysio/libredesk:clyr` (workflow `clyr-image.yml`). Railway does **not** poll GHCR unless auto-updates are on or something triggers a redeploy.

## Automatic deploy (recommended)

### 1. GitHub Actions

After each successful image push, CI can run `railway redeploy --from-source` for staging then production.

**One-time setup** (repo admin):

1. GitHub → **CartysIO/libredesk** → **Settings** → **Environments** → create or open **staging** and **production**.
2. In each GitHub environment, add secret **`RAILWAY_TOKEN`**: the Railway **project token** for that environment (Railway → libredesk → same environment name → **Settings** → **Tokens**).

Both jobs read `${{ secrets.RAILWAY_TOKEN }}`; the value comes from the GitHub **environment** attached to the job (`staging` then `production`). Until a token is set, that job’s deploy step is skipped (warning in the workflow log).

### 2. Railway image auto-updates (optional backup)

For service **Libredesk** in each environment:

**Settings** → **Source** → **Configure auto updates** → maintenance window **Anytime**.

With image `ghcr.io/cartysio/libredesk:clyr`, Railway redeploys when a new digest is pushed to that tag (can lag a few hours per Railway docs).

## Manual deploy

```bash
chmod +x deploy/clyr-railway-redeploy.sh
./deploy/clyr-railway-redeploy.sh
```

Or one environment:

```bash
railway link -p 86f4a5dc-54b6-43b3-ad04-f88f0fd77916 -e staging -s Libredesk
railway redeploy --from-source -y
```

Use `--from-source` so Railway pulls the current digest for tag `clyr`, not the image from the last deploy.

## Verify

- `GET /api/v1/reports/issues?days=7` → **401** (not **404**) when the backend is current.
- Compare deploy digest in Railway with `docker buildx imagetools inspect ghcr.io/cartysio/libredesk:clyr-<sha>` (needs GHCR pull auth).
