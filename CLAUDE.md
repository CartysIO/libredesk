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
- Fork migrations: name each one as a prerelease of the latest upstream version in `cmd/upgrade.go` `migList`, for example `v2.9.0-rc.9.clyr.1`, and add it after that version. A database never runs a migration that sorts below its last recorded version. Put every schema change in `schema.sql` too, and make the migration idempotent: a fresh install runs every migration against the final schema.
- Frontend: ESLint and Prettier. Some upstream files were already unformatted; do not reformat files you did not change.
- Before you report a change as working, run it: boot the image against Postgres 17 and Redis 7 and exercise the changed path (HTTP request or headless browser).

## Clyr additions in this fork

- `Dockerfile.clyr`, `deploy/start.sh`, `.github/workflows/clyr-image.yml`: the image is built from source and runs as a non-root user. On boot it installs the schema the first time, runs upgrades, and binds to `$PORT`. CI builds the image on PRs to `clyr`. On each push to `clyr` it publishes the private image `ghcr.io/cartysio/libredesk:clyr` and `:clyr-<sha>`.
- Clerk sign-in (`internal/clerk`, `cmd/auth_clerk.go`, `frontend/apps/main/src/utils/clerk.js`, `views/auth/UserLoginView.vue`).
  - The login page mounts Clerk's drop-in `SignIn` component. It loads `@clerk/ui@1` and `@clerk/clerk-js@6` from the Clerk Frontend API at runtime; v6 needs both, through `Clerk.load({ ui: { ClerkUI: window.__internal_ClerkUICtor } })`.
  - `POST /api/v1/auth/clerk` checks the session token against the instance's JWKS. It checks the issuer, `exp`/`nbf`, and that `azp` equals the root URL origin. It then reads the user's verified primary email from the Clerk Backend API.
  - The agent with that email is signed in. Libredesk does not create accounts at sign-in.
  - Enabled when `[clerk] publishable_key` and `secret_key` are set.
  - When the domain changes, set `app.root_url` to the new origin: `GET /api/v1/settings/general`, change `app.root_url`, and `PUT` the full object back. The PUT replaces every general setting, so a partial body resets the others. Until then every Clerk sign-in fails with "authorized party … want …". Change the Slack app's Request URL too.
- Organizations (`internal/organization`, `cmd/organizations.go`): customer companies keyed by Clyr teamId (`external_id`).
  - A contact belongs to several organizations (`contact_organizations`). `users.organization_id` is its default organization, and it is always one of those rows.
  - A conversation has one `organization_id`. A new email conversation gets the contact's default. A chat gets the organization from the widget JWT, which the widget session stores. An agent can change it to another of the contact's organizations (permission `conversations:update_organization`).
  - `POST /api/v1/organizations/sync` adds memberships and never removes one. A contact with `default: true` gets that organization as its default. Each organization can also carry `domains` (added to the stored ones) and `custom_attributes` (merged into the stored ones). `PUT /api/v1/organizations/{id}` replaces whichever of the two it gets.
  - Custom attributes can apply to `organization` (stored in `organizations.custom_attributes`). Context links get `{{organization_id}}`, `{{organization_name}}`, `{{organization_external_id}}` and `{{organization_custom_attributes.<key>}}`. Automation conditions: `organization` (equals, or contains = "is any of" a list of IDs) and field type `organization_custom_attribute`.
  - An email contact with no organization joins the one organization that has its email domain (`AssignByEmailDomain`). Free-mail domains and a domain that two organizations share are ignored.
- Automation condition `outside_business_hours` ("true"/"false") checks the time a rule runs against the default business hours (`app.business_hours_id`, holidays included) in `app.timezone`. With no business hours set it is false.
- Slack bridge (`internal/slackbridge`, `cmd/slack_bridge.go`): one Slack thread per conversation in one channel.
  - It gets conversation events through `eventFanout` (`cmd/slack_bridge.go`), which wraps the webhook manager. All its work runs on one worker goroutine, in order; that order is what stops a note from Slack being posted back.
  - Slack sends events to `POST /webhooks/slack/events`, signed with the signing secret. A thread reply becomes a private note by the agent with the Slack user's email; a reply that mentions the bot is sent to the customer.
  - Enabled when `[slack] bot_token`, `signing_secret` and `channel_id` are set. The app manifest is `deploy/slack-app-manifest.yml`.
- Planned: a Pylon import script in `scripts/`. See the plan doc.

## Configuration

Config comes from environment variables: `LIBREDESK_<SECTION>__<KEY>`, for example `LIBREDESK_DB__HOST` or `LIBREDESK_CLERK__SECRET_KEY`. `config.sample.toml` lists the keys. Some settings live in the database instead (for example `app.root_url`), and you change them through the admin API.

## Infrastructure

- Railway project `libredesk` (id `86f4a5dc-54b6-43b3-ad04-f88f0fd77916`, Clyr workspace).
  - Environment `staging` (`b05dd541-5662-4ed7-a6f4-d0c0554b2c18`): https://libredesk-staging-6deb.up.railway.app.
  - Environment `production` (`fce6e5f4-08f9-458f-a485-ed512e2404fe`): https://clyr-support.up.railway.app.
  - Each environment has Postgres, Redis, a bucket, and `Libredesk` (`7e663356-68f3-4c61-9343-adee48f9da3f`, same service id in both). One process serves the API and the built UI; keep it that way (decided 2026-10-16).
  - The database and Redis connect through Railway reference variables (`${{Postgres.*}}`, `${{Redis.REDIS_URL}}`). They live only on Railway, because Doppler cannot store `${{…}}` values.
  - Production Postgres has daily, weekly and monthly backups. Better Stack monitor "Libredesk production" (id 5083974) checks the site every 3 minutes.
  - GHCR pull credentials are set on the service through GraphQL `serviceInstanceUpdate` (`registryCredentials`). The token is in Doppler `clyr-infra/prd` (`GHCR_PULL_USERNAME`, `GHCR_PULL_TOKEN`).
  - Deploy after config changes with GraphQL `serviceInstanceDeployV2`. `railway redeploy` reuses the previous deploy's settings.
  - To prove a deploy landed, compare the deploy's image digest with `docker buildx imagetools inspect ghcr.io/cartysio/libredesk:clyr-<sha>`.
- Doppler project `libredesk` (configs `stg`, `prd`) holds the app secrets. Each config syncs to the `Libredesk` service in its environment through the "Railway Connection" integration, and a change redeploys the service. Change secrets in Doppler, never on Railway.
  - The sync adds, updates, and removes the keys it owns. It leaves Railway-only variables alone.
  - Never create a sync with `service_id: "shared"`. Its failed initial sync made Doppler disable the whole Railway connection, clyr-v2 production syncs included (2026-10-16). Before you add any sync, ask Eran.
- Uploads: Railway Buckets (S3 API, endpoint `https://t3.storageapi.dev`, region `auto`), one bucket per environment with its own keys. Settings are the `LIBREDESK_UPLOAD__S3__*` keys in Doppler.
  - Cloudflare R2 is not used: the wrangler login cannot create R2 S3 keys. The bucket `libredesk-staging` in R2 is unused. Never give Libredesk the Clyr backend's R2 key: it reaches every bucket, receipts included.
- Clerk: both environments use the dev instance (`sacred-chimp-95`). A production Clerk instance only works on clyr.io domains.
- Agents: Eran, Omer and Rodrigo (`@clyr.io`, Admin) and "Dogfood E2E" exist on both environments.

## Working with the running instance

- Admin API: sign in with `POST /api/v1/auth/login` as `System` (password in Doppler `libredesk/<config>` as `LIBREDESK_SYSTEM_USER_PASSWORD`). Keep the cookies, and send the `csrf_token` cookie's value as the `X-CSRFTOKEN` header on writes.
- The Clerk Backend API rejects requests that have no User-Agent: use `curl -A`.
- The `rtk` shell hook rewrites the output of curl, grep and docker. Use `rtk proxy <cmd>` for raw output.
- Browser checks: run `agent-browser` headless with a new session, `--session <name> --allowed-domains "<host>,*.clerk.accounts.dev,*.accounts.dev"`. The default session uses an allowlist from `~/.zshrc`.
- The dogfood Clerk user is in Doppler `clyr-server-v2/stg` (`E2E_CLERK_USER_EMAIL`, `E2E_CLERK_USER_PASSWORD`). On both environments it maps to the agent "Dogfood E2E". For a headless sign-in, create a sign-in ticket with the Clerk Backend API (`POST /v1/sign_in_tokens`).

## Reporting to Eran

Give the result first, then at most 5 one-line bullets with bold labels. No time estimates. No markdown blockquotes. Ask questions one line each, numbered, with your recommendation. Never say done, fixed or deployed without a check that ran in the same turn.
