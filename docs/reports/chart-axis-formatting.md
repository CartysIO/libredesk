# Report chart and table formatting

Product spec for `/reports` (Overview, Issues, Workforce, SLA Breaches). Implemented in `frontend/apps/main/src/features/reports/`.

## Time semantics

| Source | Meaning on the client |
| --- | --- |
| API fields `date`, `week` (`YYYY-MM-DD`) | **Civil calendar labels** already bucketed in **`app.timezone`** on the server (`AT TIME ZONE` in SQL). Parse with `parseReportDay()` (local `Date(y, m-1, d)`), never `parseISO` alone, so the label does not shift with the viewer’s offset. |
| API hour `0–23` | Hour-of-day in **`app.timezone`** at event time. Shown as a 12-hour clock label, not a raw integer. |
| API `dow` `1–7` (ISO, Mon–Sun) | Day-of-week in **`app.timezone`**. Localized weekday name; do not use SQL `Dy` strings. |
| Instants (`lastUpdate`, `created_at` without display) | Format in **`app.timezone`** with the viewer’s **locale** (labels), 12-hour clock. |

**Viewer locale** (`vue-i18n` → `date-fns` via `reportDateFnsLocale`) controls language and date order. **App timezone** (`app.timezone` from general settings) controls instant and bucket interpretation.

## Formatter matrix

| Data kind | Axis / cell | Tooltip / crosshair title | Tables / scalar cards |
| --- | --- | --- | --- |
| Count | `formatChartCount` | same | integers / compact `Intl` for large counts |
| Rate (%) | one decimal + `%` | same | one decimal + `%` |
| Duration (seconds) | `formatCompactDuration` | `formatResolutionDaysHours` | `formatReportDurationSeconds` (null → `—`, `0` → `0s`, else `formatResolutionDaysHours`) |
| Day bucket | `formatReportDate` (short `P`) | same as axis | N/A |
| Week bucket | `formatReportDate` (week **start** on axis) | `formatReportWeekTooltipTitle` (“Week of …” + Mon–Sun range) | N/A |
| Hour bucket | `formatReportHour12` (`h:mm a`) | same | N/A |
| Day of week | localized `EEE` from `dow` | same | N/A |
| Duration category (`<1h`, …) | API label as-is | same | N/A |
| Instant | N/A | N/A | `formatReportDateTimeInZone` / `formatReportDateTimeReadable` (+ optional `created_at_display`) |

## Chart components

- **`IssuesBarChart` / `IssuesLineChart`**: `format-index-as-date` (daily), `format-index-as-week` (weekly), `duration-axis` (Y compact, values detailed).
- **`OverviewLineChart`**: daily index, shared index formatters.
- Wrappers use `useReportChartIndexFormatters` for X-axis and crosshair titles.

## Composables

- **`useReportFormatContext()`** — `dateLocale`, `appTimezone`, `i18nLocale` (for `Intl`).
- **`useReportLocale()`** — shorthand for `dateLocale` only.

## i18n

- `report.chart.weekOf`, `report.chart.weekRange` — en-US source strings; Crowdin for other languages.

## References

- SQL buckets: `internal/report/queries.sql` (`$1` = app timezone).
- Workforce parity notes: [pylon-workforce-parity.md](pylon-workforce-parity.md).