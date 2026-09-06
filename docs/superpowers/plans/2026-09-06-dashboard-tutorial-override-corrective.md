# Dashboard, Tutorial and Override Corrective Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Observer state readable, keep Tutorial users on Portal Details until existing `Forward` returns them to Dashboard, and make Leyline Override visually unmistakable without a redundant banner.

**Architecture:** Keep authoritative counts and Tutorial progression unchanged. Derive pip presentation inside `LabSummary`, make `TutorialPanel` route-aware for the existing `Forward` control, and implement Override as a pointer-transparent shell presentation layer controlled by the existing `data-override` attribute.

**Tech Stack:** React 19, TypeScript, React Router, CSS Modules, Vitest, Testing Library, Go verification suite.

---

### Task 1: Observer Life classification

**Files:**
- Modify: `web/src/components/dashboard/LabSummary.tsx`
- Modify: `web/src/components/dashboard/LabSummary.module.css`
- Test: `web/src/pages/DashboardPage.test.tsx`

- [ ] **Step 1: Write the failing summary test**

Set `available = 4`, `in_worlds = 13`, `in_transit = 2`, `lost = 1`, render Dashboard, and assert:

```tsx
expect(summary).toHaveTextContent("19 / 20");
expect(summary).not.toHaveTextContent("Available 4");
expect(within(summary).getAllByTestId("observer-pip")).toHaveLength(20);
expect(within(summary).getAllByTestId("observer-pip", { exact: true })
  .filter((pip) => pip.dataset.state === "available")).toHaveLength(4);
expect(screen.getByLabelText(/4 available, 15 deployed, 1 lost/i)).toBeVisible();
```

- [ ] **Step 2: Run the focused test and observe RED**

Run: `npm --prefix web test -- src/pages/DashboardPage.test.tsx`

Expected: FAIL because the old visible Observer row remains, the numeric value uses `in_lab`, and pips have no three-state classification.

- [ ] **Step 3: Implement the minimal derived presentation**

In `LabSummary.tsx`, derive:

```ts
const totalObservers = 20;
const survivingObservers = totalObservers - snapshot.observers.lost;
const deployedObservers =
  snapshot.observers.in_worlds + snapshot.observers.in_transit;
const pipStates = [
  ...Array(snapshot.observers.available).fill("available"),
  ...Array(deployedObservers).fill("deployed"),
  ...Array(snapshot.observers.lost).fill("lost"),
];
```

Render `survivingObservers / 20`, give the pip container the accessible count sentence, assign each pip `data-state`, and remove the visible compact Observers row. In CSS, keep deployed pips dark, retain green available pips, add a red lost gradient/glow, and increase `.compact` and compact label sizes.

- [ ] **Step 4: Run focused GREEN**

Run: `npm --prefix web test -- src/pages/DashboardPage.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit RED/GREEN checkpoint**

```bash
git add web/src/components/dashboard/LabSummary.tsx web/src/components/dashboard/LabSummary.module.css web/src/pages/DashboardPage.test.tsx
git commit -m "fix(ui): clarify observer life states"
```

### Task 2: Reuse Forward for the Details-to-Dashboard transition

**Files:**
- Modify: `web/src/pages/PortalDetailsPage.tsx`
- Modify: `web/src/features/tutorial/TutorialPanel.tsx`
- Test: `web/src/pages/PortalDetailsPage.test.tsx`
- Test: `web/src/features/tutorial/TutorialPanel.test.tsx`

- [ ] **Step 1: Write failing navigation tests**

Change the Portal Details signal test to assert that accepting authoritative Step 2 leaves the route at `/portals/42`. Add Tutorial tests proving:

```tsx
// At the latest Tutorial card on /portals/42:
expect(screen.getByRole("button", { name: "Forward" })).toBeEnabled();
await user.click(screen.getByRole("button", { name: "Forward" }));
expect(screen.getByTestId("location")).toHaveTextContent("/");

// While viewing an older card, Forward advances history without navigating.
expect(screen.getByTestId("location")).toHaveTextContent("/portals/42");
```

- [ ] **Step 2: Run focused tests and observe RED**

Run: `npm --prefix web test -- src/pages/PortalDetailsPage.test.tsx src/features/tutorial/TutorialPanel.test.tsx`

Expected: FAIL because Details currently calls `navigate("/")` after the signal and `Forward` is disabled at the newest card.

- [ ] **Step 3: Implement route-aware Forward behavior**

Remove the success-path `navigate("/")` from `PortalDetailsPage`. In `TutorialPanel`, read `pathname`, derive `canReturnFromDetails` for `/portals/<positive-id>` when the shown card is the latest objective and Step 1 has completed, then use:

```ts
const runForward = () => {
  if (presentation.cursor < presentation.history.length - 1) {
    forward();
    return;
  }
  if (canReturnFromDetails) navigate("/");
};
```

The existing `Forward` button calls `runForward` and is disabled only when neither history advancement nor Details return is available.

- [ ] **Step 4: Run focused GREEN**

Run: `npm --prefix web test -- src/pages/PortalDetailsPage.test.tsx src/features/tutorial/TutorialPanel.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit RED/GREEN checkpoint**

```bash
git add web/src/pages/PortalDetailsPage.tsx web/src/pages/PortalDetailsPage.test.tsx web/src/features/tutorial/TutorialPanel.tsx web/src/features/tutorial/TutorialPanel.test.tsx
git commit -m "fix(tutorial): keep details open until forward"
```

### Task 3: Strong Override energy presentation without banner

**Files:**
- Modify: `web/src/app/AppShell.tsx`
- Modify: `web/src/app/AppShell.module.css`
- Modify: `web/src/pages/DashboardPage.tsx`
- Modify: `web/src/pages/DashboardPage.module.css`
- Test: `web/src/app/App.test.tsx`
- Test: `web/src/pages/DashboardPage.test.tsx`

- [ ] **Step 1: Write failing presentation tests**

Assert the active shell contains an inert visual layer and Dashboard contains no override deadline status:

```tsx
expect(screen.getByTestId("leyline-energy-layer")).toHaveAttribute("aria-hidden", "true");
expect(screen.queryByText(/Leyline Override active until/i)).not.toBeInTheDocument();
expect(screen.getByText("ACTIVE")).toBeVisible();
```

- [ ] **Step 2: Run focused tests and observe RED**

Run: `npm --prefix web test -- src/app/App.test.tsx src/pages/DashboardPage.test.tsx`

Expected: FAIL because the energy layer is absent and the floating deadline status is still rendered.

- [ ] **Step 3: Implement the energy layer**

Add `<div aria-hidden="true" className={styles.overrideEnergy} data-testid="leyline-energy-layer" />` inside the shell. Style it as absolute, full-shell, pointer-transparent and hidden by default. Under `[data-override]`, show layered violet/white diagonal gradients with `mix-blend-mode: screen`, animate background position and opacity, and keep all shell content above it. Under `prefers-reduced-motion`, disable both Override animations while retaining the static glow. Remove the Dashboard override paragraph and its obsolete CSS selector.

- [ ] **Step 4: Run focused GREEN**

Run: `npm --prefix web test -- src/app/App.test.tsx src/pages/DashboardPage.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit RED/GREEN checkpoint**

```bash
git add web/src/app/AppShell.tsx web/src/app/AppShell.module.css web/src/app/App.test.tsx web/src/pages/DashboardPage.tsx web/src/pages/DashboardPage.module.css web/src/pages/DashboardPage.test.tsx
git commit -m "fix(ui): strengthen leyline override feedback"
```

### Task 4: Source-of-truth and final verification

**Files:**
- Modify: `00_FINAL_SPEC_v5.md`
- Modify: `01_AI_WORKLOG_CURRENT.md`
- Modify: `docs/traceability.md`

- [ ] **Step 1: Update documentation**

Revise Dashboard presentation wording to record the compact Observer Life encoding without changing authoritative API counts. Record the corrective pass and map its frontend tests in traceability.

- [ ] **Step 2: Run frontend quality suite**

```bash
npm --prefix web run format:check
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web test
npm --prefix web run build
```

Expected: every command succeeds; all 192 existing tests plus new assertions pass.

- [ ] **Step 3: Run repository quality suite**

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
```

Expected: `gofmt -l .` emits no paths and all Go commands succeed.

- [ ] **Step 4: Commit documentation and verification record**

```bash
git add 00_FINAL_SPEC_v5.md 01_AI_WORKLOG_CURRENT.md docs/traceability.md
git commit -m "docs(ui): record dashboard corrective verification"
```
