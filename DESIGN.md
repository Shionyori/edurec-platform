---
name: edurec
description: 学习桌面 — a warm editorial design system for a personalized learning recommender.
colors:
  primary: "#C9451F"
  primary-hover: "#B23A15"
  primary-active: "#9C3010"
  primary-soft: "#FBECE6"
  primary-softer: "#FDF5F1"
  surface: "#FFFFFF"
  surface-2: "#FCFAF6"
  paper-bg: "#F7F4EE"
  sunken: "#F1ECE3"
  border: "#E7E1D6"
  border-strong: "#D4CABB"
  control: "#8A8172"
  ink: "#0B1B2B"
  ink-secondary: "#4C5661"
  ink-muted: "#6B665C"
  rail: "#0B1B2B"
  rail-soft: "#14293D"
  line-teal: "#0E7C7B"
  line-blue: "#1D4ED8"
  line-amber: "#A15C00"
  line-rose: "#BE123C"
  success: "#0E7C5A"
  warning: "#A15C00"
  danger: "#B3261E"
typography:
  display:
    fontFamily: 'Manrope, system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "52px"
    fontWeight: 800
    lineHeight: 1.08
    letterSpacing: "-0.035em"
  headline:
    fontFamily: 'Manrope, system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "30px"
    fontWeight: 750
    lineHeight: 1.2
    letterSpacing: "-0.03em"
  section:
    fontFamily: 'Manrope, system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "24px"
    fontWeight: 750
    lineHeight: 1.3
    letterSpacing: "-0.025em"
  title:
    fontFamily: 'Manrope, system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "16px"
    fontWeight: 700
    lineHeight: 1.45
    letterSpacing: "-0.015em"
  body:
    fontFamily: 'Manrope, system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.65
    letterSpacing: "0em"
  small:
    fontFamily: 'Manrope, system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.7
    letterSpacing: "0em"
  micro:
    fontFamily: 'Manrope, system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "12px"
    fontWeight: 650
    lineHeight: 1.5
    letterSpacing: "0em"
rounded:
  field: "10px"
  card: "14px"
  panel: "16px"
  pill: "999px"
spacing:
  "1": "4px"
  "2": "8px"
  "3": "12px"
  "4": "16px"
  "5": "20px"
  "6": "24px"
  "8": "32px"
  "10": "40px"
  "12": "48px"
  "14": "56px"
  "20": "80px"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.surface}"
    typography: "{typography.micro}"
    rounded: "{rounded.field}"
    padding: "0 18px"
    height: "40px"
  button-primary-hover:
    backgroundColor: "{colors.primary-hover}"
  button-secondary:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.micro}"
    rounded: "{rounded.field}"
    padding: "0 18px"
    height: "40px"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.field}"
    padding: "10px 12px"
    height: "42px"
  card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.card}"
    padding: "20px"
  chip:
    backgroundColor: "{colors.sunken}"
    textColor: "{colors.ink-secondary}"
    typography: "{typography.micro}"
    rounded: "{rounded.pill}"
    padding: "3px 10px"
  dark-panel:
    backgroundColor: "{colors.rail}"
    textColor: "{colors.surface}"
    rounded: "{rounded.panel}"
    padding: "28px"
  mobile-tabbar:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink-muted}"
    typography: "{typography.micro}"
    height: "68px"
---

# Design System: edurec

## Overview

**Creative North Star: "学习桌面 / The Study Desk"**

edurec is a Chinese educational-resource recommender. Its interface is a calm study desk: warm paper, deep ink, one vermilion pen. The product's job is to answer "下一站，学什么？" and the design makes that answer feel like a note placed on the desk — a real recommendation card, a readable reason, a path that unlocks one step at a time.

The previous visual world (transit signage: navy rail bands, 4px corners, 2px route strokes, station dots, uppercase micro-labels) is gone. The replacement keeps the product's existing name, routes, data, and vocabulary, but changes the visual language completely: generous editorial type, 14px paper cards, soft hairlines, large quiet spacing, and one warm signal color. It is still **Operate** mode: the user is here to complete a task, so familiar controls, scanable tables, and predictable navigation outrank decoration.

The system is deliberately warm rather than "enterprise blue": the ground is `#F7F4EE`, the ink is `#0B1B2B`, and the action color is a burnt vermilion `#C9451F`. Manrope is self-hosted for Latin letters and numerals; Chinese falls back to PingFang / Microsoft YaHei. A small four-point spark is the only brand mark. A live "下一步" card on the home hero — real recommendation data, not an illustration — is the signature move.

**Key Characteristics:**
- Warm paper ground, white paper cards, deep ink type.
- One vermilion signal for action and current state; semantic level colors only inside level/type badges and graphs.
- Editorial scale: 52px display, 30px headline, 24px section, 16px card title, 14px body.
- Small, exact geometry: 10px fields, 14px cards, 16px panels; pills only for chips and compact controls.
- Borders at rest, one ink-tinted hover lift; no ambient card shadows.
- A dark "ink panel" is reserved for orientation moments: auth, discipline/knowledge mastheads, and the learning-path header.
- Mobile gets a real bottom tab bar instead of a squeezed desktop nav.

## Colors

A warm editorial palette: one signal color, a family of paper neutrals, and a small semantic set that only appears when it carries meaning.

### Primary
- **Signal Vermilion** `#C9451F`: primary buttons, links that must stand out, the current step, the "为你推荐" mark, progress fill. White text on it passes AA.
- **Vermilion Deep** `#B23A15`: hover fill.
- **Vermilion Pressed** `#9C3010`: active fill and small text on soft vermilion.
- **Vermilion Soft** `#FBECE6` / **Vermilion Softer** `#FDF5F1`: recommendation-reason blocks, selected states, icon tiles.

### Paper neutrals
- **Paper Ground** `#F7F4EE`: page background.
- **Ticket White** `#FFFFFF`: cards, fields, header, tab bar.
- **Surface Quiet** `#FCFAF6`: inset groups and quiet cards.
- **Sunken** `#F1ECE3`: segmented control tracks, skeletons, quiet icon tiles.
- **Paper Hairline** `#E7E1D6`: card and section borders.
- **Hairline Strong** `#D4CABB`: field borders and hover borders.
- **Control Bar** `#8A8172`: 1px inset control stroke; meets 3:1 on paper.

### Ink
- **Station Ink** `#0B1B2B`: headings, primary text, brand mark, dark panels.
- **Slate Ink** `#4C5661`: body copy and secondary metadata.
- **Warm Muted** `#6B665C`: timestamps, counts, placeholders, helper text. It clears 4.5:1 on both paper and white.

### Semantic
- **Go Green** `#0E7C5A`: completed steps, success.
- **Caution Amber** `#A15C00`: warnings and article-type marks.
- **Halt Red** `#B3261E`: destructive actions and errors.
- **Info Blue** `#1D4ED8`: informational badges.
- **Level colors** `#0E7C7B` (1), `#1D4ED8` (2), `#A15C00` (3), `#BE123C` (4): only for knowledge levels and resource type/level marks. They are never used as page decoration.

### Named rules
**The One Signal Rule.** Vermilion means "act here" or "you are here." It may fill a button, mark the current step, tint a selected row, or color a recommendation reason. It never becomes a page wash or a gradient.

**The Meaningful Color Rule.** If a color does not encode level, type, or state, it is not in the palette. Use a neutral.

**The Dark Panel Rule.** Deep ink panels are orientation surfaces, not decoration: auth's brand panel, discipline and knowledge mastheads, and the learning-path header. Content pages stay on paper.

## Typography

**Latin & numerals:** self-hosted Manrope variable (`frontend/src/assets/fonts/manrope-latin-variable.woff2`).
**Chinese:** system stack — PingFang SC, Hiragino Sans GB, Microsoft YaHei; no Chinese webfont is loaded.
**Numbers:** tabular numerals (`.tnum`) for counts, ratings, percentages, dates, and levels.

### Hierarchy
- **Display** (800, 32px mobile → 52px desktop, 1.08, -0.035em): the home hero only. One line of voice, no kicker above it.
- **Headline** (750, 24px mobile → 30px desktop, 1.2, -0.03em): page titles.
- **Section** (750, 20px mobile → 24px desktop, 1.3, -0.025em): section headings.
- **Title** (700, 16px, 1.45): card titles.
- **Body** (400, 14px, 1.65): UI copy; lead paragraphs rise to 15–16px.
- **Small** (400, 13px): metadata and helper copy.
- **Micro** (650, 12px): chips, field labels, compact status text. No uppercase tracking as a default.

### Named rules
**The No-Eyebrow Rule.** Headings carry their own weight. Do not put a decorative kicker, uppercase label, or floating eyebrow above a heading. A code, level, or status chip is metadata and may sit beside a title, but not as a generic kicker.

**The Measure Rule.** Prose is constrained to 34–38rem; data tables may run wider. Card copy uses line clamping, never a tiny font.

## Layout

Content lives in a 1200px shell (`frontend/src/styles/index.css`, `.shell`) with 20px side padding on mobile and 32px from `sm`. Focused flows narrow: auth 420px, questionnaire 920px, profile 920px, learning path 1040px, upload 760px. Page rhythm is 36px / 64px on mobile and 52px / 88px on desktop.

The front shell is a 64px / 72px sticky white header with a 1px hairline, the spark wordmark, three nav items (首页 / 发现 / 知识点), a search field from `lg`, and an account menu. Active nav is a soft vermilion pill, not an underline. Below `md` the header keeps the wordmark, a search toggle, and the account action; the four primary destinations become a fixed bottom tab bar (首页 / 发现 / 知识点 / 我的) with a soft vermilion active capsule. Content reserves 66px at the bottom so the tab bar never covers the footer.

The admin shell uses a 248px light sidebar with the same wordmark, icon-led nav, a soft vermilion active pill, and an account block pinned to the bottom. The content header is a 64px white bar with a breadcrumb-like title; on mobile the sidebar becomes a horizontal chip row.

Resource grids are 1 column, 2 at `sm`, 3 at `lg`, with 20px gaps. Detail pages use a `1fr 340px` split at `lg`; the rating rail is sticky at `top: 96px`. The home hero is a `1.02fr / 0.98fr` split at `lg`; the right side is the live "下一步" card, not an abstract illustration.

Sections are separated by space, not rules. Section headers are a heading plus one line of quiet context; a right-aligned "查看全部" link uses the arrow-slide micro-interaction. Tables sit inside a 14px card; their header row is `Surface Quiet`, their dividers are hairlines, and their last row needs no border.

## Elevation & Depth

The system is flat at rest. Cards separate from paper by a 1px hairline and white fill, never by an ambient shadow.

There is one elevation vocabulary:
- **Hover lift** `0 2px 4px rgba(11,27,43,.04), 0 18px 40px -20px rgba(11,27,43,.28)` paired with `translateY(-2px)` on interactive cards.
- **Float** `0 1px 2px rgba(11,27,43,.05), 0 28px 60px -28px rgba(11,27,43,.32)` for the home hero's live card and the sticky questionnaire bar.

No shadows on headers, mastheads, tables, or static cards. A shadow is always ink-tinted, never neutral gray or a colored halo.

## Shapes

- Fields and buttons: 10px (large button 12px).
- Cards: 14px.
- Panels and dark mastheads: 16px.
- Chips, avatars, tab capsules, progress tracks: pill.
- Connectors in the knowledge graph: 1.4–1.8px curves with arrowheads.

Circles mean a state or a person: avatars, progress dots, step markers. They are not general decoration. Icons are authored 24px SVG strokes with round caps and a consistent 1.7px weight (`AppIcon.vue`); filled variants exist only for star/heart/spark.

## Components

### Buttons
40px default / 46px large / 32px small, 10–12px radius, 650 weight, 18px horizontal padding. Primary is vermilion with a 1px inner highlight; hover deepens and lifts 1px; active presses back. Secondary is white with a strong hairline; ghost is transparent with a sunken hover. Element Plus buttons are retuned to the same scale so there is one button vocabulary.

### Fields
42px height, 10px radius, white fill, 1px `Hairline Strong` inset stroke. Hover darkens to `Control Bar`; focus becomes 1px vermilion plus a 3px 14%-opacity vermilion ring. Placeholder is `Warm Muted` (4.5:1). Textareas share the treatment. Every field has a visible label or an `aria-label`; placeholders are not labels.

### Chips and tags
Pill, 12px/650, 3px 10px padding. Neutral chips use `Sunken`; state chips use semantic soft backgrounds (`chip-primary`, `chip-success`, `chip-warning`, `chip-danger`, `chip-info`). Status is always written out; color is never the only signal.

### Cards
White, 1px hairline, 14px radius, 20–24px internal padding. Interactive cards lift 2px and gain the hover shadow; their cover image scales to 1.03 inside the frame. Resource cards show type, category, rating, title, two lines of description, an optional vermilion recommendation-reason block, and a ruled footer with view count and a "查看" arrow.

### Dark panels
Ink fill, 16px radius, white type, a low-opacity orbit diagram in the corner (authored SVG geometry, not an illustration), and translucent white chips. They are used only for orientation: auth, discipline/knowledge mastheads, and the learning-path header.

### Navigation
Sticky white header with a 1px hairline; active nav is a soft vermilion pill. Account menu is an avatar pill with a chevron. The mobile tab bar is fixed, white, 1px top hairline, 68px tall, with a 44px active capsule around the icon and an 11px label. Admin sidebar is light, icon-led, with a soft active pill.

### Tables
14px card wrapper, `Surface Quiet` header, 12px/650 muted header labels, 13.5px body, 13px vertical padding, hairline dividers, transparent row backgrounds, and a `Surface Quiet` hover. Pagination is 32px buttons with a soft vermilion active state.

### Knowledge graph
White rounded node cards (10px), level-colored 4px dots, 12.5px bold labels, 10.5px metadata, and 1.4px curved prerequisite edges with arrowheads. The current node is ink-filled with a vermilion 1.6px stroke; its incoming edges turn vermilion. The graph is scrollable on desktop and scales to width on mobile.

### Path timeline
A 1px hairline rail with 32px state nodes: green check for done, vermilion play with a 4px soft ring for current, muted lock for locked. Step cards use a soft vermilion tint for the current step, quiet surface for done, white for locked. Progress is a 6px track with a vermilion fill; the header also shows the percent and `已完成 n / m 节`.

## Motion

Motion is state, not theater. Most transitions are 150–220ms with `cubic-bezier(.22, 1, .36, 1)`.

- Hover: cards lift 2px, covers zoom 1.03, arrow links slide 3px, buttons lift 1px.
- Page transition: 180ms fade plus a 6px rise; out is a 120ms fade.
- List transitions: 220ms fade/move.
- Progress bars: 500–700ms width transition.
- Skeletons: 1.6s shimmer.
- No hero choreography, parallax, scroll-jacking, or autoplay carousels. Data-driven cards may fade up on first render with a capped 35ms stagger.
- `prefers-reduced-motion: reduce` removes animation and transforms.

## Do's and Don'ts

### Do
- Do keep the split: warm paper for reading, white for interactive surfaces, deep ink for orientation.
- Do use one vermilion signal for action and current state; use level colors only where they encode level or type.
- Do use 10px fields, 14px cards, 16px panels, and pills only for chips/small controls.
- Do use Manrope for Latin/numerals and the system Chinese stack for Chinese; set data in tabular numerals.
- Do write one line of context under a section heading and leave generous space above it.
- Do provide a clear next action in every empty and error state.
- Do keep the mobile bottom tab bar, the sticky rating rail, and keyboard focus rings.
- Do use the spark mark, the live "下一步" card, and the orbit geometry as the only recurring brand devices.

### Don't
- Don't reintroduce the old transit language: navy rail bands everywhere, 2px route strokes, station dots, uppercase micro-labels, 4px corners.
- Don't add gradient text, glass decoration, or colored halos.
- Don't put a decorative kicker or uppercase eyebrow above a heading.
- Don't use a colored left border thicker than 1px on cards or alerts.
- Don't use pills for large containers or buttons; pills are for chips and small controls.
- Don't use ambient shadows on static cards; elevation is a hover response.
- Don't load a Chinese webfont; Chinese uses the system stack.
- Don't convey state by color alone; always write the label.
- Don't nest interactive elements; graph nodes are buttons inside a group, not inside a `role="img"`.

## Accessibility

- All body, placeholder, and helper text clears 4.5:1; large display text clears 3:1. The full page set is checked with axe-core at 1440, 768, and 390.
- Focus-visible is a 2px vermilion outline with 2px offset on every control and link.
- The skip link is the first focusable element in the front shell.
- Status is text + color (已完成 / 当前学习 / 未解锁; 已改 n; error copy).
- Every input/select has a visible label or an `aria-label`.
- The mobile tab bar uses `aria-current="page"`; the path progress bar has an accessible name.
- The knowledge graph is a `role="group"` with keyboard-selectable nodes.
- `prefers-reduced-motion` is honored.
