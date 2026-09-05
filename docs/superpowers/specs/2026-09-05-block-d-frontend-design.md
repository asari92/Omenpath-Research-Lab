# Block D — Frontend Design

**Status:** approved for detailed planning  
**Scope:** Stages 15–21 only  
**Date:** 2026-09-05

## 1. Authority and scope

This design is subordinate to [`00_FINAL_SPEC_v5.md`](../../../00_FINAL_SPEC_v5.md).
It defines presentation and frontend integration choices for Block D without
changing gameplay semantics completed in Stages 0–14. The implementation must
consume the existing REST, WebSocket, Tutorial, Event and persistence contracts.

Two narrow additive transport extensions support approved UI behavior without
changing gameplay rules:

1. Stage 15 exposes a machine-readable reason when a quick command is
   unavailable. This makes existing backend rules explainable without sending a
   rejected command merely to learn why it would fail.
2. Stage 18 adds authoritative current Observer presence counts to each Plane
   item used by the Extraction chooser. This avoids reconstructing current state
   from historical Events in the browser.

Block E and its delivery work remain out of scope.

## 2. Product experience

The application is an English-language real-time mission-control interface for
the Omenpath Research Lab. A new evaluator should be able to understand the Lab
state, identify the priority Portal and take a valid action in about one minute.

The visual direction combines a compact scientific control panel with restrained
arcane effects:

- dark, high-contrast control-room surfaces;
- teal system accents and semantic LOW/MEDIUM/HIGH/CRITICAL colors;
- strong information hierarchy before decoration;
- animated Portal artwork as the main visual identity;
- stable card geometry, so realtime changes never move controls unexpectedly.

The interface remains English to match the Final Spec, event names, transport
contract and evaluator vocabulary. Plans and engineering documentation may stay
in Russian.

## 3. Application shell and routes

The shared application shell contains:

- top bar: product name, Tutorial/Live mode, WebSocket state, latest-update age,
  and `OPEN EXTRACTION`;
- desktop side navigation: Dashboard, Event Log and AI Worklog;
- compact mobile bottom navigation with the same destinations;
- a toast region, confirmation modal host and Tutorial guidance layer;
- a small `ART CREDITS` entry that opens attribution and fan-content notices.

Routes:

| Route | Screen |
|---|---|
| `/` | Dashboard |
| `/portals/:id` | Portal Details |
| `/events` | Global Event Log |
| `/ai-worklog` | AI Worklog |
| unknown route | in-app Not Found with a return to Dashboard |

Tutorial is not a separate route. Its guidance panel follows the player through
Dashboard, matching Portal Details and Event Log, because those route changes are
part of the Tutorial itself.

## 4. Responsive Dashboard

### 4.1 Shared behavior

Dashboard contains the complete Final Spec summary, Needs Attention, Leyline
Override, Empty State and exactly seven stable Slot positions. Slots are keyed by
`slot_index`; they are never sorted by risk or Portal ID. Needs Attention links
to the matching Portal without moving its Slot.

Each occupied Slot always shows:

- Slot number, Portal name and destination Plane;
- Portal artwork;
- status, Stability, Energy, Time Remaining and Creatures;
- a `DETAILS` navigation control;
- exactly four gameplay commands: `STABILIZE`, `CLOSE`, `SEND OBSERVER` and
  `RECALL OBSERVER`.

`DETAILS` is navigation, not a fifth gameplay action. All four command controls
remain rendered in a fixed position. An unavailable command is visually muted
but remains keyboard-focusable and clickable with `aria-disabled="true"`; it
shows the backend-provided explanation and does not issue a POST.

An empty Slot keeps identical outer dimensions and reserved internal regions. It
shows `AWAITING ANOMALY`; the absent Portal artwork disappears, while the four
muted commands explain that no Portal occupies the Slot.

### 4.2 Desktop and tablet

Wide desktop uses two rows in a fixed `4 + 3` composition. The second row is
centered without renumbering positions. Portal artwork is large enough to be the
card focus. Tablet may use a `2 + 2 + 2 + 1` grid, but all seven Slots remain on
the Dashboard and no carousel or pagination is introduced.

### 4.3 Phone command board

Phone layout prioritizes simultaneous monitoring rather than desktop-sized
artwork. The Dashboard occupies `100dvh` and reserves the remaining height after
the top bar, compact two-row summary, Needs Attention banner and bottom nav for a
`2 × 4` Slot grid. Slot 7 is centered in the last row.

The grid and its rows use flexible height; Portal diameter, spacing, type and
button height scale with `clamp()` and viewport height. All seven Slot positions
and all four commands are visible at once without scrolling the Portal list.
Long names are truncated visually but remain available through accessible labels
and Details. Desktop Portal sizing is not reduced by these mobile rules.

## 5. Portal artwork and animation

### 5.1 Appearance

An open Portal is a large circular window into its destination Plane:

- destination art fills the clean inner circle;
- no ripple, concentric circle or rotating texture covers the image;
- particles are born on the circumference, receive tangential motion and an
  outward component, and fade after flying away from the edge;
- a selected or Needs Attention Portal uses the high-density profile;
- other OPEN Portals use the low-density profile;
- color is deterministically derived from Portal ID, so it is stable across
  renders and differs between Portal instances.

The desired physical principle follows the publicly visible p5.js
[Doctor Strange Portal example](https://codepen.io/vishalpatial/pen/bGbZWQd),
but the project will implement its own typed Canvas 2D renderer rather than copy
the pen or depend on p5.js.

### 5.2 Rendering boundary

Portal animation is isolated behind a `PortalEffect` component and a pure
particle-model module. All mounted Portal canvases share one scheduler instead of
creating one `requestAnimationFrame` loop per Slot.

Performance rules:

- high-density profile: at most 170 live particles;
- low-density profile: at most 42 live particles;
- device-pixel ratio is capped for the effect canvas;
- animation pauses outside the viewport and while the document is hidden;
- reduced-motion mode renders a static glow with no emitted particles;
- Canvas is decorative and has `aria-hidden="true"`; meaningful Plane text and
  status remain in HTML.

### 5.3 Local Plane asset pipeline

Runtime must not hotlink artwork or call an external art API. A one-time asset
preparation tool creates optimized local WebP files and a committed manifest:

```text
web/public/planes/<plane-id>-<slug>.webp
data/plane_images_manifest.json
```

Each manifest entry records Plane ID, local path, original source URL, source
object/card ID when available, artist/credit, license or usage-policy reference,
content hash and fallback status. A frontend helper resolves artwork strictly by
Plane ID. Every one of the 85 seeded Plane IDs must resolve either to a curated
local image or an explicit local fallback; a missing remote image can never break
Dashboard or Extraction.

Preferred discovery sources are official Wizards galleries and the Scryfall API.
Bulk preparation respects the [Scryfall API guidance](https://scryfall.com/docs/faqs/i-m-having-trouble-accessing-the-scryfall-api-or-i-m-blocked-17).
The shipped UI includes source/artist credits and an unofficial fan-content
notice consistent with the [Wizards Fan Content Policy](https://company.wizards.com/en/legal/fancontentpolicy).

## 6. Frontend architecture

### 6.1 Dependencies and module boundaries

The foundation uses React, TypeScript, Vite and React Router. Styling uses CSS
Modules plus a small set of global design tokens. A heavyweight component system,
Redux, Storybook and client-side gameplay engine are deliberately excluded.

Primary boundaries:

```text
web/src/
  app/          composition, routes, providers, shell
  api/          generated-by-hand transport types, REST and WebSocket clients
  state/        authoritative snapshot store and selectors
  features/     portal-actions, extraction, tutorial, event-filters
  components/   reusable presentation and feedback components
  pages/        Dashboard, PortalDetails, EventLog, AIWorklog
  portal-fx/    pure particles, shared scheduler, Canvas adapter
  assets/       manifest adapter and fallback selection
  styles/       tokens, reset and global layout rules
  test/         MSW, WebSocket fake, builders and render harness
```

Each unit has a single responsibility. Pages compose features; features call the
typed API; only the snapshot store owns server state. Components never import Go
domain implementation or reproduce domain eligibility rules.

### 6.2 State model

A small external typed store based on React's subscription API holds:

- latest authoritative `StateSnapshot`;
- bootstrap status;
- WebSocket connection state and reconnect attempt;
- last accepted `generated_at`;
- command-in-flight keys.

Ephemeral concerns such as an open modal, local filter text and expanded history
rows remain local to their feature. There is no optimistic mutation of Lab,
Portal, Observer or Plane state.

## 7. Data flow and errors

### 7.1 Snapshot lifecycle

On startup, the app requests `GET /api/state` and connects to `/ws/lab`. Every
REST command already returns a fresh `StateSnapshot`; WebSocket also sends an
initial snapshot, approximately one update per second and an immediate update
after significant actions.

All sources enter the same atomic `acceptSnapshot` function. A snapshot older
than the current `generated_at` is ignored, preventing a late bootstrap or POST
response from replacing newer realtime state. Equal timestamps are safe and
idempotent.

The WebSocket reconnects with bounded backoff and visible connection state. A
successful reconnect accepts the server's initial snapshot. The client does not
run a local one-second simulation or decrement authoritative values itself.

While Portal Details is open, accepted snapshots coalesce a refresh of
`GET /api/portals/{id}` so Risk, Recommendation and History remain current.
Event Log similarly refreshes its shared event source while visible. At most one
request of each kind may be in flight, with one trailing refresh if a newer
snapshot arrived meanwhile.

### 7.2 Additive quick-action reason contract

Existing `can_*` booleans remain backward compatible. `QuickActionsDTO` gains
nullable machine-readable reason fields:

```json
{
  "can_stabilize": false,
  "stabilize_unavailable_reason": "INSUFFICIENT_LAB_ENERGY",
  "can_close": true,
  "close_unavailable_reason": null,
  "can_send_observer": false,
  "send_observer_unavailable_reason": "PORTAL_CRITICAL_RISK",
  "can_recall_observer": false,
  "recall_observer_unavailable_reason": "NO_WAITING_OBSERVER"
}
```

The backend computes availability and the first deterministic reason through a
pure evaluator that reuses the same predicates and validation ordering as the
real command. It does not execute the command, mutate state, consume random or
create `ACTION_REJECTED`. A confirmable command is still available; if the actual
POST returns `CONFIRMATION_REQUIRED`, the normal confirmation flow runs.

The frontend maps known codes to English copy and has a safe generic fallback for
future codes. It never infers gameplay eligibility from Energy, Risk, direction,
creatures or Observer counts.

### 7.3 Command lifecycle

Only the selected Portal/action key is marked busy during a POST. Other commands
and realtime updates continue. On success, the returned snapshot is accepted
through the normal monotonic gate.

For `409` with `confirmable: true`, the UI opens a modal describing the exact
action and affected Portal. Confirm repeats the same endpoint with
`{"confirm":true}`. Cancellation has no side effect. Non-confirmable domain
errors become a toast and optional inline action message. Network errors preserve
the authoritative screen and offer retry. `401` handling is not invented because
authentication is outside the Final Spec.

### 7.4 Additive Plane presence contract

`PlaneDTO` keeps its existing fields and adds:

```json
{
  "observers_in_plane": 2,
  "observers_waiting_return": 1
}
```

`observers_in_plane` counts Observers whose current Plane ID matches the Plane,
including EXPLORING, WAITING_RETURN and RETURNING until a successful return.
`observers_waiting_return` is the actionable subset used to explain Extraction
choices. Both values are derived from the same resolved snapshot as the rest of
`StateSnapshot`; they are not persisted columns and do not alter observer
lifecycle. The Extraction chooser uses these fields for badges and the
`OBSERVER PRESENT` filter.

## 8. Screens and features by stage

### Stage 15 — Frontend foundation

Create the frontend workspace, route shell, typed clients, snapshot store,
feedback infrastructure, test harness, design tokens, artwork pipeline and Canvas
boundary. Complete the additive backend unavailable-reason corrective pass before
Dashboard consumes it.

### Stage 16 — Dashboard

Implement the complete summary, Energy and Override presentation, Needs
Attention navigation, seven fixed Slots, responsive desktop/mobile command board,
Empty State and all four quick commands.

### Stage 17 — Portal Details

Show the large Portal, all Portal and Destination fields required by the Final
Spec, Risk, Recommendation, expandable `How Risk Works`, shared four-command
controls and Portal-filtered History timeline. Opening the current Tutorial target
sends the explicit matching Tutorial signal; GET alone never advances Tutorial.

### Stage 18 — Extraction, confirmations and errors

Extraction opens as a focused modal or drawer over the current screen. The 85
Plane choices use the same artwork resolver and support search plus `ALL`,
`UNEXPLORED`, `EXPLORED` and `OBSERVER PRESENT` filters. A card shows Plane name,
exploration state and Observer presence. Selection clearly shows the 30 Energy
cost before submission. Backend state and errors remain authoritative.

Before the chooser consumes Observer badges, Stage 18 performs the additive
`PlaneDTO` presence pass described in §7.4 and proves that the derived counts use
the same snapshot timestamp as the rest of the response.

This stage completes reusable confirmation, toast, inline, reconnect and not-found
states across Dashboard and Details.

### Stage 19 — Event Log

Render the complete chronological source from `GET /api/events`. Filters cover
event type, Portal, Observer and Plane without altering order. Each row shows its
message and identifiers; `payload_json` is an opt-in technical disclosure. Portal
History and Event Log share event formatting primitives.

### Stage 20 — Tutorial UI

The persistent guidance panel renders authoritative `step`, `phase`, target IDs
and `expected_action`. Copy follows Final Spec §28.2 exactly in meaning: general
orientation at Step 0, then prices and rules only immediately before the relevant
action. The panel highlights the target Slot or destination screen, displays
waiting phases without fake progress, follows recreated target IDs, handles LOST
retry and exposes Reset deliberately. Step 9 starts Live and preserves continuity.

Only explicit user actions send Tutorial signals. Ordinary GET and route rendering
do not mutate Tutorial state.

### Stage 21 — AI Worklog

Render a readable application view derived at build time from
[`01_AI_WORKLOG_CURRENT.md`](../../../01_AI_WORKLOG_CURRENT.md). The Markdown file
remains the single maintained content source. The page provides the Final Spec
categories: tools, time, available token usage, stages, developer/AI contribution,
key prompts, decisions, AI mistakes, manual rewrites, verification and future
improvements. The build must fail clearly if the source cannot be loaded.

## 9. Test strategy

Frontend unit and integration tests use Vitest, React Testing Library,
`user-event`, MSW and a controlled WebSocket fake. Playwright covers browser
flows and actual responsive layout.

Required coverage includes:

- bootstrap, newest-snapshot-wins, duplicate snapshot idempotence and reconnect;
- no optimistic gameplay changes;
- all seven Slot positions and Empty State;
- all four command controls and unavailable-reason behavior;
- confirmation retry with exactly one `confirm:true` POST;
- Details live refresh, Risk/Recommendation and shared History formatting;
- Extraction search/filter/selection/error behavior;
- Event Log filters without order mutation;
- every Tutorial step, explicit signals, recreated target and Live transition;
- AI Worklog source rendering;
- particle spawn radius, outward/tangential movement, population caps, shared
  scheduler lifecycle, visibility pause and reduced-motion fallback;
- Playwright desktop and phone critical paths;
- a phone layout assertion that all seven Slot boxes lie inside the available
  Dashboard control-board viewport and the Portal list has no vertical overflow.

Each Stage 15–21 follows stage-level RED/GREEN commits. Stage completion requires
frontend format/lint, TypeScript checking, frontend tests/build and the repository
Go quality commands required by `AGENTS.md`. The Block D boundary repeats all
frontend tests and Playwright flows plus the full Go suite and race detector.

## 10. Explicit exclusions

Block D does not add authentication, multiplayer, sound, WebGL, a client-side
simulation, pagination, Storybook, general localization, user-selectable themes,
new Recommendation semantics or any Stage 22–27 delivery work.

## 11. Acceptance summary

Block D is complete only when the evaluator can use every required screen and
primary action without a broken flow; all seven Slots remain simultaneously
monitorable on the phone command board; Portal art and effects work without
runtime network dependencies; inaccessible actions explain themselves without
creating rejected domain events; Tutorial reaches Live; and Stage 21 exposes an
honest AI Worklog view.
