# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Clyr's internal support and customer-success team (today: Eran, Omer, Rodrigo, all Admin agents). They work on desktop browsers, all day, answering B2B customers of Clyr (property managers, contractors, SMB finance teams using Clyr for cards, receipts, bills and accounting sync). Inferred from the team and usage; not a confirmed device survey.

## Product Purpose

Clyr Support is Clyr's fork of Libredesk, replacing Pylon as the support desk. Agents triage and answer tickets from email (support@clyr.io), the in-app chat widget, and a Slack bridge, move them across a status board (New, On you, On customer, On hold, Resolved), and hand work to AI help (the "Juno" copilot panel). Success: a ticket is understood at a glance and answered fast, and nothing waits on Clyr without someone seeing it.

## Positioning

A self-hosted desk shaped to Clyr's own workflow: customers are organizations keyed by Clyr teamId, sign-in is Clyr's Clerk instance, and ticket state reflects whose move it is (On you vs On customer).

## Operating Context

- High-volume, dense desk work: list + ticket + context sidebar side by side, a Kanban board with a side drawer, keyboard shortcuts and a command palette.
- Most tickets are email threads with quoted history, signatures and logos; replies go out as email.
- Pylon is still live during the migration; some messages are mirrors of replies sent from Pylon.

## Capabilities and Constraints

- Stack: Vue 3 + Tailwind + shadcn-vue (`shared-ui/components/ui`), Go backend; one process serves API + built UI.
- Scope of the facelift (confirmed 2026-10-20): the agent app (`frontend/apps/main`) only. The customer chat widget and public help center keep their current look.
- Upstream constraint (confirmed): keep rebases onto upstream Libredesk easy. Prefer changes to design tokens (`shared-ui/assets/styles/main.scss`, `tailwind.config.cjs`) and shared components, plus a few key screens (message feed, conversation list, sidebar). Avoid rewriting whole upstream screens.
- i18n: edit only `i18n/en-US.json`.
- Both light and dark themes exist and must keep working.

## Brand Commitments

Clyr brand (confirmed): logo (`clyr-site/media-archive/wp-content/uploads/2022/11/Clyrlogo.svg`), brand green `#62C47A`, Urbanist typeface, as used on clyr.io.

## Evidence on Hand

- Live desk: https://clyr-support.up.railway.app (production), https://libredesk-staging-6deb.up.railway.app (staging).
- Brand assets in `C:\Coding\Clyr\clyr-site` (logo files, CSS with `--color-green: #62C47A`, Urbanist).

## Product Principles

1. Density over decoration: the agent sees more of the conversation and more tickets per screen.
2. Whose move it is must always be obvious.
3. Customer text is the content; chrome (headers, metadata, quoted history) recedes until asked for.
4. Stay close to upstream so the fork can keep taking Libredesk releases.
