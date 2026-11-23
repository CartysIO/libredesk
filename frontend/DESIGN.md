---
name: Clyr Support desk
description: Clyr's support desk (Signal Console visual system). Agent app only; widget and help center keep upstream Libredesk tokens.
colors:
  brand: "hsl(135 45% 58%)"
  primary: "hsl(213 52% 24%)"
  primary-foreground: "hsl(0 0% 100%)"
  background: "hsl(214 32% 96%)"
  foreground: "hsl(222 47% 11%)"
  foreground-lighter: "hsl(158 10% 38%)"
  card: "hsl(0 0% 100%)"
  muted: "hsl(150 14% 95.5%)"
  muted-foreground: "hsl(160 9% 36%)"
  accent: "hsl(150 18% 93.5%)"
  border: "hsl(150 12% 89%)"
  input: "hsl(150 10% 82%)"
  ring: "hsl(137 45% 46%)"
  canvas: "hsl(150 12% 89%)"
  private: "hsl(42 92% 93%)"
  destructive: "hsl(4 68% 45%)"
  success: "hsl(147 60% 30%)"
  sidebar-background: "hsl(150 16% 97%)"
  code-new: "hsl(137 50% 36%)"
  code-you: "hsl(11 80% 50%)"
  code-customer: "hsl(217 72% 50%)"
  code-hold: "hsl(215 14% 50%)"
  code-done: "hsl(155 8% 62%)"
  dark-background: "hsl(160 18% 7.5%)"
  dark-foreground: "hsl(140 12% 92%)"
  dark-card: "hsl(160 15% 10%)"
  dark-popover: "hsl(160 14% 12%)"
  dark-primary: "hsl(136 45% 58%)"
  dark-primary-foreground: "hsl(156 54% 9%)"
  dark-muted: "hsl(160 12% 13%)"
  dark-muted-foreground: "hsl(145 7% 68%)"
  dark-accent: "hsl(160 12% 15.5%)"
  dark-border: "hsl(160 10% 17%)"
  dark-input: "hsl(160 10% 21%)"
  dark-private: "hsl(38 35% 17%)"
  dark-destructive: "hsl(6 85% 70%)"
  dark-success: "hsl(140 50% 58%)"
  dark-sidebar-background: "hsl(160 16% 9%)"
  dark-code-new: "hsl(137 48% 56%)"
  dark-code-you: "hsl(12 85% 64%)"
  dark-code-customer: "hsl(217 85% 68%)"
  dark-code-hold: "hsl(215 12% 62%)"
  dark-code-done: "hsl(150 6% 46%)"
  rail: "hsl(155 54% 12%)"
  rail-raised: "hsl(153 42% 20%)"
  rail-text: "hsl(140 25% 96%)"
  rail-text-quiet: "hsl(145 20% 74%)"
  dark-rail: "hsl(156 52% 9%)"
  dark-rail-raised: "hsl(153 40% 16%)"
typography:
  title:
    fontFamily: "Lexend, ui-sans-serif, system-ui, sans-serif"
    fontSize: "18px"
    fontWeight: 600
    lineHeight: "26px"
  reference:
    fontFamily: "Lexend, ui-sans-serif, system-ui, sans-serif"
    fontSize: "18px"
    fontWeight: 700
    letterSpacing: "-0.03em"
    fontFeature: "tnum"
  body:
    fontFamily: "Source Sans 3, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: "20px"
  body-strong:
    fontFamily: "Source Sans 3, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "14px"
    fontWeight: 600
    lineHeight: "20px"
  email-body:
    fontFamily: "Source Sans 3, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.55
  meta:
    fontFamily: "Source Sans 3, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "12px"
    fontWeight: 500
    lineHeight: "16px"
    fontFeature: "tnum"
  label:
    fontFamily: "Lexend, ui-sans-serif, system-ui, sans-serif"
    fontSize: "11px"
    fontWeight: 700
    letterSpacing: "0.04em"
rounded:
  swatch: "1px"
  sm: "2px"
  md: "4px"
  lg: "6px"
  xl: "10px"
  full: "9999px"
spacing:
  rail: "48px"
  nav-column: "224px"
  lane: "288px"
  pane-gap: "1px"
  row-x: "12px"
  thread-x: "16px"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.primary-foreground}"
    rounded: "{rounded.md}"
    height: "36px"
  status-trigger:
    backgroundColor: "{colors.background}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.md}"
    padding: "4px 10px"
  list-row:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    typography: "{typography.body}"
    padding: "8px 12px"
  list-row-current:
    backgroundColor: "{colors.accent}"
  unread-count:
    backgroundColor: "{colors.foreground}"
    textColor: "{colors.background}"
    rounded: "{rounded.full}"
    height: "20px"
  board-card:
    backgroundColor: "{colors.card}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.md}"
    padding: "10px 12px"
  fold-row:
    textColor: "{colors.muted-foreground}"
    typography: "{typography.body}"
    padding: "8px 16px"
  rail:
    backgroundColor: "{colors.rail}"
    textColor: "{colors.rail-text}"
    width: "48px"
---

# Design System: Clyr Support desk

This file covers two systems in one codebase.

- **The Clyr desk system** (the agent app, `apps/main`) is primary. It is the frontmatter above and every canonical section below.
- **The upstream Libredesk system** stays in force for the chat widget and the public help center. It is kept intact in the last section, "Upstream Libredesk system".

**Source of truth for the desk:**
- Tokens: `apps/main/src/assets/desk-theme.scss`. Every rule is scoped to `html[data-app='desk']`, set in `apps/main/index.html`, so nothing leaks into the widget or help center.
- Tailwind mapping: `tailwind.config.cjs` (`brand` and `code.{new,you,customer,hold,done}`, both read `hsl(var(--x))`).
- Typography tokens: `apps/main/src/assets/desk-typography.scss` (families, scale, weights). Theme surface rules: `desk-theme.scss`.
- Fonts: Google Fonts in `apps/main/index.html` — **Lexend** (display) and **Source Sans 3** (body/UI). Upstream widget/help center keep **Geist** via `static/public/static/fonts.css`.
- Brand pieces: `apps/main/src/components/brand/ClyrMark.vue`, `StatusSwatch.vue`.
- Behaviour: `apps/main/src/utils/conversationCode.js`, `threadFold.js`, `initials.js` (each with a `.test.js`).

The desk overrides the upstream token names (`--primary`, `--border`, `--canvas` ...) rather than inventing new ones, so every shadcn primitive and upstream screen picks up the desk palette with no markup change. New desk-only tokens are `--brand`, `--code-*`, `--desk-font`, `--desk-font-display`, and the `--desk-type-*` / `--desk-leading-*` scale in `desk-typography.scss`. Values are HSL channel triples, like upstream; the frontmatter writes them as `hsl()`.

## Overview

**Creative North Star: "Signal Console"**

A calm operations console for support agents: cool slate grounds, Lexend headings, Source Sans 3 body copy, and navy for primary chrome. Clyr green remains identity on the mark and selection accents only. Panes float as rounded cards with light elevation on a soft gradient canvas—not a hairline grid. The whose-move code still carries ticket state on circular swatches.

Density is high but scannable. The agent reads rows by their coded swatch, opens a ticket whose reference number is the headline, and reads the newest email first, with older messages folded to one-line rows. Chrome (headers, recipients, quoted history, signatures) stays out of view until asked for.

The system rejects the grey shadcn admin with one green accent. Green is identity (rail, mark, primary action), and the whose-move code is the only other colour family in the chrome.

**Key Characteristics:**
- Deep green rail (48px) with the Clyr "C" mark; everything else is light panes on hairlines.
- One five-value whose-move code, shown as an 8px square swatch, filled or hollow.
- Lexend for headings and chrome emphasis; Source Sans 3 for body, forms, and tables. Rank by weight and case on a stepped scale (11–20px).
- Tabular figures for every number, time and reference.
- Gmail-style email thread: folded rows, newest open, as layered cards on a tinted thread ground with a floating composer card.
- Unread, priority and counts are neutral; they never borrow a code colour.

## Colors

Tinted neutrals and one deep green carry the chrome; a five-colour code carries ticket state, and nothing else is coloured.

### Primary
- **Clyr Brand Green** (`brand`): the clyr.io green (#62C47A in the brand book; the token renders #64C47C). Used for the Clyr mark on the rail, the rail's active item (`sidebar-primary`), text selection at 35% alpha, and as `primary` in dark mode. It is too light for white text, so it never fills a light-mode button.
- **Deep Desk Green** (`primary`, about #1A6141): primary buttons, links in chrome, checkbox and caret colour (`accent-color`, `caret-color`), selected-row tint (`primary/5`).

### Secondary: the whose-move code
One code, one meaning: whose turn it is on this ticket. The mapping lives in `CONVERSATION_BOARD_COLUMNS` (`apps/main/src/constants/conversation.js`) and is read through `statusCode()`.

- **New Green** (`code-new`): status Open, shown as "New". Nobody has touched it.
- **Signal Vermilion** (`code-you`): "On you". The customer is waiting on Clyr. The loudest colour in the system, on purpose.
- **Waiting Blue** (`code-customer`): "On customer". Clyr replied; the ball is with the customer.
- **Hold Slate** (`code-hold`): "On hold" (status Snoozed). Also the fallback for any custom status that has no board column.
- **Resolved Grey-Green** (`code-done`): "Resolved". The only hollow swatch.

Each code has a dark-mode value (`dark-code-*`) lifted in lightness so it reads on the dark panes.

### Neutral
- **Desk Ink** (`foreground`, about #0F1F1A): all primary text. A green-black, never pure black.
- **Quiet Ink** (`muted-foreground`, about #54645F): meta text, snippets, timestamps, placeholders. Holds 4.5:1 on the panes; the composer placeholder is forced to this colour instead of the editor's #adb5bd.
- **Pane White** (`card`, `popover`) and **Tinted Ground** (`background`, about #FCFDFC): work panes and content.
- **Nav Ground** (`sidebar-background`, about #F6F9F7): the secondary nav column (inboxes, views), one step darker than the panes.
- **Hairline** (`border`, about #E0E6E3): every divider. `canvas` is the same value, so the 1px gap between panes reads as a rule.
- **Field Line** (`input`): form-field borders and the status trigger's outline.
- **Note Amber** (`private`): private-note background, kept from upstream with a desk value.
- **Destructive** and **Success** keep their upstream roles (errors, delete, SLA breached; positive, verified) with desk values.

### Rail
The rail (`.desk-rail`) redefines its own tokens so shadcn sidebar parts, avatars and badges inside it sit on green: `rail` (about #0E2F21) ground, `rail-raised` (about #1E4835) for hover, active and avatar chips, `rail-text` and `rail-text-quiet` for icons and labels. In dark mode the rail drops to `dark-rail` (about #0B2319) and `dark-rail-raised`. It stays green in both themes.

### Named Rules
**The Whose-Move Rule.** The five code colours mean whose move it is, and they appear only on a status swatch. Other colour in the chrome is limited to brand green (identity) and the upstream semantic states (SLA badges, errors and delete, private-note amber).

**The Neutral Signal Rule.** Unread counts, unread dots and priority markers are neutral (`foreground` or `muted-foreground`). The unread count is a `foreground` pill with `background` text; the board unread dot is `foreground`; priority is the bar count of `PriorityIcon` (low 1 bar muted, medium 2 bars, high 3 bars in `foreground`), never red or amber.

**The One-Token-Name Rule.** The desk restyles upstream by giving upstream token names new values under `html[data-app='desk']`. Do not fork a component to recolour it.

## Typography

**Display font:** Lexend (`--desk-font-display`, Tailwind `font-display`). Headings, pane titles, ticket references, command palette input, toast titles, table headers, and metric values.

**Body / UI font:** Source Sans 3 (`--desk-font`, `--font-sans`, Tailwind `font-sans`). Paragraphs, list rows, form fields, buttons, sidebar items, descriptions, and chart labels (`--vis-font-family`).

**Mono:** system UI stack (`--desk-font-mono`, Tailwind `font-mono`) for kbd hints and shortcuts only.

**Character:** Lexend is tuned for reading proficiency and gives headings a calm, modern voice. Source Sans 3 is Adobe's workhorse for dense UI and long shifts at the desk. Together they replace upstream Geist inside `html[data-app='desk']` without forking components.

**Source file:** `apps/main/src/assets/desk-typography.scss` defines families, `--desk-type-*` sizes, `--desk-leading-*`, weights, and tracking tokens. Tailwind exposes `text-desk-*` utilities when those variables are in scope.

### Hierarchy
- **Page / pane title** (Lexend 600, 18px / 26px): `.desk-title`, list headers, dialog titles (`ui-dialog-heading`).
- **Reference** (Lexend 700, 18px, `tracking-tighter`, tabular): `.desk-ticket-ref` in the conversation header.
- **Section heading** (Lexend 600, 16px / 24px): `.ui-heading`, `h3`.
- **Subheading / subject** (Lexend or Source 600, 14px / 20px): `.ui-subheading`, ticket subject line.
- **Body** (Source 400, 14px / 20px): default UI copy, list snippets, sidebar values.
- **Body strong** (Source 600–700, 14px): sender names, unread rows, emphasis in lists.
- **Prose / email** (Source 400, 15px / ~1.55): `.native-html` in thread bubbles (inline email styles still win).
- **Meta** (Source 500, 12px, tabular): timestamps, counts, secondary lines.
- **Micro label** (Lexend 700, 11px, uppercase, `tracking-wider`): `.sidebar-section-label`, command group headings, admin table headers.
- **Caption** (Source 500, 12px, muted): `.desk-type-caption`, spec sidebar labels.

### Named Rules
**The Two-Family Rule.** Lexend carries hierarchy; Source Sans 3 carries everything else. Do not introduce a third UI face on the desk.

**The Tabular Rule.** Every reference number, time, count and date uses `tabular-nums` (or `data-desk-numeric`).

## Layout

Four columns on a 1px grid, left to right:

1. **Rail** (48px, `SIDEBAR_WIDTH_ICON`): Clyr mark at the top (24px tall, links to Inboxes), icon nav below.
2. **Nav column** (224px, `--sidebar-width: 14rem`): inboxes and views on `sidebar-background`, flush against the rail at `calc(3rem + 1px)`, with a right hairline.
3. **Work pane(s)**: the conversation list, then the ticket (header, thread, composer). On the board view, lanes replace the list.
4. **Context sidebar**: conversation details and contact, sections on hairline rules.

The app shell is `bg-canvas` with `gap-px` and no padding, so panes meet on a single 1px hairline. Rhythm is 4/8px; list rows pad 8px by 12px, thread cards 12px by 16px (14px bottom) with 12px between cards (0 inside a grouped run), fold rows 8px by 16px, spec rows 7px vertical. The thread itself pads 12px (16px from 640px) so its cards and the composer card share one left and right edge. On a wide pane the message column (email cards, fold stacks, chat bubbles, chips) stops at `--desk-thread-measure` (760px) and centres.

The thread pane (`.desk-message-list`) is a size container named `desk-thread`. Rows that adapt to a narrow column (fold rows, activity chips) use `@container desk-thread (max-width: 30rem)`, not a viewport query: the column is as narrow on a 1280px laptop with the list and sidebar open as on a phone. The rest of the shell follows upstream's responsive behaviour.

### Named Rules
**The Hairline Pane Rule.** Panes are separated by one 1px `border` line, never by a gutter, a rounded card or a shadow. Inside the ticket pane, the thread is the one place that layers (see The Layered Thread Rule).

## Elevation & Depth

Mostly flat. Depth comes from the hairline grid and three grounds (rail green, nav tint, pane white). Panes, list rows and board cards carry no shadow; board cards show state on hover by a border shift (`border` to `input`) and an `accent/40` wash. Popovers, menus and dialogs keep upstream's `shadow-md` / `shadow-lg`, because they really are above the page.

The conversation thread is the deliberate exception: it is a layered surface with three levels.

1. **Thread ground** (`--desk-thread-ground`): `background` mixed towards `accent` in light mode, plain `background` in dark. It fills the whole ticket body, behind the messages and behind the composer.
2. **Message surfaces** (`--desk-thread-surface`): `card` (lifted 5% towards `foreground` in dark, where `card` sits too close to the ground), a 1px hairline (`--desk-thread-hairline`) and `--desk-shadow-sm`. Email cards, stacked fold cards, chat bubbles, the typing bubble, day-separator chips and skeletons are all on this level. Tints stay light: our replies take a `--primary` wash (`--desk-thread-ours`, a step lighter on full-width email cards: `--desk-thread-ours-soft`), private notes a light `--private` tint (`--desk-thread-private`) plus a 3px inset `--warning` accent bar on the leading edge. A note keeps the hairline border; it never gets a full `--warning` border.
3. **Floating chrome**: the reply composer card, the scroll-to-bottom button, the older-messages spinner and the self-assign nudge use the same surface and hairline with `--desk-shadow-md`. The scroll-to-bottom button is a 32px circle centred in the gutter just above the composer. The composer adds a soft upward lift so messages read as passing under it, and a short ground-coloured fade sits on the top edge of its dock. A matching 12px fade sits on the thread's top edge, so a row cut by the pane edge reads as scrolled content, not a clipped chip.

All of these colours are derived from existing tokens (`hsl(var(--x) / alpha)` or `color-mix`); the thread adds no palette entries.

### Named Rules
**The Flat Pane Rule.** If it is part of the page, it has no shadow. Only overlays (menus, popovers, dialogs, the board drawer) lift.

**The Layered Thread Rule.** The exception to the Flat Pane Rule. Inside the thread, messages are rounded cards with a hairline and `--desk-shadow-sm`, and the composer is a rounded card with `--desk-shadow-md`, all on the thread ground. Nothing else in the page borrows this treatment, and no thread surface goes above `--desk-shadow-md`.

## Shapes

Tight corners. `--radius` is 6px on the desk (upstream 8px), so `rounded-lg` 6px, `rounded-md` 4px, `rounded-sm` 2px, `rounded-xl` 10px. Panes are square (radius 0). Thread cards, stacked fold cards and the composer card use `--radius`; chat bubbles use `--radius` + 4px, tightening to 4px on the sender's side inside a grouped run; a grouped run of email cards joins flush (inner corners square, one outer radius); activity and day-separator chips are pills. The "Private note" marker is plain text with a lock, not a pill. The whose-move swatch is an 8px square at 1px radius. Avatars and the unread count stay round.

## Components

### Whose-move swatch (signature)
The one mark that carries ticket state. `StatusSwatch.vue` renders an 8px by 8px square (`.code-swatch`, radius 1px) coloured by `currentColor` from `statusCode(status)`. **Filled** means someone must act (New, On you, On customer, On hold, any unmapped status). **Hollow** (`data-hollow`, a 1.5px inset ring, no fill) means Resolved. It has `role="img"` and the status as its `aria-label`. The same swatch appears on list rows, board cards, board lane headers, the ticket header status trigger and each item of the status menu.

### Conversation list row
- **Structure:** a 16px by 20px swatch cell, then name + priority + channel icon + time on line one, subject (14px `foreground`) on line two, snippet + unread count on line three, SLA badges below.
- **Rule:** a 1px `border/70` line under each row, no card.
- **States:** current `accent`; bulk-selected `primary/5` (hover `primary/10`); hover `accent/50`. On hover, or while selecting, the bulk checkbox takes the swatch's cell.
- **Unread:** name 700, snippet 500 `foreground`, and a neutral count pill (20px, `foreground` on `background`, "9+" cap).

### Ticket header
Two lines: `#reference` (16px bold, tabular) with the contact (12px muted) beside it, then the subject (14px semibold). The status trigger is an outline button (`input` border, `background`, 600 weight) with the swatch and the status name; it is not filled with the code colour.

### Email thread (signature)
Email conversations (`inbox_channel === 'email'`) render as a Gmail-style thread of cards on the thread ground (see The Layered Thread Rule).
- **Open cards:** full width up to the 760px measure, `--radius` corners, hairline border, `--desk-shadow-sm`. One surface: header and body share it, with no header strip and no divider. The header is a 32px avatar, then two lines (sender in Lexend 14px semibold; below it the lock + "Private note" label or the recipients summary that expands the envelope), and the time on the right (11px tabular, full timestamp in a tooltip; the date is dropped for today's messages).
- **Who is speaking:** four roles, read from the message itself (`messageRole()` in `utils/messageRole.js`: `private`, `author.type`, `type`, `sender_type`), never from names. **Customer** is ink, the other party: the plain thread surface with a 3px ink bar on its leading edge (`--desk-thread-customer-accent`, from `--foreground`), an inverted avatar (ink disc, `--background` initials) and a "Customer" tag beside the name. **Agent** is ours: the `--primary` wash and a primary-tinted round avatar, no tag. **AI assistant** is ours and automated: the same wash, plus a sparkle + "AI" tag and a squared avatar with a primary ring. **Private note** is amber, as below. The tag (`MessageRoleTag.vue`, `.desk-role-tag`) is an 11px Lexend chip shown once per run; a long translation truncates inside the chip (max 120px). The customer colour is ink only: never amber, never `code-customer` blue, never brand green. Our replies carry the soft `--primary` wash (`--desk-thread-ours-soft`), its border colour (`--desk-thread-ours-border`) and a primary-tinted avatar. A private note has the light `--private` tint, a 3px `--warning` accent bar on its leading edge, an amber-tinted avatar and a small lock + "Private note" label in `--desk-thread-private-ink` (no pill, no full amber border). The label keeps its text at every width; below 480px the header shows the time only (full date in `title`). The sender name owns its line, so it truncates only when the pane is narrower than the name. None of these use a code colour.
- **Quiet controls:** the ⋯ menu (delete note) and the collapse button sit left of the time and are hidden at rest where a pointer can hover (`hover: hover` and `pointer: fine`). They appear on card hover, on keyboard focus inside the card and while their menu is open. Touch devices always show them.
- **Grouped runs:** consecutive open messages from one sender form a run when each follows the previous within 5 minutes (`GROUP_WINDOW_MS`), or 10 minutes for private notes (`NOTE_GROUP_WINDOW_MS`). Only a message without an envelope (a note) continues a run; a real email always keeps its own header so its recipients stay reachable. The continuation sits flush under the previous card with a faint inner rule and no avatar or name; its controls and time appear on hover or focus as a small raised chip on the run's inner rule, top right, clear of the body text (a slim static row on touch). The chip pokes out of its card, so a continuation card does not clip (`overflow: visible`); its wide content scrolls inside the body wrapper (`.desk-message-content`) instead. A fold row, a failed message or another sender breaks the run.
- **What opens by default** (`defaultOpenUUIDs`): the newest message; every message after the last public agent reply; pending or failed sends; and a message a link scrolls to. Automated replies, CSAT and continuity sends do not count as agent replies.
- **Folded rows** (`MessageFoldRow.vue`): one line of 26px role avatar, sender, role tag ("Customer" or "AI"), lock + "Private note" label (on notes, at every width) and paperclip icons, a one-line snippet (max 160 characters), and time. The sender column is 144px. In a narrow column (container at most 30rem) the row shows the time only (full stamp in `title`) and the name hugs its text (max 128px, 84px on a row with a label or tag: note, AI, customer), so the label never drops; the name and the snippet truncate, the time never clips. A folded row keeps its role: a customer row has the ink bar and the inverted avatar, agent and AI rows the soft `--primary` wash (AI with its tag and squared avatar), a note the light tint and the amber bar. The "Customer" and "AI" tags stay at every width; the name and the snippet give way. The role disc colour is set on the avatar fallback itself (`.avatar-initials`), so disc and initials always change together. Click expands. Consecutive fold and summary rows join into one stacked card: outer `--radius`, inner hairlines, one shadow.
- **Long runs:** four or more consecutive folded messages (`FOLD_RUN_MIN = 4`) collapse to first row, an "N earlier messages" row with the count in a 24px ring, and last row.
- **Activity** ("assigned to…", status changes): a centred chip between two hairline stubs, never a card. The stubs drop in a narrow column (container at most 30rem).
- **Day separators** are hidden in email threads.
- **Quotes and signatures:** "Show quoted text" folds the Gmail quote block (with its "On ... wrote:" line) and the sender's signature. The toggle appears for a signature alone only when the email carries a Gmail (`gmail_signature`) or Outlook (`id="Signature"`) marker.

### Live chat thread
Chat channels render bubbles on the same thread ground.
- **Customer:** a `card` bubble on the left with hairline, `--desk-shadow-sm` and the 3px ink bar on its leading edge; the sender line has the inverted avatar and the "Customer" tag. **AI assistant:** the agent's bubble on the right, with the sparkle + "AI" tag in the sender line and the squared, ringed avatar. **Agent:** a `--primary`-washed bubble (`--desk-thread-ours`) on the right; a wash, not a filled primary bubble, so links, ticks and muted text keep their contrast. **Private note:** the email note's treatment: light `--private` tint, hairline border, 3px `--warning` accent bar on the leading edge, and the small lock + "Private note" label once per run. No dashed or full amber border.
- **Sender line:** once per run, above its first bubble: name (Lexend 12px semibold) and a quiet time (11px tabular, muted; time only, because the day separator carries the date; full timestamp in a tooltip). On our side the time sits on the inside and the name next to the avatar. There is no separate timestamp row under a run. A continuation bubble shows its own time beside it on hover or keyboard focus (pointer devices only; the slot is zero-width, so it never narrows the bubble).
- **Avatars:** 32px, by role like the email header's (customer inverted ink, agent primary, AI squared primary with a ring, note amber). From 1024px the avatar sits in a gutter beside the run's first bubble, lifted to the sender line; below that it sits in the sender line itself, so it never takes width from the bubbles.
- **Measure:** the chat column shares the email column's 760px cap and is centred. A bubble takes at most 75% of the column from 1024px and 88% below.
- **Grouping:** consecutive messages from one sender within 5 minutes of each other (`GROUP_WINDOW_MS`; 10 minutes for private notes, `NOTE_GROUP_WINDOW_MS`) form a run: sender line and avatar once, 4px between bubbles, sender-side corners tightened, and the delivery tick only on the last bubble. A failed message never groups, so its error and retry always show.
- **Day separators** are pill chips on a hairline; activity uses the same chip as in email. The typing indicator is a small incoming bubble, aligned with the bubbles inside the same measure.

### Reply composer
- **Dock:** the composer sits on the thread ground, not on a footer strip. A 14px ground-to-transparent fade on the dock's top edge lets messages scroll under it; the thread keeps at least 24px of bottom padding so the fade never covers the last timestamp.
- **Card:** thread surface, hairline, `--radius`, `--desk-shadow-md` plus a soft upward lift. Its side margins match the thread's padding. Focus inside turns the border to `ring` with a 2px `ring/10` glow. The board drawer keeps compact margins and `--desk-shadow-sm`.
- **Private note mode:** `--private` surface with a `--warning` border and a `--warning` focus glow. This input-state cue is the one place a note keeps a full amber border; notes in the thread do not.
- **Reply / Private note switch:** a recessed `foreground/7` track with a raised active segment (Lexend, semibold). Send stays the one primary button; while the draft is empty it is disabled at 85% opacity.
- **Collapsed (narrow screens):** a one-row card with the same surface, hairline, radius and shadow; expanding opens the fullscreen editor.

### Board
- **Lanes:** 288px columns on a right hairline (`border/70`), no rounded lane, no tinted lane fill; a drop target washes `accent/50`. The lane header is a swatch plus the lane name in 14px semibold `foreground`.
- **Cards:** 4px radius, 1px `border`, `card` fill, 10px by 12px padding, no shadow. Line one: swatch, `#reference` (12px bold tabular), unread dot (8px, `foreground`).
- **Drawer:** opens from the right over the board.

### Conversation details (spec table)
A two-column grid (`.desk-spec`): label column `max-content`, value column `minmax(0, 1fr)`, 12px gap. Every row is a subgrid, so all values align on one column. Each row has a hairline (`border/70`) below it, 7px vertical padding, baseline alignment. Labels are 12px at 500; values wrap at word boundaries, and only an unbroken string breaks mid-word. Subject and reference are hidden here because the header carries them. Dates show without seconds (`PP p`). The Information section is open on first use.

### Context sidebar sections
Accordion sections on a top hairline (`border/70`), no grey rounded blocks. Triggers are 12px uppercase labels with `tracking-wider` in `muted-foreground`, turning `foreground` on hover.

### Avatars and initials
Initials are the first letter of the first name and of the last name ("Dana Levi" gives "DL"); one name gives one letter; no name gives the fallback ("A" for agent, "U" for user, "?" elsewhere). Thread avatars (email header, chat) are 32px and fold-row avatars 26px, with semibold Lexend initials on a token-derived fill (`--desk-avatar-bg` / `--desk-avatar-ink`): inverted ink for customers (`--foreground` disc, `--background` initials), a `--primary` tint for agents, the same tint on a squared tile (8px radius, primary ring) for the AI assistant, `--warning` for notes. Never a code colour. The contact avatar in the sidebar is 48px. Inside the rail, avatar fallbacks sit on `rail-raised`.

### Buttons, inputs, menus
Upstream shadcn primitives, recoloured by the desk tokens: primary buttons in Deep Desk Green, 4px radius, sizes as upstream. Focus rings use `ring` (light `hsl(137 45% 46%)`).

## Do's and Don'ts

### Do:
- **Do** show ticket state with `StatusSwatch` and `statusCode()`, filled for every status but Resolved.
- **Do** keep chrome on the `--desk-type-*` scale; rank with Lexend vs Source Sans 3, weight, and case.
- **Do** use `tabular-nums` on every reference, time, date and count.
- **Do** separate panes, rows and sections with one 1px `border` line.
- **Do** keep thread messages and the composer as rounded cards on the thread ground, with `--desk-shadow-sm` for messages and `--desk-shadow-md` for the composer and floating thread chrome.
- **Do** show ours vs theirs with the `--primary` wash, and private notes with the light `--private` tint, a thin `--warning` accent bar and a visible lock + "Private note" label.
- **Do** group consecutive messages from one sender (5 minutes; notes 10) under one header in email and in chat.
- **Do** reveal secondary message controls (⋯, collapse, a continuation's time) on hover, and keep them reachable by keyboard focus and always visible on touch.
- **Do** keep unread and priority neutral (`foreground` / `muted-foreground`).
- **Do** scope new desk styling under `html[data-app='desk']` and reuse upstream token names, so rebases on upstream stay easy.
- **Do** keep incoming email on a light canvas in dark mode (`.dark .email-light-canvas`: white, #1a1a1a text, `color-scheme: light`). Sender colours in email HTML are only legible on white. This is a deliberate exception to the theme.

### Don't:
- **Don't** put a code colour on anything that is not a status swatch: no coloured chips, coloured left bars, coloured row tints or coloured lane fills.
- **Don't** colour unread counts, unread dots or priority markers.
- **Don't** add a fourth type size or a display tier.
- **Don't** give a private note a thick, dashed or full `--warning` border, or an outlined "Private note" pill, in the thread. A run of notes must not read as a wall of amber boxes.
- **Do** mark the customer with ink (leading bar, inverted avatar, "Customer" tag) and the AI assistant with its "AI" tag and squared avatar, in email cards, fold rows and chat alike.
- **Don't** colour a role with a code colour, amber (notes only) or brand green.
- **Don't** repeat avatar, name and controls on every message of a run, and don't add a timestamp row under each chat run.
- **Don't** float panes: no gutters, rounded pane cards or pane shadows. The Layered Thread Rule covers message cards and the composer only.
- **Don't** fill a light-mode button with brand green #62C47A; white text fails on it. Use `primary`.
- **Don't** hide a signature without a Gmail or Outlook marker. Signatures are only detected by those markers, never guessed.
- **Don't** restyle the chat widget or the help center with desk tokens; they follow the upstream system below.

## Upstream Libredesk system (chat widget and help center)

The rest of this file is the upstream Libredesk guide (v2.9.0). It still governs the chat widget and the public help center, and it is the base layer the desk overrides. Where a value below differs from the desk sections above, the desk value wins inside the agent app.

Update this guide only for verified shared styles and component usage. Keep screen decisions in PRs or audit notes.

Styling and component usage for the agent dashboard and shared widget UI.

### Sources

Read these files before changing shared styles:

- [Theme tokens and typography](shared-ui/assets/styles/main.scss)
- [Tailwind mappings](tailwind.config.cjs)
- [UI primitives](shared-ui/components/ui/)

Keep token values in their source files. Use the classes and components below when building UI.

### Colors and surfaces

Use semantic theme tokens for dashboard controls. The shared stylesheet provides light and dark values.
Pair background tokens with their foreground tokens, such as `bg-primary text-primary-foreground`.
Opacity modifiers such as `bg-success/10` work with these classes.

| Token | Use |
| --- | --- |
| `canvas` | Gutters behind content panels |
| `background` / `foreground` | Main surfaces and primary text |
| `card` / `card-foreground` | Cards and raised panels |
| `popover` / `popover-foreground` | Menus, popovers, and dropdowns |
| `primary` / `primary-foreground` | Primary actions and active states |
| `secondary` / `secondary-foreground` | Secondary controls and surfaces |
| `muted` / `muted-foreground` | Subtle surfaces, descriptions, and metadata |
| `accent` / `accent-foreground` | Hover and selected surfaces |
| `foreground-lighter` | Idle sidebar navigation text |
| `destructive` / `destructive-foreground` | Errors, removal, and breached targets |
| `success` / `success-foreground` | Successful, verified, or connected states |
| `warning` / `warning-foreground` | Warning backgrounds |
| `warning-600` | Warning text and icons |
| `link` | Links inside rendered content |
| `border` | Borders and dividers |
| `input` | Field borders |
| `ring` | Keyboard focus indicators |
| `private` | Private content tint |

Use `warning-600` for readable warning text in light mode.
Use `foreground` or `muted-foreground` for neutral counts and timestamps.
Pair small status colors with text, an icon, or a shape.

Use `sidebar-*` tokens for navigation surfaces and states.
Charts inherit `primary`, `success`, and `muted-foreground` through the shared Unovis variables.

Keep customer branding within customer surfaces and their previews.
Preserve authored colors in rendered email content. Use `.email-light-canvas` for a light email surface.

### Typography

Geist is the shared font. Use typography classes for these roles:

| Role | Class | Size | Weight | Line height |
| --- | --- | --- | --- | --- |
| Page, card, sidebar, and form section headings | `ui-heading` | 16px | 600 | 24px |
| Nested headings and compact item titles | `ui-subheading` | 14px | 600 | 20px |
| Descriptions and form hints | `ui-description` | 14px | 400 | 20px |
| Dialog and sheet titles | `ui-dialog-heading` | 18px | 600 | 28px |
| Large introductory headings | `ui-display-heading` | 24px | 600 | 32px |

Keep heading levels semantic when applying these classes.
`CardTitle`, `DialogTitle`, `SheetTitle`, and shared description components provide their default styles.
Use `tabular-nums` for metrics and numeric columns. Use 12px text for compact metadata.

### Spacing and layout

Use a 4px spacing rhythm. Common group spacing is 16px, 24px, or 32px.
Leave 16px between section headings and controls, with 24px or 32px between form sections.
Keep icon and value pairs together, with larger gaps between separate groups.

Use [FormGrid](shared-ui/components/FormGrid.vue) for responsive form fields.
Pass `compact` for narrower fields. Use container queries when columns need stable widths.
Keep short fields compact and preserve room for names, URLs, and translated labels.
Group conditional controls with their related fields.

Use [AdminSplitLayout](apps/main/src/layouts/admin/AdminSplitLayout.vue) for forms with help or a preview.
Put the form in `#content` and supporting content in `#help`.
Pass `fixedHelp` for the preview layout. The component manages responsive columns.

Use [AdminToolbar](apps/main/src/layouts/admin/AdminToolbar.vue) for list controls.
Put tabs in `#left` and actions in `#actions`.
Pass `searchable` and bind `v-model:search` for search controls.

### Radius and elevation

Use named Tailwind classes. The shared `--radius` value defines the control radius scale.

| Radius class | Use |
| --- | --- |
| `rounded-sm` | Small insets |
| `rounded-md` | Controls, chips, and badges |
| `rounded-lg` | Cards, containers, and dialogs |
| `rounded-xl` | Large surfaces |
| `rounded-2xl` | Widget frames and bubbles |
| `rounded-full` | Avatars, dots, and pills |

| Shadow class | Use |
| --- | --- |
| `shadow-sm` | Cards and standard controls |
| `shadow-md` | Popovers and floating controls |
| `shadow-lg` | Dialogs, modals, and widget windows |

### Buttons and indicators

Use the shared `Button` component for styled actions.
Reserve `default` for primary actions. Use `outline`, `secondary`, or `ghost` for supporting actions.
Use `destructive` for removal and `link` for text actions.

| Button size | Dimensions | Use |
| --- | --- | --- |
| `default` | 36px high | Standard controls |
| `sm` | 32px high | Dense controls with 12px text |
| `xs` | 28px high | Compact controls |
| `lg` | 40px high | Prominent actions |
| `icon` | 36px square | Icon actions |

Standard button icons use `size-4`. Keep icons within a group consistent in size, stroke, and color.
Give icon actions an accessible label and hide decorative icons with `aria-hidden="true"`.
Inside forms, give every supporting button `type="button"` and submit buttons `type="submit"`.

Use `Badge` variants: `default`, `secondary`, `destructive`, `success`, or `outline`.
Use `AlertDialogAction` with `variant="destructive"` for removal confirmations.
Use the shared `Tooltip` for its inverted surface, 12px text, and `rounded-md` corners.

### Fields and interaction states

Use shared `Input`, `Select`, combobox, and `TimeField` components to inherit control styles.
Standard fields use 36px height, with 16px text below `sm` and 14px above it.
Use visible labels. Placeholders provide examples or format guidance.
Use `text-muted-foreground` for empty prompts and 12px left padding for select and combobox text.

Keep focus rings visible. Reveal hover actions on keyboard focus and keep touch actions visible.
Use the existing `can-hover` variant for styles requiring a mouse or fine pointer.
Use shared component transitions and preserve their reduced motion behavior.

Use `LoadingOverlay` for page loading and `Spinner` for inline feedback.
Pass `reserve-height` when a page needs space for a centered loading indicator.

### Shared utilities

| Utility | Style |
| --- | --- |
| `.box` | Border, small shadow, and large radius |
| `.sidebar-section-label` | 12px muted uppercase text with wider tracking |
| `.link-style` | Muted underlined text with foreground hover color |

Use these utilities when their shared style fits the control.

### Admin tables

Show timestamps as `28 Aug 2026, 3:46 AM`. Keep seconds and the timezone in the full timestamp on hover.

Align headers and cells left. Let data columns share available width naturally, keep statuses and dates on one line, and keep action columns compact.
Wrap long text to three lines and expose the full value through its title. Keep horizontal overflow inside the table.
Measure rendered rows so wrapping and resizing preserve virtual scrolling.
Use compact sorting controls without hover backgrounds. Show unsorted arrows on hover or keyboard focus, and keep the active sort direction visible.
