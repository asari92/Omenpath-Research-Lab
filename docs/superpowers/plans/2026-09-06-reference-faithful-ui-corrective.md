# Reference-Faithful UI Corrective Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** make the real React UI match the approved magical-laboratory reference while removing Tutorial route traps and preserving every existing gameplay/API contract.

**Architecture:** keep the current React routes, SnapshotProvider, command hooks and Portal canvas. Add a local decorative background plate, rebuild the shared shell/card styling around semantic HTML, make the occupied Portal card itself the Details affordance, and add authoritative post-response navigation for the two Tutorial route transitions. No backend or dependency change is required.

**Tech Stack:** React 19, React Router 7, TypeScript 6, CSS Modules, Vitest/Testing Library, existing Canvas Portal effect, Vite, Docker.

**Approved design:** [`docs/superpowers/specs/2026-09-06-reference-faithful-ui-corrective-design.md`](../specs/2026-09-06-reference-faithful-ui-corrective-design.md)

**Execution constraint:** run inline and sequentially; do not dispatch subagents.

---

## File map

| File | Responsibility |
|---|---|
| `web/public/ui/lab-shell-background.png` | text-free/cardless full-viewport ornament plate derived from the approved reference |
| `web/src/styles/tokens.css` | obsidian, antique-gold, parchment and semantic-value tokens |
| `web/src/styles/global.css` | full-page base, accessible helpers and page-header notifications |
| `web/src/app/AppShell.tsx` / `.module.css` | single continuous laboratory shell, integrated left summary/navigation and ornaments |
| `web/src/components/dashboard/LabSummary.tsx` / `.module.css` | arcane Energy gauge, 20 Observer markers, Override and compact secondary values |
| `web/src/components/dashboard/PortalSlot.tsx` / `.module.css` | full-card Details navigation, reference card hierarchy and semantic metrics |
| `web/src/features/portal-actions/PortalActions.tsx` / `.module.css` | four-button row and notification feedback without fixed bottom overlays |
| `web/src/features/tutorial/TutorialPanel.tsx` / `.module.css` | parchment overlay and authoritative Live-to-Dashboard navigation |
| `web/src/features/tutorial/tutorial-copy.ts` | Step 1 Portal-card instruction |
| `web/src/pages/PortalDetailsPage.tsx` / `.module.css` | authoritative Details-to-Dashboard navigation and one-viewport composition |
| `web/src/components/portal/{PortalFacts,ObserverTransit}.tsx` | semantic value attributes for styling and accessibility |
| `web/src/pages/{DashboardPage,EventLogPage,AIWorklogPage}.module.css` | route-specific placement inside the shared visual system |
| `web/src/**/*.test.*` | regression evidence for interaction, copy and route choreography |
| `00_FINAL_SPEC_v5.md`, `01_AI_WORKLOG_CURRENT.md`, `docs/traceability.md` | approved UI wording and honest evidence |

## Task 1 — Capture interaction and Tutorial regressions in RED

**Files:**

- Modify: `web/src/pages/DashboardPage.test.tsx`
- Modify: `web/src/pages/PortalDetailsPage.test.tsx`
- Modify: `web/src/features/tutorial/TutorialPanel.test.tsx`
- Modify: `web/src/features/tutorial/tutorial-copy.test.ts`
- Modify: `web/src/components/feedback/FeedbackProvider.test.tsx`
- Modify: `web/src/pages/HelpPage.test.tsx`

- [ ] **Step 1: add card-navigation assertions**

Extend the Dashboard harness with routes for `/` and `/portals/:id` plus a
location probe. Add these assertions:

```tsx
const card = screen.getByRole("link", {
  name: /inspect Omenpath #0001 on Alara/i,
});
expect(screen.queryByRole("link", { name: "Details" })).toBeNull();
await user.click(card);
expect(screen.getByTestId("location")).toHaveTextContent("/portals/1");
```

Add separate keyboard cases for `Enter` and `Space`. Add a nested-command case
that clicks `Stabilize`, expects the command API once, and asserts the route is
still `/`.

- [ ] **Step 2: add authoritative route-transition assertions**

In `PortalDetailsPage.test.tsx`, make the matching Tutorial signal resolve to
Step 2 and expose `/` as `Dashboard route`. Assert that successful
`PORTAL_DETAILS_OPENED` navigation reaches `/`, while a rejected signal remains
on `/portals/42`.

In `TutorialPanel.test.tsx`, render the panel at `/events` with a location
probe. Assert that `Start Live` stays on `/events` until the promise resolves to
an authoritative `mode: "LIVE"` snapshot, then reaches `/`.

- [ ] **Step 3: add copy and notification-placement assertions**

Require Step 1 to contain exactly:

```text
Select the highlighted Portal on the Dashboard to inspect it.
```

Require the Step 1/Help copy to omit `Open Details` and `Details button`.
Require the notification region to carry:

```tsx
expect(region).toHaveAttribute("data-placement", "page-header");
```

- [ ] **Step 4: run focused tests and preserve RED**

Run:

```bash
npm --prefix web test -- --run \
  src/pages/DashboardPage.test.tsx \
  src/pages/PortalDetailsPage.test.tsx \
  src/features/tutorial/TutorialPanel.test.tsx \
  src/features/tutorial/tutorial-copy.test.ts \
  src/components/feedback/FeedbackProvider.test.tsx \
  src/pages/HelpPage.test.tsx
```

Expected: failures for the still-present Details link, absent card link,
unchanged Step 1 copy, absent route redirects and absent placement marker.

- [ ] **Step 5: commit RED**

```bash
git add web/src/pages/DashboardPage.test.tsx \
  web/src/pages/PortalDetailsPage.test.tsx \
  web/src/features/tutorial/TutorialPanel.test.tsx \
  web/src/features/tutorial/tutorial-copy.test.ts \
  web/src/components/feedback/FeedbackProvider.test.tsx \
  web/src/pages/HelpPage.test.tsx
git commit -m "test(ui-corrective): RED reference interaction and tutorial flow"
```

## Task 2 — Implement card navigation, Tutorial redirects and header feedback

**Files:**

- Modify: `web/src/components/dashboard/PortalSlot.tsx`
- Modify: `web/src/components/dashboard/PortalSlot.module.css`
- Modify: `web/src/pages/PortalDetailsPage.tsx`
- Modify: `web/src/features/tutorial/TutorialPanel.tsx`
- Modify: `web/src/features/tutorial/tutorial-copy.ts`
- Modify: `web/src/pages/HelpPage.tsx`
- Modify: `web/src/components/feedback/ToastRegion.tsx`
- Modify: `web/src/features/portal-actions/PortalActions.tsx`
- Modify: `web/src/features/portal-actions/PortalActions.module.css`
- Modify: `web/src/styles/global.css`

- [ ] **Step 1: replace Details with a stretched semantic link**

Inside each non-ghost occupied `PortalSlot`, render an absolute sibling link,
not an interactive wrapper around buttons:

```tsx
<Link
  aria-label={`Inspect ${portal.name} on ${portal.destination_plane_name}`}
  className={styles.cardLink}
  onClick={() => recordNavigationIntent({ kind: "portal", id: portal.id })}
  to={`/portals/${portal.id}`}
/>
```

Give card content `pointer-events: none`; give `.actions` and any unavailable
reason controls `pointer-events: auto; position: relative; z-index: 2`. A ghost
or empty Slot renders no card link. Remove the standalone `Details` link.

- [ ] **Step 2: navigate only after authoritative Tutorial responses**

In `PortalDetailsPage`, use `useNavigate`. After a matching signal resolves:

```tsx
const next = await api.tutorialSignal(request);
store.acceptSnapshot(next);
if (next.app.expected_action !== "OPEN_PORTAL_DETAILS") navigate("/");
```

Do not navigate from the error path. In `TutorialPanel`, after `startLive()`:

```tsx
const next = await api.startLive();
store.acceptSnapshot(next);
if (next.app.mode === "LIVE") navigate("/");
```

- [ ] **Step 3: update player-facing instructions**

Set Step 1 instruction to the approved exact sentence and change Help Step 1 to
describe selecting the highlighted Portal card. Do not reorder server Tutorial
steps or change completion signals.

- [ ] **Step 4: move every non-modal message to the page header**

Add `data-placement="page-header"` to `ToastRegion`. Change its CSS from
bottom-fixed to the upper-right of the main content header. Remove the fixed
`PortalActions.module.css .wrapper p` rule. Send unavailable/offline reason text
through `useFeedback().notify(...)` from `PortalActions`, so there is one
bounded notification surface rather than competing overlays.

- [ ] **Step 5: run focused tests to GREEN**

Run the Task 1 command. Expected: all selected files PASS.

- [ ] **Step 6: commit behavioural GREEN**

```bash
git add web/src/components/dashboard/PortalSlot.tsx \
  web/src/components/dashboard/PortalSlot.module.css \
  web/src/pages/PortalDetailsPage.tsx \
  web/src/features/tutorial/TutorialPanel.tsx \
  web/src/features/tutorial/tutorial-copy.ts web/src/pages/HelpPage.tsx \
  web/src/components/feedback/ToastRegion.tsx \
  web/src/features/portal-actions/PortalActions.tsx \
  web/src/features/portal-actions/PortalActions.module.css \
  web/src/styles/global.css
git commit -m "fix(ui-corrective): keep tutorial actions in view"
```

## Task 3 — Create the reference background and rebuild the shared shell

**Files:**

- Create: `web/public/ui/lab-shell-background.png`
- Modify: `web/src/styles/tokens.css`
- Modify: `web/src/styles/global.css`
- Modify: `web/src/app/AppShell.tsx`
- Modify: `web/src/app/AppShell.module.css`
- Modify: `web/src/components/dashboard/LabSummary.tsx`
- Modify: `web/src/components/dashboard/LabSummary.module.css`
- Test: `web/src/app/App.test.tsx`
- Test: `web/src/pages/DashboardPage.test.tsx`

- [ ] **Step 1: add structural RED assertions**

Require the shell to expose `data-visual-theme="arcane-laboratory"`, the Energy
gauge to expose `data-testid="lab-energy-gauge"`, exactly 20 Observer markers,
and the integrated summary to include Override, exploration and Portal totals.
Run the two test files and confirm the missing markers fail.

- [ ] **Step 2: generate one text-free decorative plate**

Use the approved user image as the image-generation reference with this exact
constraint:

```text
Create a 16:9 empty fantasy arcane laboratory interface background matching the
reference: blackened stone and parchment, thin antique-gold filigree frame,
subtle runic circles and compass geometry, small candle clusters in the lower
corners. Remove every card, portal, panel, button, label, number, logo and text.
Keep the center and left information areas dark and low-contrast so real UI can
be overlaid. No readable symbols, no watermark, no science-fiction elements.
```

Save the generated local asset as
`web/public/ui/lab-shell-background.png`. Inspect it at original resolution;
reject output containing pseudo-text, cards or Portal circles.

- [ ] **Step 3: implement the shared visual tokens and shell**

Define distinct tokens for background, panel glass, antique gold, parchment,
Energy, Time, transit, success, LOW, MEDIUM, HIGH and CRITICAL. Apply the local
background plate with a dark overlay in `.shell`, add a CSS fallback frame, and
remove the old sci-fi-like separated surfaces. Preserve `data-override` with a
purple magical overlay and reduced-motion fallback.

Rebuild `LabSummary` into explicit sections: Energy gauge first, 20 Observer
markers second, Override third, compact exploration/Portal figures last. Keep
all values sourced from the same snapshot.

- [ ] **Step 4: verify and commit the visual foundation**

```bash
npm --prefix web test -- --run src/app/App.test.tsx src/pages/DashboardPage.test.tsx
npm --prefix web run typecheck
git add web/public/ui/lab-shell-background.png web/src/styles \
  web/src/app/AppShell.tsx web/src/app/AppShell.module.css \
  web/src/components/dashboard/LabSummary.tsx \
  web/src/components/dashboard/LabSummary.module.css \
  web/src/app/App.test.tsx web/src/pages/DashboardPage.test.tsx
git commit -m "feat(ui-corrective): build arcane laboratory shell"
```

## Task 4 — Match the reference Dashboard and semantic value hierarchy

**Files:**

- Modify: `web/src/pages/DashboardPage.tsx`
- Modify: `web/src/pages/DashboardPage.module.css`
- Modify: `web/src/components/dashboard/PortalSlot.tsx`
- Modify: `web/src/components/dashboard/PortalSlot.module.css`
- Modify: `web/src/components/dashboard/EmptyPortalSlot.tsx`
- Modify: `web/src/features/portal-actions/PortalActions.module.css`
- Modify: `web/src/components/portal/ObserverTransit.tsx`
- Test: `web/src/pages/DashboardPage.test.tsx`
- Test: `web/src/features/portal-actions/PortalActions.test.tsx`

- [ ] **Step 1: add semantic styling RED assertions**

Require occupied cards to expose distinct metric hooks:

```tsx
expect(within(card).getByTestId("portal-energy")).toHaveAttribute("data-value-kind", "energy");
expect(within(card).getByTestId("portal-time")).toHaveAttribute("data-value-kind", "time");
expect(within(card).getByText("UNSTABLE")).toHaveAttribute("data-stability", "UNSTABLE");
```

Require exactly four command buttons per occupied or empty Slot and no Details
control. Run the focused files and confirm RED on the missing hooks.

- [ ] **Step 2: implement the exact `4 + 3` composition**

Keep the eight-column/two-row grid. Slots 1–4 each span two columns; Slots 5–7
occupy `2/4`, `4/6`, `6/8`. Give every Slot the same min/max sizing and reserve
one viewport row for the title/notification area. The board and Dashboard use
`overflow: hidden`.

- [ ] **Step 3: rebuild card hierarchy and commands**

Use a tall antique-gold card with compact header, large circular `PortalEffect`,
one semantic metric line, optional compact transit line and a single four-column
command row. Portal colour continues to come from the existing deterministic
Portal hue. Empty cards use the identical grid footprint and disabled buttons.

Add explicit `data-value-kind`, `data-stability` and `data-direction` attributes
without changing DTOs. Apply the approved Energy/Time/transit/risk colours and
retain text labels for accessibility.

- [ ] **Step 4: run GREEN and commit**

```bash
npm --prefix web test -- --run \
  src/pages/DashboardPage.test.tsx \
  src/features/portal-actions/PortalActions.test.tsx \
  src/portal-fx/PortalEffect.test.tsx
git add web/src/pages/DashboardPage.tsx web/src/pages/DashboardPage.module.css \
  web/src/components/dashboard web/src/components/portal/ObserverTransit.tsx \
  web/src/features/portal-actions/PortalActions.module.css \
  web/src/pages/DashboardPage.test.tsx \
  web/src/features/portal-actions/PortalActions.test.tsx
git commit -m "feat(ui-corrective): match reference portal board"
```

## Task 5 — Apply the same composition to Details and secondary routes

**Files:**

- Modify: `web/src/pages/PortalDetailsPage.tsx`
- Modify: `web/src/pages/PortalDetailsPage.module.css`
- Modify: `web/src/components/portal/PortalFacts.tsx`
- Modify: `web/src/components/portal/Diagnostics.module.css`
- Modify: `web/src/pages/EventLogPage.module.css`
- Modify: `web/src/pages/AIWorklogPage.module.css`
- Modify: `web/src/features/tutorial/TutorialPanel.module.css`
- Test: `web/src/pages/PortalDetailsPage.test.tsx`
- Test: `web/src/pages/EventLogPage.test.tsx`

- [ ] **Step 1: add Details semantic RED assertions**

Require Energy, Time, Status and Stability values to expose their semantic data
attributes. Retain the existing LOW/MEDIUM/HIGH/CRITICAL and Recommendation
attributes. Require the Actions section to remain directly beneath the central
Portal and History to remain a collapsed bounded region.

- [ ] **Step 2: implement the one-viewport Details composition**

Use a three-column body: Portal facts/transit, large Portal plus actions,
destination/diagnostics. Keep History as the bottom row and allow only its inner
list to scroll. Apply terminal grayscale, semantic value colours, parchment
disclosures and the shared antique-gold panel treatment.

- [ ] **Step 3: finish Tutorial and secondary-route styling**

Make Tutorial a parchment scroll matching the reference, still pointer-safe and
layout-independent. Give Events, Help and AI Worklog the same transparent dark
parchment surfaces and gold rules. Preserve bounded inner scrolling for their
long-form content.

- [ ] **Step 4: run GREEN and commit**

```bash
npm --prefix web test -- --run \
  src/pages/PortalDetailsPage.test.tsx \
  src/pages/EventLogPage.test.tsx \
  src/features/tutorial/TutorialPanel.test.tsx
git add web/src/pages/PortalDetailsPage.tsx \
  web/src/pages/PortalDetailsPage.module.css \
  web/src/components/portal/PortalFacts.tsx \
  web/src/components/portal/Diagnostics.module.css \
  web/src/pages/EventLogPage.module.css web/src/pages/AIWorklogPage.module.css \
  web/src/features/tutorial/TutorialPanel.module.css \
  web/src/pages/PortalDetailsPage.test.tsx web/src/pages/EventLogPage.test.tsx
git commit -m "feat(ui-corrective): unify portal and archive surfaces"
```

## Task 6 — Verify, document and prepare the deployed update

**Files:**

- Modify: `00_FINAL_SPEC_v5.md`
- Modify: `01_AI_WORKLOG_CURRENT.md`
- Modify: `docs/traceability.md`
- Modify: `docs/deployment.md`

- [ ] **Step 1: run the full frontend gate**

```bash
npm --prefix web run format
npm --prefix web run format:check
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web test
npm --prefix web run build
```

Expected: every command exits `0` and all frontend test files pass.

- [ ] **Step 2: rebuild the production image**

```bash
docker compose --env-file .env config
docker build -t omenpath:local .
```

Expected: Compose still publishes only `127.0.0.1:8080:8080`; Docker builds the
new SPA into the unchanged single-container runtime.

- [ ] **Step 3: perform visual acceptance**

Launch the application locally and capture Dashboard plus Portal Details at
1280×720. Confirm against the approved image: continuous background/frame,
integrated left summary, parchment Tutorial, centred `4 + 3`, four-button cards,
semantic value colours and no page scrollbar. Also inspect the compact phone
layout with all seven Slots present.

- [ ] **Step 4: update authoritative documentation**

Change Final Spec Tutorial wording from opening a Details button to selecting
the highlighted Portal card. Record the two frontend-only authoritative route
redirects. Update traceability with the new regression tests and visual evidence.
Append actual RED/GREEN hashes and verification output to Worklog. Add the
normal server update command to deployment notes without changing topology.

- [ ] **Step 5: commit the corrective close**

```bash
git add 00_FINAL_SPEC_v5.md 01_AI_WORKLOG_CURRENT.md docs/traceability.md docs/deployment.md
git commit -m "docs(ui-corrective): record reference fidelity verification"
```

- [ ] **Step 6: deploy the already-built change**

After pushing `main`, run on the VPS:

```bash
cd /opt/omenpath
git pull --ff-only
docker compose --env-file .env build
docker compose --env-file .env up -d --remove-orphans
docker compose ps
curl -fsS http://127.0.0.1:8080/health
curl -fsS https://omenpath.duckdns.org/health
```

The nginx configuration, persistent `/opt/omenpath/data`, sessions and database
remain unchanged.
