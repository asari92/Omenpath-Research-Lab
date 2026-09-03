# Omenpath Research Lab — Stage 6
## Emergency / Leyline Override Implementation Plan via TDD

> **For agentic workers:** REQUIRED SUB-SKILL: use
> `superpowers:subagent-driven-development` when explicitly permitted, or
> `superpowers:executing-plans` to implement this plan checkpoint-by-checkpoint.
> Steps use checkbox (`- [ ]`) syntax for execution tracking.
>
> Depends on:
> - `00_FINAL_SPEC_v5.md`
> - `01_AI_WORKLOG_CURRENT.md`
> - `02_IMPLEMENTATION_ROADMAP_TDD.md`
> - `04_STAGE_02_PORTAL_CORE_TDD.md`
> - `05_STAGE_03_OBSERVER_LIFECYCLE_TDD.md`
> - `06_STAGE_04_PORTAL_OBSERVER_FLOW_TDD.md`
> - `07_STAGE_05_LABORATORY_ENERGY_TDD.md`

**Goal:** implement Collapse-driven Laboratory Energy reset and the
20-second Leyline Override, including free Close/Stabilize during the active
window while preserving every completed Portal, Observer and Lab restriction.

**Architecture:** add a focused emergency domain module that atomically wraps
`Portal.ResolveLifecycle`, applies a Collapse to `LabState` at the Portal's
semantic `ClosedAt`, and exposes a pure active-window query. Extend only the
Stage 5 Close/Stabilize cost selection; keep Portal lifecycle, Observer flow,
Extraction, simulation, events and persistence outside this stage.

**Tech stack:** Go, `time.Time`/`time.Duration`, existing `config.Config`,
`testify/require`, deterministic `testutil` fixtures, and repository Git/TDD
protocol.

---

# 1. Goal

Stage 6 answers:

```text
Did an OPEN Portal become COLLAPSED during lifecycle resolution?
At what semantic moment did the Collapse occur?
What Lab Energy baseline and override deadline follow from it?
Is the Override active at a supplied time?
What effective Close/Stabilize cost applies at that time?
```

Required behavior:

```text
every COLLAPSED transition
  → Lab Energy baseline 0 at Portal.ClosedAt
  → LeylineOverrideUntil = Portal.ClosedAt + 20 sec

active interval
  → [collapseAt, collapseAt + 20 sec)

during active Override
  → Close cost 0
  → Stabilize cost 0
  → all other restrictions remain
  → Extraction still costs 30
  → Lab Energy continues derived +1/sec

another Collapse
  → Energy baseline reset to 0 again
  → deadline replaced with new collapseAt + 20 sec
```

Stage 6 must reuse rather than duplicate:

```text
Stage 2 Portal.ResolveLifecycle / Portal.Stabilize
Stage 4 ClosePortalWithObservers
Stage 5 LabState derived energy and energy-aware wrappers
```

---

# 2. Source-of-truth order

Use the repository hierarchy fixed in `AGENTS.md`:

```text
00_FINAL_SPEC_v5.md
→ 01_AI_WORKLOG_CURRENT.md (history only; never overrides Final Spec)
→ 02_IMPLEMENTATION_ROADMAP_TDD.md
→ this Stage 6 detailed plan
→ docs/requirements.md
→ tests
→ implementation
```

Final Spec §§14, 21, 22 and 37 control the gameplay behavior. Stop if an
implementation checkpoint reveals a real contradiction rather than choosing
new gameplay semantics locally.

---

# 3. Verified starting point

Stage 5 ended at commit `a6167c0` with a clean worktree and these APIs:

```go
func (l LabState) CurrentEnergy(now time.Time, cfg config.Config) int
func (l LabState) CanAfford(now time.Time, cost int, cfg config.Config) bool
func (l *LabState) SpendEnergy(now time.Time, cost int, cfg config.Config) error

func ClosePortalWithLabEnergy(
    lab *LabState,
    portal *Portal,
    plane *Plane,
    observers []Observer,
    now time.Time,
    confirm bool,
    cfg config.Config,
) error

func StabilizePortalWithLabEnergy(
    lab *LabState,
    portal *Portal,
    now time.Time,
    cfg config.Config,
) error
```

`LabState` already contains the Stage 6 storage field:

```go
type LabState struct {
    EnergyBase   int
    EnergyBaseAt time.Time

    LeylineOverrideUntil *time.Time
}
```

`Portal.ResolveLifecycle(now)` already:

```text
returns changed=false for OPEN before every deadline
returns changed=false for an already-terminal Portal
uses semantic ClosedAt for a newly reached terminal transition
distinguishes CLOSED from COLLAPSED
distinguishes ENERGY_DEPLETED from INSTABILITY
preserves deterministic tie rules
```

`config.Default().EmergencyDuration` is already 20 seconds.

---

# 4. Requirements covered

Primary family:

```text
EMERGENCY-001  every Collapse resets Lab Energy to 0
EMERGENCY-002  Leyline Override lasts 20 seconds
EMERGENCY-003  Close/Stabilize cost 0 while active; restrictions remain
EMERGENCY-004  Extraction remains cost 30
EMERGENCY-005  Lab Energy regeneration continues during Override
EMERGENCY-006  another Collapse resets Energy and deadline
```

Existing rows enhanced without changing their semantics:

```text
PORTAL-007     ENERGY_DEPLETED Collapse
PORTAL-008     INSTABILITY Collapse
LAB-003        derived regeneration
LAB-007        Close cost selection
LAB-008        Stabilize cost selection
LAB-009        Extraction cost boundary
LAB-010        insufficient paid-action rejection
STABILITY-005  successful Stabilize transition
ENERGY-006..008 Portal Energy remains independent
OBSERVER-012   confirmed manual Close loss behavior
```

Expected honest boundary:

```text
EMERGENCY-001  GREEN
EMERGENCY-002  GREEN
EMERGENCY-003  GREEN
EMERGENCY-004  PARTIAL  # generic cost invariant; real Extraction Stage 7
EMERGENCY-005  GREEN
EMERGENCY-006  GREEN

LAB-009        remains PARTIAL until Stage 7
LAB-010        remains PARTIAL until real Extraction integration Stage 7
```

---

# 5. Hard stage boundary

Stage 6 MUST NOT implement:

```text
Extraction Portal creation
Extraction Plane selection
Extraction synchronization or automatic RECALL
natural Portal spawning or scheduling
simulation tick/goroutine
Event creation, deduplication or ACTION_REJECTED
SQLite schemas/repositories/recovery
LabManager, mutexes or concurrent command arbitration
REST/HTTP/WebSocket/frontend behavior
Tutorial progression
automatic clearing writes when Override expires
```

Do not start Stage 7. `cfg.ExtractionCost` may appear only in characterization
tests using the already-completed generic energy primitive.

---

# 6. Selected architecture choice

## A. Emergency module plus thin Stage 5 cost extension — selected

Create one emergency module with:

```text
pure override-window query
atomic Collapse application to LabState
atomic Portal lifecycle + Lab emergency orchestration
```

Modify Stage 5 wrappers only where effective Close/Stabilize costs are chosen.

Advantages:

- preserves Portal/Lab aggregate separation;
- uses the existing semantic Portal transition time;
- makes late resolution deterministic;
- prevents replaying an already-terminal Portal from resetting Lab Energy;
- keeps Stage 7 Extraction and Stage 11 locking out of scope;
- gives future LabManager one reusable transaction boundary.

## B. Put LabState inside `Portal.ResolveLifecycle` — rejected

This would couple the Portal entity to a different aggregate and change a
completed Stage 2 API only to serve a later stage.

## C. Wait for LabManager and implement no Collapse integration — rejected

That would leave EMERGENCY-001/006 untestable and make Stage 6 only a cost
helper rather than a complete domain slice.

---

# 7. Production file map

Create:

```text
internal/domain/lab_emergency.go
```

Responsibilities:

```text
LabState.LeylineOverrideActive
LabState.ActivateLeylineOverride
ResolvePortalLifecycleWithLabEmergency
```

Modify:

```text
internal/domain/lab_energy_commands.go
```

Responsibility:

```text
select effective Close/Stabilize cost from active Override state
```

Do not modify unless a genuine RED proves it necessary:

```text
internal/domain/portal.go
internal/domain/observer_commands.go
internal/domain/portal_observer_close.go
internal/domain/observer_lifecycle.go
```

No new error is expected. Invalid emergency state/time uses the stable Stage 5
`ErrLabEnergyInvariant`.

---

# 8. Proposed emergency API

Use this public surface:

```go
func (l LabState) LeylineOverrideActive(
    now time.Time,
    cfg config.Config,
) bool

func (l *LabState) ActivateLeylineOverride(
    collapseAt time.Time,
    cfg config.Config,
) error

func ResolvePortalLifecycleWithLabEmergency(
    lab *LabState,
    portal *Portal,
    now time.Time,
    cfg config.Config,
) (changed bool, err error)
```

The lifecycle wrapper is deliberately singular: one Portal transition and one
Lab aggregate. Stage 8/11 will iterate and serialize multiple Portals.

---

# 9. Override interval semantics

The approved active interval is half-open:

```text
[collapseAt, collapseAt + cfg.EmergencyDuration)
```

With the default duration:

```text
T - 1ns        inactive
T              active
T + 19.999s    active
T + 20s        inactive
T + 20s + 1ns  inactive
```

Only the deadline is stored. The start for pure historical evaluation is:

```text
start = LeylineOverrideUntil - cfg.EmergencyDuration
```

Therefore `LeylineOverrideActive` must check both bounds. This remains correct
if a future Stage 7 Extraction debit changes `EnergyBaseAt` while the Override
is active; the active window must not depend on the current energy baseline.

---

# 10. Expiry is derived

Reaching the deadline must not write state:

```text
LeylineOverrideUntil remains stored
LeylineOverrideActive(deadline) returns false
```

No timer, goroutine, cleanup write or tick belongs in Stage 6. Persistence and
UI can retain the deadline while deriving the active flag.

---

# 11. Collapse timestamp

The Collapse is applied at:

```text
*Portal.ClosedAt
```

not at the later resolver call time.

Example:

```text
semantic Collapse at T+10s
resolved at T+15s

EnergyBase   = 0
EnergyBaseAt = T+10s
OverrideUntil = T+30s
CurrentEnergy(T+15s) = 5
```

This preserves the completed semantic-time convention and Final Spec's rule
that regeneration continues during Override.

---

# 12. Activating the Override

`ActivateLeylineOverride(collapseAt, cfg)` performs one atomic transition:

```text
EnergyBase = 0
EnergyBaseAt = collapseAt
LeylineOverrideUntil = collapseAt + cfg.EmergencyDuration
```

It must validate before mutation:

```text
LabState pointer is non-nil
stored EnergyBase is inside 0..cfg.LabEnergyMax
collapseAt is not before EnergyBaseAt
cfg.EmergencyDuration is positive
```

An invalid call returns `ErrLabEnergyInvariant` and leaves the entire LabState
unchanged.

The chronological check enforces the repository's resolve-first rule. An old
unresolved Collapse cannot move a newer Lab baseline backward; Stage 8/11 must
process semantic transitions in nondecreasing time order.

---

# 13. Repeated Collapse

A later Collapse is not an extension from the old deadline. It replaces the
state from the new semantic moment:

```text
first Collapse  T     → base 0 at T,     until T+20
regen to 7      T+7   → current 7
second Collapse T+7   → base 0 at T+7,   until T+27
```

The same operation applies whether the new reason is ENERGY_DEPLETED or
INSTABILITY.

An already-terminal Portal is not a new Collapse transition. Re-resolving it
must not reset energy or the deadline.

---

# 14. Atomic lifecycle orchestration

`ResolvePortalLifecycleWithLabEmergency` uses copy-then-commit:

```text
1. validate LabState and supplied now
2. reject nil Portal before mutation
3. copy Portal and LabState
4. call ResolveLifecycle(now) on the Portal copy
5. if unchanged, return changed=false without writes
6. if result is COLLAPSED, activate on the Lab copy at Portal copy ClosedAt
7. commit both copies only after every step succeeds
```

Natural/manual CLOSED is not an emergency and leaves LabState unchanged.

Copy-then-commit is required because `Portal.ResolveLifecycle` mutates before
the Lab transition is known to be valid. Do not mutate the real Portal and
then attempt rollback.

Nil Portal should return the existing `ErrPortalNotOpen`; it must not panic or
change LabState.

---

# 15. Effective cost selection

Add unexported helpers in `lab_energy_commands.go`:

```go
func effectiveCloseCost(lab *LabState, now time.Time, cfg config.Config) int
func effectiveStabilizeCost(lab *LabState, now time.Time, cfg config.Config) int
```

Rules:

```text
active Override   → 0
inactive Override → cfg.CloseCost / cfg.StabilizeCost
nil LabState      → ordinary cost; prepareLabEnergySpend returns invariant
```

The existing wrapper transaction order remains:

```text
validate/derive and preflight effective cost
→ delegate completed command
→ commit effective debit after success
```

Do not introduce a generic action enum or modify `Portal.Close`/
`Portal.Stabilize`.

---

# 16. Free Close behavior

During active Override, `ClosePortalWithLabEnergy`:

```text
permits an otherwise-valid Close at Lab Energy 0
does not change EnergyBase or EnergyBaseAt
does not change LeylineOverrideUntil
still requires OPEN Portal
still requires creature/transit confirmation
still makes an active transiting Observer LOST after confirmation
still preserves every Stage 4 aggregate invariant
```

At the exact override deadline, normal configured cost applies again.

---

# 17. Free Stabilize behavior

During active Override, `StabilizePortalWithLabEnergy`:

```text
permits otherwise-valid Stabilize at Lab Energy 0
does not change Lab baseline fields or override deadline
still requires OPEN + UNSTABLE
still rejects current Portal Energy >85
still clears the hidden collapse timestamp on success
still adds 15 Portal Energy and re-baselines Portal Energy
still preserves Portal decay
```

Lab Energy continues to derive from the Collapse baseline. A free action does
not erase already-earned regeneration.

---

# 18. Restrictions remain

Override changes only two prices. It does not bypass:

```text
terminal Portal rejection
STABLE Portal rejection for Stabilize
Portal overcharge rejection
creature confirmation
active-transit confirmation and LOST result
Portal/Plane/Observer structural validation
direction or Observer flow rules
```

Confirmation/domain failures never debit or re-baseline Lab Energy.

---

# 19. Extraction boundary

Stage 6 proves only:

```go
lab.SpendEnergy(now, cfg.ExtractionCost, cfg)
```

uses 30 even when `LeylineOverrideActive(now, cfg)` is true.

No effective-cost exception is added to `SpendEnergy`. No Extraction wrapper,
Portal, Plane selection or synchronization is added.

Consequently `EMERGENCY-004` and `LAB-009` remain PARTIAL until Stage 7 wires
the tested debit into the real Extraction command.

---

# 20. Regeneration during Override

Activation reuses the normal energy baseline model:

```text
Collapse at T      → base 0 at T
T + 999ms          → 0
T + 1s             → 1
T + 19s            → 19
T + 20s            → 20, Override inactive
```

Do not add a special emergency regeneration function. `CurrentEnergy` remains
the single source for derived energy.

---

# 21. Stage 5 compatibility

Outside the active interval, the exact Stage 5 behavior remains:

```text
SEND       0
RECALL     0
CLOSE      cfg.CloseCost / default 5
STABILIZE  cfg.StabilizeCost / default 20
EXTRACTION cfg.ExtractionCost / default 30
```

Existing Stage 5 tests must remain unchanged and GREEN. Add Stage 6 tests; do
not rewrite ordinary-cost assertions to depend on an Override fixture.

---

# 22. Resolve-first boundary

The Stage 6 lifecycle wrapper resolves one Portal and Lab emergency together,
but it does not become `LabManager`.

Future Stage 11 still owns:

```text
one lock around full aggregate resolution and command
cross-Portal chronological ordering
Observer lifecycle resolution
persistence and event transaction
concurrent command arbitration
```

Stage 6 code is deterministic sequential domain orchestration only.

---

# 23. No event semantics

Do not emit:

```text
PORTAL_COLLAPSED
LAB_ENERGY_CHANGED
LEYLINE_OVERRIDE_STARTED
LEYLINE_OVERRIDE_EXPIRED
ACTION_REJECTED
```

Stage 9 defines durable Events. An expired deadline is derived state, not a
Stage 6 event or mutation.

---

# 24. Test file map

Create:

```text
internal/domain/lab_emergency_active_test.go
internal/domain/lab_emergency_energy_collapse_test.go
internal/domain/lab_emergency_instability_test.go
internal/domain/lab_emergency_repeat_test.go
internal/domain/lab_emergency_close_test.go
internal/domain/lab_emergency_stabilize_test.go
internal/domain/lab_emergency_extraction_test.go
internal/domain/lab_emergency_integration_test.go
```

Reuse Stage 4/5 test helpers where their package-level names remain clear:

```text
labAt
stage4Portal
stage4Plane
outboundObserver
unstablePortal
```

No fixture may use wall-clock time, sleep, a real random source, database,
HTTP, goroutines or production commands to construct its initial state.

---

# 25. Fixed fixtures

Use:

```text
Base time             testutil.BaseTime
Emergency duration    config.Default().EmergencyDuration = 20 sec
Lab maximum           100
Lab regeneration      1/sec
Close cost            5
Stabilize cost        20
Extraction cost       30
Portal ID              11 where Stage 4 fixtures are involved
Destination Plane ID  7
```

Provide an explicit collapsed fixture only for read/activation tests. Tests of
the lifecycle wrapper must start from OPEN and allow the wrapper itself to
produce the terminal transition.

---

# 26. TDD checkpoint A — active window and activation

Requirements:

```text
EMERGENCY-001
EMERGENCY-002
```

Required tests:

```text
TestLabState_ActivateLeylineOverrideResetsEnergyToZero
TestLabState_ActivateLeylineOverrideUsesCollapseTimestampAsBaseline
TestLabState_ActivateLeylineOverrideSetsConfiguredDeadline
TestLabState_ActivateLeylineOverrideReplacesExistingDeadline
TestLabState_ActivateLeylineOverrideRejectsBackwardCollapseTime
TestLabState_ActivateLeylineOverrideRejectsInvalidBaseline
TestLabState_ActivateLeylineOverrideRejectsNonPositiveDuration
TestLabState_ActivateLeylineOverrideFailureIsAtomic

TestLabState_LeylineOverrideActiveAtStart
TestLabState_LeylineOverrideActiveBeforeDeadline
TestLabState_LeylineOverrideInactiveBeforeStart
TestLabState_LeylineOverrideInactiveAtDeadline
TestLabState_LeylineOverrideInactiveAfterDeadline
TestLabState_LeylineOverrideInactiveWithoutDeadline
TestLabState_LeylineOverrideActiveDoesNotMutateState
```

Representative test shape:

```go
func TestLabState_LeylineOverrideInactiveAtDeadline(t *testing.T) {
    cfg := config.Default()
    lab := labAt(75, testutil.BaseTime)
    require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))

    require.False(t, lab.LeylineOverrideActive(
        testutil.BaseTime.Add(cfg.EmergencyDuration), cfg,
    ))
}
```

Execution:

- [ ] Add only checkpoint A tests.
- [ ] Run `go test -count=1 ./internal/domain`.
- [ ] Observe RED from undefined `ActivateLeylineOverride` and
      `LeylineOverrideActive`.
- [ ] Commit `test(stage6): RED checkpoint A override activation and window`.
- [ ] Add `internal/domain/lab_emergency.go` with the two minimal methods.
- [ ] Run targeted tests and observe GREEN.
- [ ] Run the complete required verification suite.
- [ ] Mark EMERGENCY-001 PARTIAL for the primitive and EMERGENCY-002 GREEN.
- [ ] Commit `feat(stage6): GREEN checkpoint A override activation and window`.

---

# 27. TDD checkpoint B — energy-depletion Collapse orchestration

Requirements:

```text
EMERGENCY-001
PORTAL-007 integration
EMERGENCY-005 late-resolution basis
```

Required tests:

```text
TestResolvePortalLifecycleWithLabEmergency_EnergyDepletionResetsLab
TestResolvePortalLifecycleWithLabEmergency_EnergyDepletionStartsOverride
TestResolvePortalLifecycleWithLabEmergency_UsesPortalClosedAt
TestResolvePortalLifecycleWithLabEmergency_LateResolutionIncludesRegeneration
TestResolvePortalLifecycleWithLabEmergency_BeforeDeadlineChangesNothing
TestResolvePortalLifecycleWithLabEmergency_NaturalCloseDoesNotStartOverride
TestResolvePortalLifecycleWithLabEmergency_AlreadyCollapsedDoesNotResetAgain
TestResolvePortalLifecycleWithLabEmergency_AlreadyClosedDoesNotChangeLab
TestResolvePortalLifecycleWithLabEmergency_InvalidLabIsAtomic
TestResolvePortalLifecycleWithLabEmergency_NilPortalDoesNotMutateLab
```

Representative late-resolution assertion:

```go
changed, err := domain.ResolvePortalLifecycleWithLabEmergency(
    &lab, &portal, collapseAt.Add(5*time.Second), cfg,
)
require.NoError(t, err)
require.True(t, changed)
require.Equal(t, collapseAt, lab.EnergyBaseAt)
require.Equal(t, 5, lab.CurrentEnergy(collapseAt.Add(5*time.Second), cfg))
```

Execution:

- [ ] Add only checkpoint B tests.
- [ ] Run the targeted package and observe undefined-wrapper RED.
- [ ] Commit `test(stage6): RED checkpoint B energy collapse orchestration`.
- [ ] Implement copy-then-commit lifecycle orchestration.
- [ ] Preserve the existing Portal outcome and semantic `ClosedAt`.
- [ ] Run targeted and full verification suites.
- [ ] Keep EMERGENCY-001 PARTIAL until both Collapse causes are covered; add
      wrapper evidence to PORTAL-007 and LAB-003.
- [ ] Commit `feat(stage6): GREEN checkpoint B energy collapse orchestration`.

---

# 28. TDD checkpoint C — instability Collapse orchestration

Requirements:

```text
EMERGENCY-001
PORTAL-008 integration
```

Required tests:

```text
TestResolvePortalLifecycleWithLabEmergency_InstabilityResetsLab
TestResolvePortalLifecycleWithLabEmergency_InstabilityStartsOverride
TestResolvePortalLifecycleWithLabEmergency_InstabilityUsesHiddenCollapseTime
TestResolvePortalLifecycleWithLabEmergency_LateInstabilityResolutionRegenerates
TestResolvePortalLifecycleWithLabEmergency_NaturalCloseTieDoesNotStartOverride
TestResolvePortalLifecycleWithLabEmergency_EnergyInstabilityTieStartsOneOverride
TestResolvePortalLifecycleWithLabEmergency_PreservesEnergyDepletedTieReason
TestResolvePortalLifecycleWithLabEmergency_StablePortalCannotCollapseByHiddenTime
```

Do not change Stage 2 tie semantics. The emergency wrapper reacts to the
resulting `PortalStatusCollapsed`; it does not choose a termination reason.

Execution:

- [ ] Add checkpoint C tests after checkpoint B is GREEN.
- [ ] Run targeted tests.
- [ ] If they are already GREEN, record characterization honestly; do not
      manufacture a failure.
- [ ] If a real orchestration gap appears, commit RED before its minimal fix.
- [ ] Run the full verification suite.
- [ ] Add wrapper evidence to PORTAL-008 and promote EMERGENCY-001 to GREEN.
- [ ] Commit `test(stage6): GREEN checkpoint C characterize instability collapse`.

---

# 29. TDD checkpoint D — repeated Collapse reset

Requirement:

```text
EMERGENCY-006
```

Required tests:

```text
TestLabState_SecondCollapseResetsRegeneratedEnergyToZero
TestLabState_SecondCollapseRebasesAtSecondCollapse
TestLabState_SecondCollapseReplacesDeadline
TestLabState_SecondCollapseDoesNotExtendFromOldDeadline
TestResolvePortalLifecycleWithLabEmergency_SecondCollapseMayUseDifferentCause
TestLabState_SecondCollapseAtSameTimestampIsDeterministic
TestLabState_OutOfOrderCollapseRejectsWithoutMutation
TestResolvePortalLifecycleWithLabEmergency_TwoPortalsResetTwice
TestResolvePortalLifecycleWithLabEmergency_ReplayOfSecondPortalIsIdempotent
```

Use two distinct OPEN Portal instances for the aggregate test. Resolve the
first, advance to the second semantic Collapse, and resolve the second.

Execution:

- [ ] Add checkpoint D tests.
- [ ] Run targeted tests and observe any genuine reset/idempotence RED.
- [ ] Commit RED only if behavior is missing.
- [ ] Implement only the minimum chronological replacement behavior.
- [ ] Run targeted tests and full verification.
- [ ] Mark EMERGENCY-006 GREEN with concrete tests/symbols.
- [ ] Commit `test(stage6): GREEN checkpoint D repeated collapse reset` when
      the prior implementation already satisfies the characterization, or a
      RED/GREEN pair when it does not.

---

# 30. TDD checkpoint E — free Close during Override

Requirements:

```text
EMERGENCY-003 Close half
LAB-007 integration
OBSERVER-012 regression
```

Required tests:

```text
TestClosePortalWithLabEnergy_ActiveOverrideCostsZero
TestClosePortalWithLabEnergy_ActiveOverrideAllowsZeroEnergy
TestClosePortalWithLabEnergy_ActiveOverrideDoesNotRebaseEnergy
TestClosePortalWithLabEnergy_ActiveOverridePreservesDeadline
TestClosePortalWithLabEnergy_ActiveOverrideStillRequiresCreatureConfirmation
TestClosePortalWithLabEnergy_ActiveOverrideStillRequiresTransitConfirmation
TestClosePortalWithLabEnergy_ActiveOverrideConfirmedTransitLosesObserver
TestClosePortalWithLabEnergy_ActiveOverrideTerminalPortalStillRejects
TestClosePortalWithLabEnergy_AtOverrideDeadlineChargesNormalCost
TestClosePortalWithLabEnergy_ExpiredOverrideInsufficientIsAtomic
TestClosePortalWithLabEnergy_ExpiredDeadlineIsNotClearedByReadOrCommand
```

Expected RED:

```text
current Stage 5 wrapper returns ErrInsufficientLabEnergy at energy 0
```

Minimal production change:

```go
cost := effectiveCloseCost(lab, now, cfg)
remaining, err := prepareLabEnergySpend(lab, now, cost, cfg)
// delegate unchanged ClosePortalWithObservers
// commit using the same effective cost
```

Execution:

- [ ] Add checkpoint E tests.
- [ ] Observe the energy-0 active-Override RED.
- [ ] Commit `test(stage6): RED checkpoint E free close during override`.
- [ ] Add effective Close cost selection without changing Stage 4 commands.
- [ ] Run targeted and full verification suites.
- [ ] Update EMERGENCY-003, LAB-007 and OBSERVER-012 evidence.
- [ ] Commit `feat(stage6): GREEN checkpoint E free close during override`.

---

# 31. TDD checkpoint F — free Stabilize during Override

Requirements:

```text
EMERGENCY-003 Stabilize half
LAB-008 integration
STABILITY-005 / ENERGY-006..008 regressions
```

Required tests:

```text
TestStabilizePortalWithLabEnergy_ActiveOverrideCostsZero
TestStabilizePortalWithLabEnergy_ActiveOverrideAllowsZeroEnergy
TestStabilizePortalWithLabEnergy_ActiveOverrideDoesNotRebaseLabEnergy
TestStabilizePortalWithLabEnergy_ActiveOverridePreservesDeadline
TestStabilizePortalWithLabEnergy_ActiveOverrideStillRejectsStablePortal
TestStabilizePortalWithLabEnergy_ActiveOverrideStillRejectsOvercharge
TestStabilizePortalWithLabEnergy_ActiveOverrideStillRejectsTerminalPortal
TestStabilizePortalWithLabEnergy_ActiveOverridePreservesPortalBoost
TestStabilizePortalWithLabEnergy_ActiveOverridePreservesPortalDecay
TestStabilizePortalWithLabEnergy_JustBeforeDeadlineIsFree
TestStabilizePortalWithLabEnergy_AtDeadlineChargesNormalCost
TestStabilizePortalWithLabEnergy_ExpiredOverrideInsufficientIsAtomic
```

Expected RED is the Stage 5 cost-20 rejection at Lab Energy 0.

Minimal production change mirrors Close but delegates the unchanged
`Portal.Stabilize`.

Execution:

- [ ] Add checkpoint F tests.
- [ ] Observe RED and commit
      `test(stage6): RED checkpoint F free stabilize during override`.
- [ ] Add effective Stabilize cost selection.
- [ ] Run targeted and complete verification suites.
- [ ] Update EMERGENCY-003, LAB-008, STABILITY-005 and ENERGY-006..008.
- [ ] Commit `feat(stage6): GREEN checkpoint F free stabilize during override`.

---

# 32. TDD checkpoint G — regeneration and Extraction invariant

Requirements:

```text
EMERGENCY-004
EMERGENCY-005
LAB-003 / LAB-009 boundaries
```

Required tests:

```text
TestLabState_LabEnergyRegeneratesDuringOverride
TestLabState_OverrideRegenerationUsesCompletedWholeSeconds
TestLabState_OverrideRegenerationCapsAtMaximum
TestLabState_FreeCloseDoesNotInterruptRegeneration
TestLabState_FreeStabilizeDoesNotInterruptRegeneration
TestLabState_ExtractionCostDuringOverrideIsThirty
TestLabState_ExtractionSpendDuringOverrideAllowsExactThirty
TestLabState_ExtractionSpendDuringOverrideRejectsTwentyNine
TestLabState_ExtractionSpendDuringOverrideDoesNotChangeDeadline
TestLabState_ExtractionRebaseDoesNotChangeActiveWindow
TestLabState_ExtractionCostCharacterizationDoesNotCreatePortal
```

These should be GREEN characterizations over Stage 5 primitives plus Stage 6
active-window logic. Do not add `OpenExtractionPortal` or an Extraction cost
exception.

Execution:

- [ ] Add checkpoint G tests.
- [ ] Run targeted tests; record already-GREEN behavior honestly.
- [ ] Run the complete verification suite.
- [ ] Mark EMERGENCY-005 GREEN.
- [ ] Mark EMERGENCY-004 PARTIAL and retain LAB-009 PARTIAL with an explicit
      Stage 7 remainder.
- [ ] Commit `test(stage6): GREEN checkpoint G regen and extraction invariant`.

---

# 33. TDD checkpoint H — aggregate regressions and stage boundary

Required tests:

```text
TestLabEmergency_OnlyCollapsedTransitionActivatesOverride
TestLabEmergency_ClosedPortalNeverActivatesOverride
TestLabEmergency_AlreadyTerminalPortalIsIdempotent
TestLabEmergency_InvalidTransitionLeavesLabAndPortalUnchanged
TestLabEmergency_OverrideActiveQueryIsPure
TestLabEmergency_OrdinaryCostsRemainAfterExpiry
TestLabEmergency_SendAndRecallRemainZeroCost
TestLabEmergency_CloseFailureDoesNotConsumeRegeneratedEnergy
TestLabEmergency_StabilizeFailureDoesNotConsumeRegeneratedEnergy
TestLabEmergency_LabAndPortalEnergyRemainIndependent
TestLabEmergency_DoesNotMutateObserverUnlessCloseSucceeds
TestLabEmergency_NoExtractionPortalBehavior
```

These may already pass after checkpoints A–G. Manufacture no RED. If an
actual atomicity or boundary defect appears, split it into a real RED/GREEN
pair and record the failure precisely.

Execution:

- [ ] Add checkpoint H characterization tests.
- [ ] Run targeted and full suites.
- [ ] Review every EMERGENCY row and every touched Stage 2–5 row.
- [ ] Commit `test(stage6): GREEN checkpoint H emergency regressions` when no
      production correction is required.

---

# 34. TDD execution protocol

For every checkpoint:

```text
1. Map exact requirement IDs and Final Spec sections.
2. Write focused tests before production code.
3. Run targeted tests and observe genuine RED where behavior is missing.
4. Commit RED evidence.
5. Implement the smallest behavior for that checkpoint.
6. Run targeted tests and observe GREEN.
7. Run all tests and the race suite.
8. Refactor only while GREEN.
9. Update traceability honestly.
10. Commit GREEN implementation/evidence.
```

Checkpoints C/D/G/H may be already-GREEN characterizations. Do not create an
artificial failure merely to produce a RED commit.

---

# 35. Required verification commands

At every meaningful GREEN checkpoint and final completion:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
go test -race -count=1 ./...
```

`gofmt -l .` must print no project Go files.

Additional Stage 6 scope checks:

```bash
rg -n 'time\.Sleep|time\.Now\(\)|time\.NewTicker|time\.After' internal/domain/lab*.go
rg -n 'database/sql|net/http|sync\.|ACTION_REJECTED' internal/domain/lab*.go
rg -n 'OpenExtraction|ExtractionPortal|ExtractionSync' internal/domain/lab*.go
rg -n 'ResolveObserverLifecycle|FirstFreeSlot|NewNaturalPortal' internal/domain/lab_emergency.go
```

Expected output: no matches. References to `cfg.ExtractionCost` are allowed in
Stage 6 characterization tests, but no Extraction behavior symbol may exist.

---

# 36. Targeted execution commands

Use deterministic package runs:

```bash
go test -count=1 ./internal/domain -run 'TestLabState_(ActivateLeylineOverride|LeylineOverride)'
go test -count=1 ./internal/domain -run 'TestResolvePortalLifecycleWithLabEmergency'
go test -count=1 ./internal/domain -run 'TestClosePortalWithLabEnergy_.*Override'
go test -count=1 ./internal/domain -run 'TestStabilizePortalWithLabEnergy_.*Override'
go test -count=1 ./internal/domain -run 'TestLabEmergency|TestLabState_.*Override'
```

If the sandbox cannot write the default Go cache, set a task-specific cache
under `/tmp`; do not alter global environment or repository configuration.

---

# 37. Traceability update expected after Stage 6

Expected statuses:

```text
EMERGENCY-001  GREEN
EMERGENCY-002  GREEN
EMERGENCY-003  GREEN
EMERGENCY-004  PARTIAL  # generic cost invariant only; Stage 7 command absent
EMERGENCY-005  GREEN
EMERGENCY-006  GREEN

LAB-001        GREEN unchanged
LAB-002        PARTIAL unchanged
LAB-003        GREEN, add override regeneration evidence
LAB-004        GREEN unchanged
LAB-005/006    GREEN unchanged
LAB-007/008    GREEN, add effective-cost boundary evidence
LAB-009        PARTIAL unchanged, add override characterization
LAB-010        PARTIAL, override paths complete; Extraction path remains

PORTAL-007/008 remain GREEN with emergency integration evidence
STABILITY-005 remains GREEN
ENERGY-006..008 remain GREEN
OBSERVER-012 remains GREEN
EXTRACTION-* remain PLANNED
```

Every GREEN/PARTIAL row must name concrete tests, implementation symbols and
the exact remaining later-stage boundary.

`docs/requirements.md` remains semantically unchanged unless a real conflict
with Final Spec is discovered.

---

# 38. Worklog update expected

Append to `01_AI_WORKLOG_CURRENT.md`:

- Stage 6 execution agent/model;
- baseline verification result;
- every genuine RED/GREEN pair;
- every already-GREEN characterization;
- approved `Portal.ClosedAt` decision;
- half-open active interval decision;
- copy-then-commit atomicity design;
- expiry-without-write behavior;
- repeated-Collapse behavior;
- effective Close/Stabilize cost behavior;
- Extraction boundary and PARTIAL status;
- any actual implementation mistake or corrective pass;
- final verification and scope-scan results;
- explicit statement that Stage 7 was not started.

Never claim a command passed without observing its output.

---

# 39. Recommended commit history

```text
test(stage6): RED checkpoint A override activation and window
feat(stage6): GREEN checkpoint A override activation and window
test(stage6): RED checkpoint B energy collapse orchestration
feat(stage6): GREEN checkpoint B energy collapse orchestration
test(stage6): GREEN checkpoint C characterize instability collapse
test(stage6): GREEN checkpoint D repeated collapse reset
test(stage6): RED checkpoint E free close during override
feat(stage6): GREEN checkpoint E free close during override
test(stage6): RED checkpoint F free stabilize during override
feat(stage6): GREEN checkpoint F free stabilize during override
test(stage6): GREEN checkpoint G regen and extraction invariant
test(stage6): GREEN checkpoint H emergency regressions
docs(stage6): record TDD evidence and verification
```

If C/D/G/H reveal missing behavior, replace the single characterization commit
with an honest RED/GREEN pair.

---

# 40. Acceptance checklist — activation and time

- [ ] Every new COLLAPSED transition resets Lab Energy to 0.
- [ ] Reset uses the Portal's semantic `ClosedAt`.
- [ ] Energy baseline timestamp becomes that collapse time.
- [ ] Override deadline is collapse time plus configured duration.
- [ ] Default configured duration remains 20 seconds.
- [ ] Active interval includes the exact collapse time.
- [ ] Active interval excludes the exact deadline.
- [ ] Pure active-window reads do not mutate LabState.
- [ ] Expiry does not clear or rewrite the stored deadline.
- [ ] Invalid/backward activation is atomic.
- [ ] Late lifecycle resolution derives accrued regeneration correctly.

---

# 41. Acceptance checklist — lifecycle integration

- [ ] ENERGY_DEPLETED Collapse activates the emergency transition.
- [ ] INSTABILITY Collapse activates the same emergency transition.
- [ ] Existing Stage 2 termination reason/tie semantics remain unchanged.
- [ ] NATURAL_CLOSE does not activate Override.
- [ ] MANUAL_CLOSE does not activate Override.
- [ ] Resolving before a terminal deadline changes nothing.
- [ ] Re-resolving an already-terminal Portal does not reset Lab again.
- [ ] Invalid aggregate state leaves both Portal and Lab unchanged.
- [ ] Nil Portal does not panic or mutate LabState.
- [ ] Portal and Lab updates use copy-then-commit atomicity.

---

# 42. Acceptance checklist — repeated Collapse

- [ ] A second distinct Collapse resets regenerated energy to 0.
- [ ] Second baseline uses the second semantic collapse time.
- [ ] Second deadline replaces rather than extends from the old deadline.
- [ ] ENERGY_DEPLETED and INSTABILITY both participate.
- [ ] Out-of-order old Collapse rejects without moving state backward.
- [ ] Replaying the same terminal Portal is idempotent.

---

# 43. Acceptance checklist — actions

- [ ] Active Override makes Close cost exactly 0.
- [ ] Active Override makes Stabilize cost exactly 0.
- [ ] Free actions work at Lab Energy 0 when all domain restrictions pass.
- [ ] Free actions do not re-baseline Lab Energy.
- [ ] Free actions do not modify the override deadline.
- [ ] Creature and active-transit Close confirmations remain required.
- [ ] Confirmed Close still makes the active Observer LOST.
- [ ] Stable/overcharged/terminal Stabilize restrictions remain.
- [ ] Successful Stabilize retains Portal +15/rebaseline/decay semantics.
- [ ] At the exact Override deadline normal prices apply.
- [ ] After expiry insufficient energy rejects atomically.
- [ ] Outside Override all Stage 5 prices remain unchanged.

---

# 44. Acceptance checklist — regeneration and Extraction boundary

- [ ] Lab Energy regenerates during Override through `CurrentEnergy`.
- [ ] Regeneration still uses completed whole seconds.
- [ ] Regeneration still caps at maximum.
- [ ] Free Close/Stabilize do not interrupt regeneration.
- [ ] Generic Extraction debit remains exactly 30 during Override.
- [ ] Extraction debit does not alter the Override deadline.
- [ ] Extraction re-baseline does not redefine the active interval.
- [ ] No Extraction Portal, synchronization or automatic return exists.
- [ ] EMERGENCY-004 and LAB-009 remain honestly PARTIAL.

---

# 45. Acceptance checklist — quality and scope

- [ ] Completed Stage 2–5 tests remain unchanged and GREEN.
- [ ] No Portal/Observer entity receives Lab Energy fields.
- [ ] No wall-clock calls, sleep, ticker or goroutine added.
- [ ] No Events or temporary callbacks added.
- [ ] No persistence, HTTP, WebSocket or frontend code added.
- [ ] No LabManager or mutex added.
- [ ] `gofmt -l .` prints nothing.
- [ ] `go vet ./...` passes.
- [ ] `go build ./...` passes.
- [ ] `go test -count=1 ./...` passes.
- [ ] `go test -race -count=1 ./...` passes.
- [ ] Traceability is updated honestly.
- [ ] Worklog is updated honestly.
- [ ] Stage 7 was not started.

---

# 46. Explicit non-goals

Do not implement in Stage 6:

```text
OpenExtractionPortal
Extraction synchronization state
automatic first return
Plane selection for Extraction
slot reservation for Extraction
natural spawning
simulation ownership
durable Events
SQLite persistence
API/UI representation of the Override
LabManager synchronization
Tutorial behavior
```

Tests requiring these belong to later detailed stage plans.

---

# 47. Resolved Stage 6 decisions

## S6-D1 — semantic Collapse time

The emergency transition starts at `Portal.ClosedAt`, not at a later resolver
call. This was explicitly approved before this plan was written.

## S6-D2 — half-open interval

Override is active on `[collapseAt, deadline)`. The exact deadline uses normal
costs.

## S6-D3 — deadline-derived start

Historical active-window evaluation derives start as
`deadline - cfg.EmergencyDuration`, not from `EnergyBaseAt`; a paid Extraction
may re-baseline energy without moving the Override window.

## S6-D4 — no expiry write

Expiry is derived. The stored deadline is retained until a later Collapse
replaces it.

## S6-D5 — atomic lifecycle wrapper

Portal and Lab are copied, resolved and committed together. A Lab invariant
error cannot leave a terminal Portal with an unapplied emergency transition.

## S6-D6 — chronological transitions

Collapse timestamps may not precede the current Lab baseline. Future
orchestration must resolve meaningful transitions in nondecreasing semantic
time.

## S6-D7 — Override changes costs only

All Portal, Observer, confirmation and flow restrictions survive unchanged.

## S6-D8 — Extraction remains later

Stage 6 characterizes the unchanged 30-point generic debit but does not expose
or create an Extraction command.

---

# 48. Self-review checklist for the execution agent

Before the first implementation commit:

- [ ] Re-read Final Spec §§14, 21, 22 and 37.
- [ ] Confirm Stage 5 final commit and clean starting state.
- [ ] Confirm no newer detailed stage plan supersedes this file.
- [ ] Confirm every proposed symbol matches this plan.
- [ ] Confirm no test assumes an event, manager or persistence layer.

Before final Stage 6 completion:

- [ ] Search required test names from checkpoints A–H.
- [ ] Review the diff against the Stage 5 final commit.
- [ ] Verify only Stage 6 files and required traceability/worklog changed.
- [ ] Run all required verification and scope commands fresh.
- [ ] Record exact commit hashes and actual outputs.

---

# 49. Stop condition

After every Stage 6 acceptance item is verified:

```text
STOP.
```

Do not begin Stage 7 in the same implementation pass.

Final Stage 6 report must include:

- files changed;
- tests added;
- RED/GREEN commit evidence;
- already-GREEN characterizations;
- semantic-time and half-open-window decisions;
- verification command results;
- traceability changes and remaining PARTIAL rows;
- commit hashes;
- explicit confirmation that Stage 7 was not started.

---

# 50. Next stage after completion

Stage 7:

```text
Extraction Portal via TDD

select Plane with WAITING_RETURN Observer
require regular free slot
charge 30 Lab Energy even during Override
create STABLE / INBOUND / creatures 0 Portal
Portal Energy 60..100, TTL 30..60 sec
5-second synchronization
automatically start only the first return
leave further returns manual
```

Stage 7 must reuse the Stage 5 debit and Stage 6 unchanged-Extraction-cost
boundary without changing completed emergency semantics.
