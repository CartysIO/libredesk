# Workforce report (v1) — omitted metrics and data requirements

This document lists **Pylon-style workforce widgets that v1 does not ship**, why Libredesk cannot compute them today, and **what data or product work** is needed to add them later.

**Shipped in v1:** [`/reports/workforce`](../../frontend/apps/main/src/views/reports/WorkforceView.vue) — API [`GET /api/v1/reports/workforce`](../../cmd/report.go) — SQL [`get-workforce-scalars` / `get-workforce-charts`](../../internal/report/queries.sql).

**Universe (v1):** One row per conversation = **latest** `applied_slas` row whose `created_at` falls in the selected range. Durations use **`conversation.created_at`** as the start timestamp and **calendar elapsed time** (wall clock), same spirit as Overview SLA averages.

---

## Summary table

| Pylon-style metric | In v1? | Blocker |
| --- | --- | --- |
| Median / bucket **first response** and **resolution** (labeled “business hours”) | Partial | Durations are calendar time only; no business-hours elapsed helper in reports |
| **Handle time** (median active work on a ticket, business hours) | No | No agent active-time or focus-time storage |
| **Out of office / Away** tables (time away, away while breaching, etc.) | No | Only live `users.availability_status`; no history |
| Assignee charts **at time of event** | Partial | Uses current `conversations.assigned_user_id`, not historical owner |
| **Shifts / scheduled coverage** | No | No shift or capacity model |
| **Per-agent CSAT** in Workforce | No | CSAT exists elsewhere; not joined to assignee-at-response |
| Customer **queue wait** vs **agent wait** split | No | Needs assignment + message-level semantics |
| Median time **in status** (Workforce tab) | No | Implemented on **Issues** report only (`status_events` from messages) |

---

## 1. Business-hours elapsed duration (first response, resolution, histograms, weekly medians)

**Pylon:** Labels such as “Issue Time to First Response **in Business Hours**” and the same for resolution and bucket charts.

**v1 behavior:** All Workforce duration metrics use `met_at - conversation.created_at` (or breach timestamp when met is missing) in **ordinary elapsed seconds**. The UI states this explicitly (`report.workforce.calendarTimeNote`).

**What we already have**

- `app.business_hours_id` and `app.timezone` (general settings).
- Business hours definitions and **forward** SLA deadline math in Go (`internal/sla/calculator.go`, `internal/business_hours`).

**What is missing**

- A **reusable inverse**: working seconds between two instants `(start, end, businessHours, timezone)` for reporting (not only deadline projection).
- Optional **precomputed columns** on `applied_slas` (e.g. `first_response_working_sec`, `resolution_working_sec`) if SQL-only reports must stay fast at scale.
- Report queries must use the **same** holiday / always-open rules as SLA application (including `outside_business_hours` automation semantics).

**To ship:** Expose the helper from Go (batch job or SQL function calling a small service is acceptable), then duplicate Issues/Workforce median and bucket SQL with working-second expressions and relabel UI copy.

---

## 2. Median business-hours handle time (agent active work)

**Pylon:** “Median business hours handle time” (and related agent-effort views).

**v1:** Omitted entirely.

**What we have today**

- Message send times, assignments, SLA met/breach timestamps.
- No notion of “agent had this conversation open” or “clock running while handling.”

**What is missing (product definition + schema)**

Agree on a definition, then persist events, for example:

| Definition (example) | Data to store |
| --- | --- |
| Time from **assignment** to next **outgoing agent message** | `conversation_assignments` history + messages (still calendar unless §1 exists) |
| Time with conversation **focused in UI** | Client heartbeat or `conversation_agent_sessions` (`user_id`, `conversation_id`, `started_at`, `ended_at`) |
| Time between **status: pending on customer** vs **agent working** | Status timeline with agent-attributed segments (see Issues `status_events` pattern, extended for assignee) |

For **business-hours handle time**, combine the chosen model with §1.

---

## 3. Out of office / Away analytics

**Pylon:** Tables such as time out of office, away during SLA breaches, compare agents’ away time.

**What we have today**

- `users.availability_status` (enum: `online`, `away`, `away_manual`, `away_and_reassigning`, `offline`) — **current state only**.
- `users.last_active_at` — last activity ping, not structured away intervals.
- Activity log entries for some away/reassign actions (`internal/activity_log`) — **not** a complete interval history suited to aggregation.

**What is missing**

- Append-only **`user_availability_events`** (recommended shape):
  - `user_id`, `status` (or `from_status` / `to_status`), `effective_at`, optional `source` (manual, auto-away, system).
  - Written on every availability change (API/WebSocket path that updates `users.availability_status`).
- Backfill is optional; reporting can start from cutover date.
- For “away while ticket breaching,” join intervals to `applied_slas` breach timestamps and open conversations.

**To ship:** Migration + writer on availability updates; Workforce SQL aggregating `SUM(end - start)` per agent in range (calendar or business hours per §1).

---

## 4. Assignee attribution (historical owner)

**Pylon:** Per-assignee medians and breach rates imply **who owned the work** when the metric occurred.

**v1 behavior:** Groups by **`conversations.assigned_user_id` as it is now**, for conversations in the SLA universe. Reassignments are not replayed.

**What is missing**

- **`conversation_assignments`** (or equivalent): `conversation_id`, `user_id`, `assigned_at`, `unassigned_at` (nullable = still assigned).
  - Populate on assign/unassign/round-robin (same code paths that set `assigned_user_id`).
- Reporting joins: attribute first response to assignee at `first_response_met_at`, resolution at `resolution_met_at`, etc.

**Lighter alternative:** Parse structured fields from `activity_log` / assignment messages — fragile; a first-class table is preferred.

---

## 5. Shifts, staffing, and capacity

**Pylon (if present in your export):** Coverage by schedule, expected headcount, etc.

**What we have:** Teams, agents, business hours for **customers** (SLA), not agent shift calendars.

**What is missing**

- Shift definitions per team or agent (`shift_templates`, `agent_shifts` with timezone).
- Optional integration with HR / calendar systems.

**To ship:** Model + admin UI, then Workforce widgets comparing SLA performance to scheduled coverage.

---

## 6. CSAT and quality metrics on Workforce

**What we have:** CSAT flows and overview reporting elsewhere in Libredesk.

**v1 Workforce:** No CSAT-by-assignee block.

**What is missing**

- Link each CSAT response to **agent at time of survey or resolution** (depends on §4).
- Optional: exclude bots, separate by channel.

---

## 7. Queue wait vs agent wait (decomposition)

**Pylon-style sophistication:** Separate customer wait before first human action vs time while assigned but silent.

**What is missing**

- First **agent** outgoing message timestamp (partially available in `conversation_messages`).
- **Assignment history** (§4).
- Optional: business-hours variant (§1).

**To ship:** SQL or Go pipeline defining milestones per conversation, then Workforce scalar/chart endpoints.

---

## 8. Metrics present on Issues but not duplicated on Workforce

These are **intentionally** on [Issues](/reports/issues), not Workforce v1:

- Issue volume, tags, sources, touches, top accounts.
- **Median time in each status by week** (derived from status change messages in SQL).

Workforce can later embed a subset if product wants one combined “operations” page.

---

## 9. Next-response SLA depth

**v1 includes:** `next_response_breach_percent` from `sla_events` (`type = 'next_response'`) in range, tied to latest applied SLA per conversation.

**Not included:** Per-assignee next-response medians, weekly next-response series, buckets — same gaps as §1 and §4 for accurate agent-level next-response analytics.

**What we have:** `sla_events` with `deadline_at`, `met_at`, `breached_at`, `status`.

**What is missing for parity:** Historical assignee (§4), business-hours duration (§1), and explicit product rules when multiple next-response cycles exist per conversation (which event counts in a range).

---

## 10. Breach rate by hour / day of week

**v1 behavior:** Rates are computed over the Workforce **SLA universe**, grouped by **`conversation.created_at`** hour/DOW in app timezone (aligned with Issues “by hour” pattern), not by breach timestamp.

**If Pylon uses breach moment instead:** Store or compute breach instant explicitly and add an alternate grouping dimension; no new schema required beyond SLA tables already present.

---

## Suggested implementation order (phase 2+)

1. **Business-hours elapsed helper** (§1) — unlocks honest Pylon labeling for FR/resolution.
2. **Assignment history** (§4) — unlocks trustworthy assignee tables and CSAT joins.
3. **Availability history** (§3) — unlocks Away / OOO tables.
4. **Handle time definition + sessions** (§2) — largest product + client surface.
5. Shifts (§5) and queue decomposition (§7) — optional, depends on ops requirements.

---

## References in repo

| Area | Location |
| --- | --- |
| Workforce SQL | `internal/report/queries.sql` (`get-workforce-*`) |
| Range semantics | `internal/report/issues_range.go` |
| SLA storage | `schema.sql` — `applied_slas`, `sla_events` |
| Business hours / SLA math | `internal/sla/calculator.go`, `internal/business_hours` |
| Agent availability (live) | `users.availability_status` in `schema.sql` |
| Issues status medians | `get-issues-charts` in `internal/report/queries.sql` |

When a section above is implemented, update this file and shorten the corresponding row in the summary table.