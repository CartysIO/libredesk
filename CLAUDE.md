# CLAUDE.md

This is Clyr's fork of Libredesk (Go + Vue 3, AGPL-3.0). Clyr uses it to replace Pylon as its support desk.

- Upstream: https://github.com/abhinavxd/libredesk
- Fork: https://github.com/CartysIO/libredesk (public)
- Plan and progress: Claude Docs, https://claude.ai/code/artifact/02264c00-b0c1-4c8b-8ca0-586185b68183. Its decisions table is the source of truth. Track work in its "Progress" section with one dated entry per finished step. Do not create Linear tickets.

## Branches and PRs

- `clyr` is the default branch and the working branch. Changes reach it through PRs from feature branches (`eranartzi/<topic>`), squash-merged.
- `main` mirrors upstream. Never push to `main` or `master`.
- `gh pr create` on a fork targets upstream by default. Always pass `--repo CartysIO/libredesk --base clyr`.
- Baz reviews every PR. Fix or answer each thread, reply in the thread, then resolve it through GraphQL `resolveReviewThread`. Merge only when all checks pass on the head commit and no thread is open.
- Translations: edit only `i18n/en-US.json`. CI (`i18n-guard`) blocks changes to other language files; Crowdin manages them.
- To rebase onto a new upstream release: fetch upstream tags, rebase `clyr` onto the tag on a branch, and run the full test suite and the image build before you move `clyr`.

## Build and test

```bash
make build                          # frontend + Go binary + stuffed assets
go test ./...                       # integration tests need LIBREDESK_TEST_DB_DSN (make test-db)
cd frontend && npx pnpm@9.15.3 install --frozen-lockfile && npx vitest run
docker build -f Dockerfile.clyr -t libredesk-clyr:local .
```

- Go changes: add tests for every new package, run `gofmt` and `go vet`.
- Frontend: ESLint and Prettier. Some upstream files were already unformatted; do not reformat files you did not change.
- Before you report a change as working, run it: boot the image against Postgres 17 and Redis 7 and exercise the changed path (HTTP request or headless browser).

## Clyr additions in this fork

- `Dockerfile.clyr`, `deploy/start.sh`, `.github/workflows/clyr-image.yml`: the image is built from source and runs as a non-root user. On boot it installs the schema the first time, runs upgrades, and binds to `$PORT`. CI builds the image on PRs to `clyr`. On each push to `clyr` it publishes the private image `ghcr.io/cartysio/libredesk:clyr` and `:clyr-<sha>`.
- Clerk sign-in (`internal/clerk`, `cmd/auth_clerk.go`, `frontend/apps/main/src/utils/clerk.js`, `views/auth/UserLoginView.vue`).
  - The login page mounts Clerk's drop-in `SignIn` component. It loads `@clerk/ui@1` and `@clerk/clerk-js@6` from the Clerk Frontend API at runtime; v6 needs both, through `Clerk.load({ ui: { ClerkUI: window.__internal_ClerkUICtor } })`.
  - `POST /api/v1/auth/clerk` checks the session token against the instance's JWKS. It checks the issuer, `exp`/`nbf`, and that `azp` equals the root URL origin. It then reads the user's verified primary email from the Clerk Backend API.
  - The agent with that email is signed in. Libredesk does not create accounts at sign-in.
  - Enabled when `[clerk] publishable_key` and `secret_key` are set.
- Planned: organizations (customer companies keyed by Clyr teamId), a Slack bridge Go module for #user_support, and a Pylon import script in `scripts/`. See the plan doc for the design.

## Configuration

Config comes from environment variables: `LIBREDESK_<SECTION>__<KEY>`, for example `LIBREDESK_DB__HOST` or `LIBREDESK_CLERK__SECRET_KEY`. `config.sample.toml` lists the keys. Some settings live in the database instead (for example `app.root_url`), and you change them through the admin API.

## Infrastructure

- Railway project `libredesk` (id `86f4a5dc-54b6-43b3-ad04-f88f0fd77916`, Clyr workspace).
  - Environment `staging` (`b05dd541-5662-4ed7-a6f4-d0c0554b2c18`).
  - Services: Postgres, Redis, and `Libredesk` (`7e663356-68f3-4c61-9343-adee48f9da3f`). URL: https://libredesk-staging-6deb.up.railway.app.
  - The database and Redis connect through Railway reference variables.
  - GHCR pull credentials are set on the service through GraphQL `serviceInstanceUpdate` (`registryCredentials`). The token is in Doppler `clyr-infra/prd` (`GHCR_PULL_USERNAME`, `GHCR_PULL_TOKEN`).
  - Deploy after config changes with GraphQL `serviceInstanceDeployV2`. `railway redeploy` reuses the previous deploy's settings.
  - To prove a deploy landed, compare the deploy's image digest with `docker buildx imagetools inspect ghcr.io/cartysio/libredesk:clyr-<sha>`.
- Doppler project `libredesk` (configs `stg`, `prd`) holds the app secrets. It does not sync to Railway yet. Until it does, copy changed values to Railway by hand.
- Cloudflare R2: wrangler reaches Clyr's account with `CLOUDFLARE_ACCOUNT_ID=fd9fae3a3d42838655d338dba386f4d2`. Bucket `libredesk-staging` exists. The wrangler login cannot create R2 S3 keys.
  - Never give Libredesk the Clyr backend's R2 key: it reaches every bucket, receipts included.
- Clerk: staging uses the dev instance (`sacred-chimp-95`). A production Clerk instance only works on clyr.io domains.

## Working with the running instance

- Admin API: sign in with `POST /api/v1/auth/login` as `System` (password in Doppler `libredesk/<config>` as `LIBREDESK_SYSTEM_USER_PASSWORD`). Keep the cookies, and send the `csrf_token` cookie's value as the `X-CSRFTOKEN` header on writes.
- The Clerk Backend API rejects requests that have no User-Agent: use `curl -A`.
- The `rtk` shell hook rewrites the output of curl, grep and docker. Use `rtk proxy <cmd>` for raw output.
- Browser checks: run `agent-browser` headless with a new session, `--session <name> --allowed-domains "<host>,*.clerk.accounts.dev,*.accounts.dev"`. The default session uses an allowlist from `~/.zshrc`.
- The dogfood Clerk user is in Doppler `clyr-server-v2/stg` (`E2E_CLERK_USER_EMAIL`, `E2E_CLERK_USER_PASSWORD`). On staging it maps to the agent "Dogfood E2E".

## Reporting to Eran

Give the result first, then at most 5 one-line bullets with bold labels. No time estimates. No markdown blockquotes. Ask questions one line each, numbered, with your recommendation. Never say done, fixed or deployed without a check that ran in the same turn.
