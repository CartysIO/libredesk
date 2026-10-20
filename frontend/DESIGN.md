---
name: Clyr Support desk
description: Clyr's support desk (a Libredesk fork). The agent app wears the Clyr desk system; the chat widget and help center keep the upstream Libredesk system.
colors:
  brand: "hsl(135 45% 58%)"
  primary: "hsl(153 58% 24%)"
  primary-foreground: "hsl(0 0% 100%)"
  background: "hsl(150 20% 99%)"
  foreground: "hsl(160 35% 9%)"
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
    fontFamily: "Urbanist, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "16px"
    fontWeight: 600
    lineHeight: "24px"
  reference:
    fontFamily: "Urbanist, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "16px"
    fontWeight: 700
    letterSpacing: "-0.025em"
    fontFeature: "tnum"
  body:
    fontFamily: "Urbanist, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: "20px"
  body-strong:
    fontFamily: "Urbanist, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "14px"
    fontWeight: 600
    lineHeight: "20px"
  email-body:
    fontFamily: "Urbanist, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.55
  meta:
    fontFamily: "Urbanist, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: "16px"
    fontFeature: "tnum"
  label:
    fontFamily: "Urbanist, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "12px"
    fontWeight: 600
    letterSpacing: "0.05em"
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
- Font: `static/public/static/fonts.css` (Urbanist variable, roman and italic, weights 100 to 900, SIL OFL), preloaded in `apps/main/index.html`.
- Brand pieces: `apps/main/src/components/brand/ClyrMark.vue`, `StatusSwatch.vue`.
- Behaviour: `apps/main/src/utils/conversationCode.js`, `threadFold.js`, `initials.js` (each with a `.test.js`).

The desk overrides the upstream token names (`--primary`, `--border`, `--canvas` ...) rather than inventing new ones, so every shadcn primitive and upstream screen picks up the desk palette with no markup change. New desk-only tokens are `--brand`, `--code-*` and `--desk-font`. Values are HSL channel triples, like upstream; the frontmatter writes them as `hsl()`.

## Overview

**Creative North Star: "The Identity Manual Desk"**

Clyr's identity manual applied to a support desk: one strict grid, flush-left Urbanist, and colour used only as a code for whose move it is. A deep Clyr-green rail holds navigation. Next to it, white work panes meet on 1px hairlines over a faintly green-tinted ground. Nothing floats: no gutters, no rounded pane cards, no pane shadows.

Density is high but scannable. The agent reads rows by their coded swatch, opens a ticket whose reference number is the headline, and reads the newest email first, with older messages folded to one-line rows. Chrome (headers, recipients, quoted history, signatures) stays out of view until asked for.

The system rejects the grey shadcn admin with one green accent. Green is identity (rail, mark, primary action), and the whose-move code is the only other colour family in the chrome.

**Key Characteristics:**
- Deep green rail (48px) with the Clyr "C" mark; everything else is light panes on hairlines.
- One five-value whose-move code, shown as an 8px square swatch, filled or hollow.
- Three type sizes (12 / 14 / 16), Urbanist only, rank by weight and case.
- Tabular figures for every number, time and reference.
- Gmail-style email thread: folded rows, newest open.
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

**Display Font:** none. The desk has no display tier.
**Body Font:** Urbanist (fallback `ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif`), via `--desk-font`.
**Label/Mono Font:** none; labels are Urbanist too.

**Character:** Urbanist is Clyr's own face from clyr.io: geometric, open, friendly at small sizes. It replaces upstream Geist on `html`, `body`, `.font-sans`, chart text (`--vis-font-family`) and form controls (`fonts.css` makes buttons and inputs inherit the page font).

### Hierarchy
- **Title** (600, 16px, 24px line): page and panel titles. Upstream `text-xl` titles marked `desk-title`, and every `text-xl` in the secondary nav column, are stepped down to 16px.
- **Reference** (700, 16px, `tracking-tight`, tabular): the `#reference` in the ticket header. It is the ticket's headline; the contact name sits beside it at 12px muted and the subject on the second line at 14px semibold.
- **Body** (400, 14px): list snippets, values, messages. Email HTML in a thread reads at 14px with a 1.55 line height; inline email styles still win.
- **Body Strong** (600, 14px): sender names on list rows, fold rows and thread headers, the ticket subject. Unread list rows go to 700 on the name and 500 `foreground` on the snippet.
- **Meta** (400, 12px, tabular): times, recipients summary, channel and counts.
- **Label** (600, 12px, uppercase, `tracking-wider`, muted): the right sidebar's section headers (accordion triggers). Spec-table field labels are 12px at 500, sentence case.

### Named Rules
**The Three Sizes Rule.** The desk uses 12, 14 and 16px. Rank comes from weight (400 / 500 / 600 / 700) and case, not size.

**The Tabular Rule.** Every reference number, time, count and date uses `tabular-nums`.

## Layout

Four columns on a 1px grid, left to right:

1. **Rail** (48px, `SIDEBAR_WIDTH_ICON`): Clyr mark at the top (24px tall, links to Inboxes), icon nav below.
2. **Nav column** (224px, `--sidebar-width: 14rem`): inboxes and views on `sidebar-background`, flush against the rail at `calc(3rem + 1px)`, with a right hairline.
3. **Work pane(s)**: the conversation list, then the ticket (header, thread, composer). On the board view, lanes replace the list.
4. **Context sidebar**: conversation details and contact, sections on hairline rules.

The app shell is `bg-canvas` with `gap-px` and no padding, so panes meet on a single 1px hairline. Rhythm is 4/8px; list rows pad 8px by 12px, thread blocks 12px by 16px (14px bottom), fold rows 8px by 16px, spec rows 7px vertical.

On narrow screens the fold row's sender column shrinks from 144px to 96px; the rest of the shell follows upstream's responsive behaviour.

### Named Rules
**The Hairline Pane Rule.** Panes are separated by one 1px `border` line, never by a gutter, a rounded card or a shadow.

## Elevation & Depth

Flat. Depth comes from the hairline grid and three grounds (rail green, nav tint, pane white). Panes, list rows, thread blocks and board cards carry no shadow; board cards show state on hover by a border shift (`border` to `input`) and an `accent/40` wash. Popovers, menus and dialogs keep upstream's `shadow-md` / `shadow-lg`, because they really are above the page.

### Named Rules
**The Flat Pane Rule.** If it is part of the page, it has no shadow. Only overlays (menus, popovers, dialogs, the board drawer) lift.

## Shapes

Tight corners. `--radius` is 6px on the desk (upstream 8px), so `rounded-lg` 6px, `rounded-md` 4px, `rounded-sm` 2px, `rounded-xl` 10px. Panes and thread blocks are square (radius 0). The whose-move swatch is an 8px square at 1px radius. Avatars and the unread count stay round.

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
Email conversations (`inbox_channel === 'email'`) render as a Gmail-style thread. Live chat keeps upstream bubbles.
- **Open blocks:** full width, square, a bottom hairline, no shadow. A one-line header: 24px avatar, sender (14px semibold), lock for private notes, a recipients summary that expands the envelope, time (12px tabular, full timestamp in a tooltip), and a collapse button.
- **What opens by default** (`defaultOpenUUIDs`): the newest message; every message after the last public agent reply; pending or failed sends; and a message a link scrolls to. Automated replies, CSAT and continuity sends do not count as agent replies.
- **Folded rows** (`MessageFoldRow.vue`): one line of avatar, sender, lock and paperclip icons, a one-line snippet (max 160 characters), and time. Click expands.
- **Long runs:** four or more consecutive folded messages (`FOLD_RUN_MIN = 4`) collapse to first row, an "N earlier messages" row with the count in a 24px ring, and last row.
- **Day separators** are hidden in threads.
- **Quotes and signatures:** "Show quoted text" folds the Gmail quote block (with its "On ... wrote:" line) and the sender's signature. The toggle appears for a signature alone only when the email carries a Gmail (`gmail_signature`) or Outlook (`id="Signature"`) marker.

### Board
- **Lanes:** 288px columns on a right hairline (`border/70`), no rounded lane, no tinted lane fill; a drop target washes `accent/50`. The lane header is a swatch plus the lane name in 14px semibold `foreground`.
- **Cards:** 4px radius, 1px `border`, `card` fill, 10px by 12px padding, no shadow. Line one: swatch, `#reference` (12px bold tabular), unread dot (8px, `foreground`).
- **Drawer:** opens from the right over the board.

### Conversation details (spec table)
A two-column grid (`.desk-spec`): label column `max-content`, value column `minmax(0, 1fr)`, 12px gap. Every row is a subgrid, so all values align on one column. Each row has a hairline (`border/70`) below it, 7px vertical padding, baseline alignment. Labels are 12px at 500; values wrap at word boundaries, and only an unbroken string breaks mid-word. Subject and reference are hidden here because the header carries them. Dates show without seconds (`PP p`). The Information section is open on first use.

### Context sidebar sections
Accordion sections on a top hairline (`border/70`), no grey rounded blocks. Triggers are 12px uppercase labels with `tracking-wider` in `muted-foreground`, turning `foreground` on hover.

### Avatars and initials
Initials are the first letter of the first name and of the last name ("Dana Levi" gives "DL"); one name gives one letter; no name gives the fallback ("A" for agent, "U" for user, "?" elsewhere). Thread and fold avatars are 24px with 10px semibold initials. The contact avatar in the sidebar is 48px. Inside the rail, avatar fallbacks sit on `rail-raised`.

### Buttons, inputs, menus
Upstream shadcn primitives, recoloured by the desk tokens: primary buttons in Deep Desk Green, 4px radius, sizes as upstream. Focus rings use `ring` (light `hsl(137 45% 46%)`).

## Do's and Don'ts

### Do:
- **Do** show ticket state with `StatusSwatch` and `statusCode()`, filled for every status but Resolved.
- **Do** keep chrome to 12, 14 and 16px Urbanist; rank with weight and case.
- **Do** use `tabular-nums` on every reference, time, date and count.
- **Do** separate panes, rows and sections with one 1px `border` line.
- **Do** keep unread and priority neutral (`foreground` / `muted-foreground`).
- **Do** scope new desk styling under `html[data-app='desk']` and reuse upstream token names, so rebases on upstream stay easy.
- **Do** keep incoming email on a light canvas in dark mode (`.dark .email-light-canvas`: white, #1a1a1a text, `color-scheme: light`). Sender colours in email HTML are only legible on white. This is a deliberate exception to the theme.

### Don't:
- **Don't** put a code colour on anything that is not a status swatch: no coloured chips, coloured left bars, coloured row tints or coloured lane fills.
- **Don't** colour unread counts, unread dots or priority markers.
- **Don't** add a fourth type size or a display tier.
- **Don't** float panes: no gutters, rounded pane cards or pane shadows.
- **Don't** fill a light-mode button with brand green #62C47A; white text fails on it. Use `primary`.
- **Don't** hide a signature without a Gmail or Outlook marker. Signatures are only detected by those markers, never guessed.
- **Don't** restyle the chat widget or the help center with desk tokens; they follow the upstream system below.

## Upstream Libredesk system (chat widget and help center)

The rest of this file is the upstream Libredesk reference. It still governs the chat widget and the public help center, and it is the base layer the desk overrides. The desk keeps its golden rule, its button `type="button"` rule and its checklist. Where a value below differs from the desk sections above, the desk value wins inside the agent app.

**Source of truth:**
- Tokens (CSS variables): `shared-ui/assets/styles/main.scss` - `:root, .light` and `.dark` blocks
- Tailwind mapping: `tailwind.config.cjs` - `theme.extend.colors` and `borderRadius`
- Primitives: `shared-ui/components/ui/` (shadcn-vue)

**Golden rule:** never hardcode a value that a token exists for. Use `bg-success`, not
`bg-green-600`. Use `rounded-lg`, not `rounded`. Every token has a light and a dark value;
if you add one, define both.

### U1. Color tokens

HSL channel triples in CSS variables, consumed via `hsl(var(--x))` (Tailwind classes like
`bg-primary` do it for you). Opacity modifiers work: `bg-success/10`, `text-primary/80`.

| Token | Light | Dark | Use for |
|---|---|---|---|
| `background` | `0 0% 99.2%` | `120 2.6% 7.6%` | app/content surface |
| `foreground` | `0 0% 1%` | `150 6% 93%` | primary text |
| `foreground-lighter` | `0 0% 41%` | `150 1% 60%` | idle sidebar nav items |
| `card` / `card-foreground` | `0 0% 100%` / `0 0% 1%` | `120 2% 10%` / `150 6% 93%` | raised card surface |
| `popover` / `popover-foreground` | `0 0% 100%` / `0 0% 1%` | `120 2% 12%` / `150 6% 93%` | menus, popovers, dropdowns |
| `primary` / `primary-foreground` | `152 39% 30%` / `0 0% 100%` | `152 58% 54%` / `120 2.6% 7.6%` | brand, active state, unread badges, primary buttons |
| `secondary` / `secondary-foreground` | `0 0% 96%` / `0 0% 1%` | `150 3% 12%` / `150 6% 93%` | secondary buttons, outgoing message bubbles |
| `muted` / `muted-foreground` | `0 0% 96%` / `0 0% 27%` | `150 3% 13%` / `120 1% 74%` | muted backgrounds, captions/meta text, secondary labels |
| `accent` / `accent-foreground` | `0 0% 95%` / `0 0% 1%` | `150 3% 15%` / `150 6% 93%` | hover and selected states |
| `destructive` / `destructive-foreground` | `2 47% 46%` / `9 100% 99%` | `4 92% 74%` / `12 38% 3%` | errors, delete, SLA breached, overdue, offline |
| `success` / `success-foreground` | `142 72% 37%` / `0 0% 98%` | `142 55% 55%` / `142 40% 10%` | positive/met, verified, online, connected, delivered |
| `warning` / `warning-foreground` | `39 85% 43%` / `24 45% 2%` | `36 87% 62%` / `24 45% 2%` | away, pending, SLA approaching, connecting (as background) |
| `warning-600` | `35 92% 33%` | `36 87% 62%` | warning as TEXT/icon color. Plain `warning` fails 4.5:1 on light |
| `link` | `210 100% 40%` | `210 90% 66%` | links inside rendered email/message content only. UI links use `.link-style` |
| `border` | `0 0% 91%` | `150 2% 16%` | all borders/dividers |
| `input` | `0 0% 85%` | `150 2% 19%` | form field borders |
| `ring` | `151 41% 45%` | `152 45% 33%` | focus rings |
| `private` | `35 90% 94%` | `30 35% 18%` | private-note background tint |
| `canvas` | `0 0% 82%` | `120 3% 4%` | app gutter behind floating panels (deepest surface) |

**Sidebar tokens** (left nav chrome): `sidebar-background` matches `background` in both
themes, plus `sidebar-foreground`, `sidebar-primary`, `sidebar-accent`,
`sidebar-accent-foreground`, `sidebar-border`, `sidebar-ring`.

**Chart tokens** (unovis): `--vis-primary-color: var(--primary)`,
`--vis-secondary-color: var(--success)`, `--vis-text-color: var(--muted-foreground)`.

#### Semantic status mapping

Color carries meaning; use the semantic token, never a raw palette color.

- **success (green):** SLA met, identity verified, agent online, widget connected, message delivered/read
- **warning (amber):** agent away, SLA approaching/remaining, widget connecting, no-internet banner
- **destructive (red):** error, delete, SLA breached, SLA overdue
- **primary (green):** brand identity, active nav, unread count badges, primary actions
- **foreground / muted:** neutral data (counts, totals, timestamps). Do not color a number
  unless the color means something. On the reports dashboard numbers are neutral; only
  met=success and breached=destructive are colored.

#### Deliberate exceptions

- `features/conversation/message/attachment/BubbleAttachmentItem.vue` colors attachment
  icons by file type (pdf=red, spreadsheet=green, doc=blue, archive=amber, audio=purple)
  with raw palette classes. The color is file-type identity, not status, and no token means
  "blue = document". Leave it.
- `components/editor/TextEditor.vue` - styles for rendered email HTML. Emails are
  standalone documents, not themed.
- `features/admin/inbox/LivechatInboxForm.vue` / `LivechatWidgetPreview.vue` - the
  customer-configurable widget brand color and its defaults.
- `features/conversation/ReplyBox.vue` - a `linear-gradient(#000 0 0)` CSS mask trick, not
  a color choice.


### U2. Surfaces and depth

Both themes use three tiers so panels read as floating, not flat:

```text
canvas (deepest gutter behind the panels)
  └─ background (app content + sidebar chrome, same value)
        └─ card / popover (lifted by border + shadow-sm)
```

In light mode `card` is pure white against a near-white background. In dark mode `card` and
`popover` are lighter than the background, so lift comes from the surface, not the shadow.


### U3. Typography

Font: Geist. Sizes: `text-xs` 12 · `text-sm` 14 · `text-base` 16 · `text-lg` 18 ·
`text-xl` 20 · `text-2xl` 24 · `text-3xl` 30. Weights: 400 body · 500 labels · 600 headings ·
700 rare emphasis.

| Role | Style |
|---|---|
| Page / panel title | `text-xl font-semibold` |
| KPI / stat value | `text-2xl` (or `text-3xl`) `font-bold tabular-nums` |
| Section label | `.sidebar-section-label` utility, or the classes it applies |
| Body | `text-sm` |
| Caption / meta / helper | `text-xs text-muted-foreground` |

Use `tabular-nums` for any numeric column, timer, or metric to stop width jitter.


### U4. Radius

`--radius` = `0.5rem` (8px). The Tailwind scale derives from it:

| Class | Value | Use for |
|---|---|---|
| `rounded-xl` | radius + 4 (12px) | large surfaces, widget window preview, dashboard message bubbles |
| `rounded-lg` | radius (8px) | cards, containers, dialogs |
| `rounded-md` | radius - 2 (6px) | buttons, inputs, chips, badges, small interactive |
| `rounded-sm` | radius - 4 (4px) | tiny insets |
| `rounded-full` | - | avatars, status dots, count badges, pills |

**Never use bare `rounded`.** It is Tailwind's fixed 4px and ignores the token.


### U5. Elevation

| Class | Use for |
|---|---|
| `shadow-sm` | cards, `.box`, default buttons |
| `shadow-md` | popovers, dropdown menus, hover/floating elements |
| `shadow-lg` | dialogs, modals, the widget window |

**Never use bare `shadow`.** Depth comes from the surface tiers and borders, not heavy
shadows.


### U6. Spacing

Follow a 4 / 8px rhythm for padding and gaps. Vertical section spacing tiers: 16 / 24 / 32 /
48. Keep the dense-desk feel; this is a high-volume support tool, not a marketing page.


### U7. Utilities

Defined in `main.scss`. Prefer them over repeating the class list.

| Utility | Expands to | Use for |
|---|---|---|
| `.box` | `border shadow-sm rounded-lg` | the standard card surface |
| `.sidebar-section-label` | `text-xs font-medium uppercase tracking-wider text-muted-foreground` | sidebar group headers (Views, Team Inboxes) |
| `.link-style` | `text-muted-foreground underline underline-offset-4 hover:text-foreground` | UI links. Not brand-colored, so links in chrome stay quiet |


### U8. Components

Reuse `shared-ui/components/ui/` primitives; do not hand-roll a styled `<button>`/`<input>`
when one exists.

**Button** - use the `size` variant, never an ad-hoc `h-*`:

| size | height | notes |
|---|---|---|
| `default` | h-9 | standard |
| `sm` | h-8 | dense (text-xs) |
| `xs` | h-7 | very dense |
| `lg` | h-10 | prominent |
| `icon` | h-9 w-9 | icon-only |

Variants: `default` (primary), `destructive`, `outline`, `secondary`, `ghost`, `link`.
`h-8 w-8` ghost triggers for dense table row actions are an intentional pattern.

Every button inside a `<form>` that is not the submit button needs an explicit
`type="button"`. HTML defaults to `type="submit"`, so a typeless button submits the form on
click and gets activated when the user presses Enter in any input. This includes buttons in
child components rendered inside a parent's form.

**Badge** variants: `default`, `secondary`, `destructive`, `success`, `outline`.

**AlertDialogAction** takes a `variant` prop, so destructive confirms use
`variant="destructive"` instead of a hand-written class.

**Sidebar nav items** are `font-medium text-foreground-lighter`; hover and active states use
`bg-sidebar-accent` (hover at `/50`). Group headers use `.sidebar-section-label`.

**Tooltip** is `bg-foreground text-background`, not brand-colored.

**Table row actions** stay hidden until hover: `[@media(hover:hover)]:opacity-0` with
`group-hover/row:opacity-100`, plus `focus-within:!opacity-100` and
`[&:has([data-state=open])]:!opacity-100`. The media query keeps them visible on touch and
focus-within keeps them reachable by keyboard.


### U9. Checklist

Before merging UI work:

- [ ] No hardcoded palette colors (`bg-green-600`, `text-blue-500`, ...). Only exception: file-type icons.
- [ ] No bare `rounded` or `shadow`.
- [ ] Every non-submit button in a form has `type="button"`.
- [ ] Color on a number or element means something, and is not decoration.
- [ ] Both light and dark verified.
- [ ] New shared words go through i18n (`i18n/en-US.json`); reused nouns in `globals.terms`.
- [ ] Reused an existing `ui/` primitive rather than building a variant.
