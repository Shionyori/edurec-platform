---
name: edurec
description: 学习线路牌 — a transit route-map design system for finding the next thing to learn.
colors:
  primary: "#C9451F"
  primary-hover: "#B93C18"
  primary-active: "#A32F10"
  surface: "#FFFFFF"
  paper-bg: "#F5F4EF"
  border: "#DCD9CF"
  control-border: "#8C8777"
  ink: "#0B1B2B"
  ink-secondary: "#4A5763"
  ink-muted: "#6B7681"
  rail: "#0B1B2B"
  rail-soft: "#14293D"
  line-teal: "#0E7C7B"
  line-blue: "#1D4ED8"
  line-amber: "#B45309"
  line-rose: "#BE123C"
  success: "#15803D"
  warning: "#B45309"
  danger: "#B91C1C"
typography:
  display:
    fontFamily: 'system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "48px"
    fontWeight: 700
    lineHeight: 1.1
    letterSpacing: "-0.025em"
  headline:
    fontFamily: 'system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "30px"
    fontWeight: 700
    lineHeight: 1.2
    letterSpacing: "-0.025em"
  title:
    fontFamily: 'system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "20px"
    fontWeight: 700
    lineHeight: 1.4
    letterSpacing: "0em"
  body:
    fontFamily: 'system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.625
    letterSpacing: "0em"
  label:
    fontFamily: 'system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "11px"
    fontWeight: 700
    lineHeight: 1
    letterSpacing: "0.14em"
    fontFeature: "tnum"
  brand-label:
    fontFamily: 'system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "10px"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "0.18em"
  control:
    fontFamily: 'system-ui, -apple-system, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    fontSize: "14px"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "0em"
rounded:
  sm: "3px"
  md: "4px"
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
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.surface}"
    typography: "{typography.control}"
    rounded: "{rounded.md}"
    padding: "8px 15px"
    height: "32px"
  button-primary-hover:
    backgroundColor: "{colors.primary-hover}"
  button-primary-active:
    backgroundColor: "{colors.primary-active}"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.control}"
    rounded: "{rounded.md}"
    padding: "1px 11px"
    height: "32px"
  nav-link:
    backgroundColor: "{colors.rail}"
    textColor: "{colors.surface}"
    typography: "{typography.control}"
    padding: "20px 12px"
  line-chip:
    backgroundColor: "{colors.rail}"
    textColor: "{colors.surface}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "3px 7px"
  resource-card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "20px"
  route-strip:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    padding: "14px 0 0"
  sticky-action-bar:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "12px 20px"
---

# Design System: edurec

## Overview

**Creative North Star: "学习线路牌 / The Study Line"**

edurec is a Chinese educational-resource recommender, and its visual world is the transit route map: knowledge points are stations on a prerequisite line, the current goal is "你在这里," and a learning path is a printed itinerary. The system operates in **Operate** mode — the route metaphor contributes palette, type, density, and one signature move (the route strip and station line); layout, controls, and form semantics stay conventional so the user can act quickly. The result reads as a rail system on paper, not as a generic white SaaS dashboard.

The ground is a cool off-white paper, the orientation surfaces are deep station navy, and one signal vermilion marks the current stop or the action. Line colors are semantic, not decorative: they encode level and resource type wherever a line, chip, or station appears. Density is deliberately tight: pages are organized by section rules and 20px card gaps rather than by generous whitespace or oversized type.

Chinese is set in the system sans stack — no webfont is loaded — and codes, counts, levels, percentages, and ratings use tabular numerals. The shape language is small and exact: 4px cards, 3px chips and controls, 2px route strokes, and round station dots as the only recurring circles.

**Key Characteristics:**
- Deep navy rail surfaces for orientation and mastheads; cool paper for reading.
- Signal vermilion reserved for the primary action and the current station.
- Four semantic line colors (teal, blue, amber, rose) mapped to levels and resource types.
- System Chinese sans with tabular numerals and 11px uppercase micro-labels.
- 4px / 3px / 2px geometry, flat by default, one ink-tinted hover lift.
- The route strip and station line as the one recurring custom component.

## Colors

A rail-signage palette: one warm signal color against a cool paper neutral field, with four line colors carrying all structural meaning.

### Primary
- **Signal Vermilion** (#C9451F): the action and current-state signal — primary buttons, the active nav underline, the current learning station, path progress fill, and the "已修改" change markers. White text on it passes AA.
- **Signal Vermilion Deep** (#B93C18): hover fill for primary buttons.
- **Signal Vermilion Pressed** (#A32F10): pressed fill, and the text color for primary-tinted status chips on white.

### Secondary
- **Teal Line** (#0E7C7B): Level 1 stations, course-type chips, recommendation-reason text, and the global focus ring.
- **Cobalt Line** (#1D4ED8): Level 2 stations and video-type chips.
- **Amber Line** (#B45309): Level 3 stations, article-type chips, cold-start chips, and in-progress crawler notices.
- **Rose Line** (#BE123C): Level 4 stations and the admin "分类管理" rail marker.

### Tertiary
- **Go Green** (#15803D): completed path stations, completed progress, and success notices.
- **Caution Amber** (#B45309): warning states; shares the amber hue with the line family because level 3 and caution are both "attention" in this world.
- **Halt Red** (#B91C1C): errors and destructive admin actions.

### Neutral
- **Station Navy** (#0B1B2B): primary ink and the rail/header surface. Body text and the top navigation are the same navy by design.
- **Depot Navy** (#14293D): the soft rail step for raised rail panels.
- **Paper Ground** (#F5F4EF): page background behind every content section.
- **Ticket White** (#FFFFFF): cards, inputs, sticky bars, and the content surface.
- **Slate Ink** (#4A5763): secondary copy, descriptions, metadata, and muted nav links.
- **Muted Slate** (#6B7681): timestamps, counts, placeholders, and micro-labels. It is darker than a typical muted gray so 11px text still clears 4.5:1.
- **Paper Hairline** (#DCD9CF): default card and section borders, dividers, and route tracks at rest.
- **Control Bar** (#8C8777): 1px inset stroke around inputs, selects, and segmented controls; below 3:1 would make controls disappear on paper.

### Named Rules
**The One Signal Rule.** Vermilion is the "you are here / act now" signal. It may fill a primary action, mark the current station, underline the active nav item, or tint a current-state chip at 10%; it never becomes a section wash, a gradient, or decorative chrome.

**The Semantic Line Rule.** Every line color must encode a real level (1–4) or resource type (course/video/article). The four-color palette is never used as decorative variety or assigned at random.

## Typography

**Display Font:** system-ui with -apple-system, "Segoe UI", Roboto, and the Chinese stack "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei" (no webfont)
**Body Font:** the same system stack
**Label/Mono Font:** the same stack; data uses tabular numerals rather than a separate monospace face

**Character:** Neutral, legible Chinese at small sizes, with the personality coming from weight and tracking rather than from a display face. Headlines are bold and tightly tracked; data is compact and aligned; micro-labels read like signage.

### Hierarchy
- **Display** (700, 30px mobile → 48px desktop, 1.1, -0.025em): the home hero only — "下一站，学什么？"
- **Headline** (700, 24px mobile → 30px desktop, 1.2, -0.025em): page titles and navy masthead titles.
- **Title** (700, 20px, 1.4, 0em): section headers such as "本站推荐" and "知识点线路"; card titles step down to 16px semibold.
- **Body** (400, 14px, 1.625): descriptions, card copy, and form labels; hero lead copy rises to 16px. Long prose is constrained to max-w-2xl / max-w-3xl rather than spanning the full grid.
- **Label** (700, 11px, line-height 1, 0.14em, uppercase, tabular): chip text, level badges, and compact data labels.
- **Brand label** (600, 10px, 0.18em, uppercase): the "学习线路" / "管理后台" lockup under the wordmark.
- **Control** (600, 14px, line-height 1): buttons, nav links, and form controls.

### Named Rules
**The Tabular Data Rule.** Every number that can be compared — counts, percentages, ratings, levels, dates, codes — uses `font-variant-numeric: tabular-nums`. Columns of stats must align without a monospace font.

**The Micro-Label Rule.** The 11px uppercase 0.14em label is a data/signage device: it belongs inside a chip or a metric row. It is never a decorative kicker or eyebrow floating above a heading, and never body copy.

## Layout

Content lives in a centered container capped at 1280px (`max-w-7xl`) with 16px side padding on mobile and 24px from the `sm` breakpoint, and 32px / 40px vertical rhythm on mobile / desktop. Focused flows narrow further: profile and learning-path pages cap at 1024px, the questionnaire at 896px, and auth at 448px.

The front shell is a 64px sticky navy rail containing the brand lockup, a search field, six primary nav links, and the account menu. Below `md` (768px) the main nav collapses into a horizontally scrollable row of 3px-radius chips beneath the rail, so navigation is never simply removed. The admin shell swaps the top rail for a 240px navy sidebar on desktop, with each section marked by a colored station dot and the same horizontal chip row on mobile.

Resource grids are 1 column, 2 at `sm` (640px), and 3 at `lg` (1024px), with 20px gaps; they never exceed three columns. Every section is introduced by a header sitting directly on the paper with a 2px line-colored bottom rule and a station node — teal for discovery, blue for resources, rose for concept layers — not by a card inside a card. The home page composes a white hero panel, an optional cold-start interest picker, and then the recommendation grid; detail pages use a navy masthead followed by ruled content sections.

The paper ground stays plain: route language appears only where it carries structure (section rules, station nodes, route strips), never as a background texture behind content. Interactive cards can repeat the route cue as a 2px inset top rule (a positioned element inside the card, not a border), keeping the corner language intact. The footer closes the paper with a five-segment line-color rule.

Two sticky regions are intentional: the top rail (64px, `z-40`) and, on the questionnaire, a bottom action bar that stays 16px above the viewport edge while the user scrolls a long self-assessment. The search filter panel sticks below the rail at `top: 64px` on desktop.

### Named Rules
**The Rail-and-Paper Rule.** Orientation lives on deep navy rail — top bar, mastheads, admin sidebar; reading lives on cool paper. A page never becomes a full-width navy field, and a masthead never floats as a white card.

**The Three-Column Cap Rule.** Resource and station grids cap at three columns; density comes from tighter type and 20px gaps, never from adding a fourth column.

## Elevation & Depth

The system is flat by default. Navy rail surfaces and white cards separate by fill and 1px paper hairlines, not by shadow. There is exactly one shadow vocabulary: a soft, ink-tinted, upward-offset hover lift for interactive cards, reused under the sticky action bar to detach it from scrolling content. Large mastheads, section blocks, and the admin sidebar carry no shadow.

### Shadow Vocabulary
- **Card Hover Lift** (`box-shadow: 0 10px 30px -18px rgba(11, 27, 43, 0.45)`): resource cards, discipline cards, and the sticky action bar, always paired with a 2px rise.
- **Point Card Lift** (`box-shadow: 0 10px 24px -18px rgba(11, 27, 43, 0.45)`): the tighter variant on knowledge-point cards inside a discipline page.

### Named Rules
**The Flat-By-Default Rule.** Surfaces are flat at rest. A shadow appears only as a response to interaction (card hover, translateY(-2px)) or to detach the one sticky action bar; never as ambient decoration, and never in the generic soft-gray SaaS style — the shadow is always tinted with Station Navy.

## Shapes

Cards are 4px-rounded rectangles; chips and controls are 3px; route strokes are 2px. A deliberate small radius keeps the interface closer to printed signage than to soft consumer SaaS. The only recurring circles are station dots (10px outline nodes, 12px strip dots), avatars, and progress tracks — circles mean "a stop on the line," not general decoration. Icons are inline SVG strokes with square line caps, which echoes the squared ends of route tracks.

### Named Rules
**The Square-End Rule.** Use 4px for cards, 3px for chips and controls, 2px for route strokes, and square-capped connector strokes. Do not introduce 8px+ radii, pill buttons, or large blobs; circles are reserved for stations, avatars, and progress.

## Components

### Buttons
- **Shape:** 4px radius (24px-small buttons use 3px), 32px default height / 40px large, 8px 15px default padding / 12px 19px large, 600 weight.
- **Primary:** Signal Vermilion fill with white text; the default action on every page ("选一个学习目标," "保存并返回知识点," "生成我的学习路径").
- **Hover / Focus:** hover fill Signal Vermilion Deep, active fill Signal Vermilion Pressed; transitions run ~150ms. The global `:focus-visible` treatment is a 2px Teal Line outline with 2px offset, applied to every control and link.
- **Secondary / Ghost:** default Element Plus buttons stay neutral on Ticket White with a hairline border; text buttons appear in admin headers. Element Plus status variants (warning for favorite state, danger for destructive admin actions) retain Element's default status colors and are not part of the branded palette.

### Chips
- **Style:** `line-chip` — 3px radius, 3px 7px padding, 11px/700 uppercase with 0.06em tracking, white text on a semantic fill passed through a `--chip` custom property (teal for course, blue for video, amber for article, teal/blue/amber/rose for levels 1–4).
- **State:** The outline variant is transparent with a 1px inset currentColor stroke, used for codes on navy mastheads. The questionnaire answer scale is a custom 3px segmented control inside a 1px Control Bar border; the interest picker uses Element Plus checkbox buttons. Status is always paired with a text label ("已改 3," "当前学习," "已全部完成"), never color alone.

### Cards / Containers
- **Corner Style:** 4px on every card, panel, and masthead.
- **Background:** Ticket White for content; Station Navy for mastheads, the learning-path header, and admin sidebar; Paper Ground for empty states and inset groups.
- **Shadow Strategy:** flat at rest; Card Hover Lift on interactive cards only.
- **Border:** 1px Paper Hairline; dashed Paper Hairline for empty states, which are action entry points rather than dead ends.
- **Internal Padding:** 16–20px for card bodies (resource cards use 16px mobile / 20px desktop), 20–28px for panels and mastheads. Resource cards put a 16:9 cover flush to the top edge, with a type chip and category row above the title, a teal dot-and-reason line for recommendation explanations, and a ruled footer of rating and views using tabular numerals.

### Inputs / Fields
- **Style:** 32px height, 4px radius, Ticket White fill, 1px inset Control Bar stroke, 14px text, Muted Slate placeholder; selects and textareas share the treatment.
- **Focus:** the inset stroke shifts from Control Bar to Station Navy. The global keyboard focus ring is separate: 2px Teal Line, 2px offset.
- **Rail variant:** inputs placed on the navy rail use a white 8% fill, 16% inset stroke, and white text; focus raises the stroke to 55% white and the placeholder to 50% white. Search in the top rail uses this variant.

### Navigation
- **Style:** 64px Station Navy rail, sticky at the top, with a white 3px-radius square holding the route-mark logo, a 17px bold wordmark, and a 10px tracked subtitle. Links are 14px Control weight.
- **States:** active link is white semibold with a 2px Signal Vermilion bottom border; inactive links are white at 65% and brighten on hover. Account menu shows a 28px avatar ringed in 25% white.
- **Mobile:** below 768px the rail keeps the brand, collapses search, and exposes nav as a horizontally scrollable chip row (3px radius, white/12 fill when active) on a 10% white top border.
- **Admin:** a 240px navy sidebar with a colored `route-node` beside each item (filled for the active section) and a white 10% active row; content header is a 64px white bar with a ruled bottom border.

### Route Strip / Station Line (signature)
The one custom visual move. A horizontal `route-strip` lays stops along a 2px track: each stop has a 12px round dot with a 3px border, a 12px label, and state encoded on the dot — done fills Go Green, current fills Signal Vermilion with an 18% halo, locked stays outline and mutes its label. The home hero uses a three-stop strip ("选目标 / 生成路径 / 完成本节") and the hero has a decorative track with two outline stations and one filled vermilion station. Vertical station lines appear on learning-path and discipline pages: a 2px track runs down the left edge, 10px `route-node` dots (outline unless unlocked) sit in the gutter, and current rows get a 40% vermilion border and 4% tint. In the knowledge graph, the same four line colors encode level and the current node is emphasized. Text labels always accompany state color.

### Sticky Action Bar
A white 4px-rounded bar, 1px Paper Hairline, 12px 20px padding, pinned `sticky; bottom: 16px` on the questionnaire. The left side carries a tabular count ("共 72 个知识点 · 已修改 3 项") and the right side the primary save action. It reuses Card Hover Lift to float over the long scrolling list and remains visible on mobile, where the count and button wrap into one compact row.

## Do's and Don'ts

### Do:
- **Do** keep the rail/paper split: navy for orientation and mastheads, paper for reading, white for interactive content surfaces.
- **Do** use 4px cards, 3px chips and controls, 2px route strokes, and square-capped connector strokes.
- **Do** map the four line colors to real levels or resource types, and pair every state color with a text label.
- **Do** use Signal Vermilion only for the primary action, the current station, the active nav underline, and small current-state tints.
- **Do** use tabular numerals for every count, rating, percentage, level, and code.
- **Do** set Chinese in the system sans stack and keep micro-labels at 11px uppercase with 0.14em tracking inside chips and data rows.
- **Do** provide a clear next action inside every empty state, using a dashed hairline container rather than a bare blank region.
- **Do** introduce sections with a 2px line-colored rule and a station node; use teal for discovery, blue for resources, rose for concept layers.
- **Do** repeat the line cue as an inset 2px top rule on discipline cards, and keep the paper ground plain so route lines always mean structure.

### Don't:
- **Don't** introduce large radii (8px+), pill buttons, gradients, or soft gray ambient shadows; the only elevation is the ink-tinted hover lift.
- **Don't** use the line palette as decoration or assign level/type colors arbitrarily; if a color does not encode level or type, it is not a line color.
- **Don't** use vermilion as a section background, a wide wash, or a generic accent.
- **Don't** use the 11px uppercase micro-label as a decorative kicker/eyebrow above headings.
- **Don't** convey done/current/locked or error/success states by color alone; carry the label.
- **Don't** load a webfont or switch Chinese body copy to a serif/display face.
- **Don't** reintroduce a decorative route texture behind content; lines and station dots must encode structure or state.
- **Don't** use a thick accent border on a rounded card; inset the 2px rule as its own positioned element instead.
