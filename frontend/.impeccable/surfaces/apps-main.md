---
version: 1
slug: "apps-main"
primary_target: "apps/main"
related_targets: []
---

# Surface brief: agent app (frontend/apps/main)

Scope: the agent desk (inbox list, ticket thread, right sidebar, board, nav). Mode: Operate. Widget and help center out of scope.
Audience/job: Clyr's 3-person support team, desktop, all day; read a ticket fast, see whose move it is, reply.
Constraints: keep upstream rebases easy (tokens, shared components, a few key screens); light and dark must work; i18n only en-US.json.
Feed decision (user, 2026-10-19): Gmail-style thread. Older messages fold to one-line rows; latest + unread open; headers, quotes, signatures hidden until clicked.
Avoid (user): marketing-site feel, dark hacker look, too dense to scan.

## Direction contract

THESIS: Clyr's identity manual applied to the desk: one strict grid, flush-left Urbanist, colour only as a code for whose move it is. Refuses the grey shadcn admin with one green accent.

OWN-WORLD: Deep Clyr-green rail (#0E2F22) with light type; tinted neutral ground; white work panes on hairline rules, no floating cards. Clyr green #62C47A is identity; a darker green carries text-on-light. Whose-move code: On you = signal vermilion, On customer = blue, New = green, On hold = slate, Resolved = muted. Three type sizes (12/13/15), rank by weight and case. Tabular figures.

STORY: The agent opens the desk, scans rows by their coded bar, opens a ticket, reads the newest reply first with history folded above, and replies without scrolling.

FIRST VIEWPORT: Left: green rail (nav). Next: ticket list, each row a 3px coded bar, sender, subject, snippet, time. Centre: header with #ref as headline, subject beneath, status chip in its code; thread with folded one-line rows, latest open; reply box. Right: details as a two-column spec table.

FORM: Swiss International Typographic Style identity manual; my list position 6 of 7; seed key 0445dd6f. Signature move: the whose-move colour code (a square swatch, not a side bar: the craft floor refuses coloured borders above 1px on list items), the same bar on list rows, board cards, and the ticket header. Raises: palette discipline (Datamatics), isolated destructive actions (Console), #ref as anchor (Catalog Sleeve), three sizes only (Timetable).

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
