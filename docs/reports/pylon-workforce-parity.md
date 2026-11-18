# Pylon Workforce SLA parity (imported history)

Derived from `internal/pylonimport/sla_hypothesis_test.go` against production `issues.json` (week `2026-11-09`).

## Resolution (locked)

| Piece | Rule |
| --- | --- |
| Week bucket | `DATE_TRUNC('week', pylon_created_at AT TIME ZONE app.timezone)`; fallback `meta.pylon.created_at`, then `conversations.created_at` |
| Universe | Conversations with `resolution_working_sec` or `meta.pylon_resolution_seconds` |
| Value | `COALESCE(resolution_working_sec, pylon_resolution_seconds)`; native tickets without Pylon meta still use calendar `resolution_met − created` |
| Aggregate | Median (`percentile_cont(0.5)`) |

## First response (best effort)

| Piece | Rule |
| --- | --- |
| Value | Calendar `first_response_met − created` |
| Exclude | Durations in **3.5h–4.5h** (Pylon auto-ack cluster) from medians and buckets |
| Note | Pylon UI **19h 15m** for that week is **not** reproduced as a non-degenerate median over the same 24 issues; see hypothesis tests |

## Tooling

```bash
go test ./internal/pylonimport -run Hypothesis -v
./libredesk --pylon-sla-reconcile --pylon-cache /path --pylon-reconcile-week 2026-11-09 \
  --pylon-reconcile-fr-target 19h15m --pylon-reconcile-res-target 22h53m
```

SQL implementation: `internal/report/queries.sql` (`get-workforce-charts`, `get-workforce-scalars`, Issues `median_sla_by_week`).