# Omenpath Research Lab — Stage 7
## Extraction Portal Implementation Plan via TDD

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
> - `08_STAGE_06_EMERGENCY_LEYLINE_OVERRIDE_TDD.md`
> - `docs/superpowers/specs/2026-09-04-stage-7-extraction-design.md`

**Goal:** implement user-selected Extraction Portal opening, its ordinary
30-point Laboratory Energy transaction, five-second synchronization, one
automatic longest-waiting return, and manual-only later returns.

**Architecture:** add a pure Extraction factory, a pure Plane-eligibility
query, an atomic opening wrapper, and a one-shot synchronization resolver.
Store only the approved synchronization-completion marker on `Portal`; reuse
the existing Slot, Lab Energy, longest-waiting, RECALL, Close and Observer
lifecycle primitives without introducing Stage 8+ orchestration.

**Tech stack:** Go, `time.Time`/`time.Duration`, existing `config.Config` and
`random.Random`, `testify/require`, deterministic `testutil` fixtures, and the
repository Git/TDD protocol.

---

# 1. Goal

Stage 7 answers:

```text
Is a selected Plane currently eligible for Extraction?
Can an Extraction Portal use the first regular free Slot and charge 30?
What exact Portal state is created?
When does its five-second synchronization complete?
Which currently waiting Observer receives the single automatic return?
What happens if none remains, or if the Portal closes before/after sync?
```

Required behavior:

```text
open preconditions
  → selected Plane has at least one WAITING_RETURN Observer
  → a regular Slot is free
  → current Laboratory Energy >= 30

created Portal
  → kind EXTRACTION
  → status OPEN
  → stability STABLE
  → flow INBOUND
  → creatures 0
  → energy random 60..100
  → decay random 0.1..1.0
  → TTL random 30..60 sec
  → synchronization marker nil

at openedAt + 5 sec
  → synchronization completes exactly once
  → choose the current longest-waiting Observer in the destination Plane
  → begin one RETURNING transit lasting the existing random 5..15 sec
  → if none remains, complete sync without a return

after synchronization
  → no second automatic return
  → later returns require manual RECALL
```

Reuse rather than duplicate:

```text
Stage 2  Portal model/lifecycle, PortalKindExtraction, FirstFreeSlot
Stage 3  Observer.StartReturning and ResolveObserverLifecycle
Stage 4  LongestWaitingObserverIndex, RecallObserver, Close behavior
Stage 5  LabState derived balance and atomic debit primitives
Stage 6  Leyline Override window with unchanged Extraction price
```

---

# 2. Source-of-truth order

Use the repository hierarchy fixed in `AGENTS.md`:

```text
00_FINAL_SPEC_v5.md
→ 01_AI_WORKLOG_CURRENT.md (history only; never overrides Final Spec)
→ 02_IMPLEMENTATION_ROADMAP_TDD.md
→ approved Stage 7 design document
→ this Stage 7 detailed plan
→ docs/requirements.md
→ tests
→ implementation
```

Final Spec §§5, 14, 19, 20, 21, 22 and 37 control gameplay behavior. The
approved design only resolves omissions needed for deterministic one-shot
synchronization. Stop on a real conflict instead of inventing semantics.

---

# 3. Verified starting point

Stage 6 ended at `a6b5643`; the approved Stage 7 design is `6965d26`. Before
implementation, independently verify both commits, current `git status`,
relevant diffs and the complete Go test baseline.

Existing APIs to preserve:

```go
func NewNaturalPortal(seq, planeID int64, slot int, now time.Time,
    cfg config.Config, rnd random.Random) Portal

func FirstFreeSlot(portals []Portal, maxSlots int) (slot int, ok bool)

func LongestWaitingObserverIndex(observers []Observer, planeID int64) (
    index int, ok bool, err error,
)

func RecallObserver(
    portal *Portal, plane *Plane, observers []Observer, now time.Time,
    confirmUnstable bool, rnd random.Random, cfg config.Config,
) (observerID int64, err error)

func RecallObserverWithLabEnergy(
    lab *LabState, portal *Portal, plane *Plane, observers []Observer,
    now time.Time, confirmUnstable bool, rnd random.Random, cfg config.Config,
) (observerID int64, err error)
```

Stage 5/6 already prove:

```text
cfg.ExtractionCost == 30
SpendEnergy uses derived regeneration and rejects insufficient balance atomically
Extraction remains 30 during Leyline Override
a debit re-baselines energy without moving the override deadline/window
```

No Extraction factory, opening command, synchronization state or automatic
return behavior exists at the starting commit.

---

# 4. Requirements covered

Primary family:

```text
EXTRACTION-001  opening costs 30
EXTRACTION-002  selected Plane requires a WAITING_RETURN Observer
EXTRACTION-003  opening requires a free regular Slot
EXTRACTION-004  EXTRACTION/STABLE/INBOUND/creatures0 parameters
EXTRACTION-005  Portal Energy random 60..100
EXTRACTION-006  Portal TTL random 30..60 sec
EXTRACTION-007  five-second synchronization
EXTRACTION-008  current longest-waiting Observer starts RETURNING after sync
EXTRACTION-009  only one return is automatic
EXTRACTION-010  all further returns are manual
```

Rows enhanced without changing completed semantics:

```text
PORTAL-001/002/003/006
SLOT-003/005/007
LAB-009/010
EMERGENCY-004
OBSERVER-006/007/011/012/013/016
FLOW-003/006/007
```

Expected Stage 7 result:

```text
EXTRACTION-001..010  GREEN
SLOT-007             GREEN
LAB-009              GREEN
LAB-010              GREEN
EMERGENCY-004        GREEN
```

Rows involving UI, simulation ownership, events, persistence or manager-level
uniqueness retain honest later-stage boundaries.

---

# 5. Explicit stage boundary

Stage 7 owns:

```text
pure Extraction factory and Plane eligibility query
ordinary first-free Slot allocation in the opening transaction
30-point opening debit, including during Override
five-second synchronization marker and resolver
one automatic current-longest-waiting return
manual RECALL gate until sync is complete
close/loss integration characterizations
```

Stage 7 does not own:

```text
natural spawn timing or simulation tick
Needs Attention aggregation
Events or ACTION_REJECTED
SQLite/restart recovery
LabManager, mutexes or concurrent serialization
REST/WebSocket or Plane-selection UI
Tutorial scripting
```

Do not start Stage 8.

---

# 6. Approved domain APIs

Add:

```go
func NewExtractionPortal(
    seq, planeID int64, slot int, now time.Time,
    cfg config.Config, rnd random.Random,
) Portal

func ExtractionPlaneEligible(
    observers []Observer, planeID int64, now time.Time,
) (bool, error)

func OpenExtractionPortal(
    lab *LabState, portals *[]Portal, plane *Plane, observers []Observer,
    seq int64, now time.Time, rnd random.Random, cfg config.Config,
) (portalID int64, err error)

func ResolveExtractionSynchronization(
    portal *Portal, plane *Plane, observers []Observer, now time.Time,
    rnd random.Random, cfg config.Config,
) (observerID int64, changed bool, err error)
```

Add stable errors:

```go
ErrExtractionInvariant     = errors.New("extraction invariant violated")
ErrExtractionSynchronizing = errors.New("extraction portal is synchronizing")
```

Do not add a service, manager, repository, goroutine or callback.

---

# 7. Portal model extension

Extend `Portal` with the approved marker:

```go
ExtractionSynchronizedAt *time.Time
```

Canonical meaning:

```text
NATURAL Portal: nil for its complete lifecycle
OPEN EXTRACTION before sync: nil
EXTRACTION after one-shot sync attempt: OpenedAt + cfg.ExtractionSync
```

The marker records completion, not an Observer reservation or return
completion. No new Observer status is allowed; do not add
`WAITING_EXTRACTION`.

---

# 8. Pure factory contract

Deterministic draw order:

```text
1. TTL seconds     IntInclusive(ExtractionTTLMin, ExtractionTTLMax)
2. initial energy FloatRange(ExtractionEnergyMin, ExtractionEnergyMax)
3. decay rate     FloatRange(PortalDecayMin, PortalDecayMax)
```

Canonical output:

```go
Portal{
    ID: seq, Name: fmt.Sprintf("Omenpath #%04d", seq),
    SlotIndex: slot, Kind: PortalKindExtraction,
    DestinationPlaneID: planeID,
    EnergyBase: energy, EnergyBaseAt: now, EnergyDecayRate: decay,
    Stability: PortalStable, InstabilityCollapseAt: nil,
    OpenedAt: now, ScheduledCloseAt: now.Add(ttl),
    CreaturesInitial: 0, ObserverFlow: PortalFlowInbound,
    Status: PortalStatusOpen, TerminationReason: TerminationNone,
    ExtractionSynchronizedAt: nil,
    CreatedAt: now, UpdatedAt: now,
}
```

The factory does not select Plane/Slot, inspect Observers, charge Lab Energy,
append shared state, emit Events or resolve lifecycle.

---

# 9. Plane eligibility contract

`ExtractionPlaneEligible`:

```text
validates the complete Observer roster at supplied now
filters only WAITING_RETURN
requires canonical CurrentPlaneID and PhaseStartedAt
matches exact Plane ID
returns true when one or more matches exist
returns false,nil when valid but none match
returns false,ErrObserverInvariant for malformed/duplicate state
never mutates or sorts the slice
```

Extract the existing Stage 4 roster-validation loop into an unexported
`validateObserverRoster(observers, now)` helper and call it from both
`validateObserverCommandAggregate` and `ExtractionPlaneEligible`. Stage 4
admission order and error identity must remain unchanged.

---

# 10. Opening transaction

Execute in this exact order:

```text
1. validate LabState at now with zero-cost preflight
2. validate portals pointer, selected Plane and Extraction config
3. validate Observer roster and selected Plane eligibility
4. validate Portal IDs/open Slot occupancy needed for safe append
5. find FirstFreeSlot(portals, cfg.MaxActivePortals)
6. preflight cfg.ExtractionCost against current derived Lab Energy
7. call NewExtractionPortal (first random consumption)
8. build copied next Lab/Portal slice
9. commit both together
10. return new Portal ID
```

Rejection precedence:

```text
malformed Lab/aggregate
→ selected Plane has no waiting Observer
→ no free Slot
→ insufficient Lab Energy
→ success
```

Opening does not mutate, select or reserve an Observer.

---

# 11. Slot and Laboratory Energy behavior

Extraction uses the ordinary Slot pool:

```text
OPEN occupies a Slot
CLOSED/COLLAPSED releases it
smallest free index wins
7/7 OPEN rejects with ErrNoFreePortalSlot
existing Portals retain order and SlotIndex
success appends exactly one Portal
```

Effective Extraction cost is always `cfg.ExtractionCost`. There is no
Override discount.

Required energy behavior:

```text
exactly 30 succeeds and leaves 0
29 rejects atomically
derived regenerated energy is spendable
success re-baselines Lab Energy at now
LeylineOverrideUntil is unchanged
the deadline-derived Override interval is unchanged
no rejected action consumes randomness or appends a Portal
```

Portal Energy is a separate float percentage and never pays Lab cost.

---

# 12. Synchronization and current selection

```go
syncAt := portal.OpenedAt.Add(cfg.ExtractionSync)
```

The synchronization interval is `[OpenedAt, syncAt)`. Before `syncAt`, the
resolver is a pure no-op. At or after `syncAt`, it uses `syncAt`, not late
`now`, as both marker and Observer phase start.

At synchronization, select again from the current roster:

```text
status == WAITING_RETURN
CurrentPlaneID == portal.DestinationPlaneID
earliest PhaseStartedAt wins
exact tie → lowest Observer ID
```

Do not remember the opening-time Observer. If that Observer left through a
different Portal, choose the current next longest-waiting Observer.

With a selected Observer, commit together:

```text
Portal.ExtractionSynchronizedAt = syncAt
Portal.UpdatedAt                = syncAt
Observer.Status                 = RETURNING
Observer.ActivePortalID         = Portal.ID
Observer.PhaseStartedAt         = syncAt
Observer.PhaseEndsAt            = syncAt + one random 5..15 sec duration
Observer.CurrentPlaneID         remains the destination Plane
```

The return transit is separate: successful arrival is 10..20 seconds after
opening under default configuration.

---

# 13. No-wait and one-shot behavior

If no matching Observer remains at synchronization:

```text
set ExtractionSynchronizedAt = syncAt
set Portal.UpdatedAt = syncAt
return observerID=0, changed=true, err=nil
consume no transit-duration random value
do not refund Lab Energy
leave Portal OPEN/STABLE/INBOUND
```

An Observer that becomes `WAITING_RETURN` later is not automatic.

Once the marker equals expected `syncAt`, every replay returns unchanged and
consumes no randomness. A non-nil marker with another timestamp is malformed
and returns `ErrExtractionInvariant` atomically. The marker never clears after
return, Close or Collapse.

---

# 14. Manual RECALL gate

Extend `RecallObserver` after its OPEN check:

```go
if portal.Kind == PortalKindExtraction &&
    portal.ExtractionSynchronizedAt == nil {
    return 0, ErrExtractionSynchronizing
}
```

Consequences:

```text
closed pre-sync Portal → ErrPortalNotOpen
OPEN pre-sync Portal → ErrExtractionSynchronizing
exact syncAt before resolver → ErrExtractionSynchronizing
after completed sync → existing RECALL rules
NATURAL Portal → behavior unchanged
```

The energy-aware RECALL wrapper inherits the same gate and remains cost 0.
Future Stage 8/11 orchestration must resolve sync before a same-time command.

---

# 15. Resolver implementation shape

Use copy-then-commit:

```text
validate Portal/Plane/roster/config
terminal Portal → unchanged no-op
valid completed marker → unchanged no-op
before syncAt → unchanged no-op

copy Portal and Observer slice
set marker on copied Portal
delegate automatic return through existing RECALL behavior at syncAt
ErrNoWaitingObserver → commit marker only
any other error → commit nothing
success → commit marker and selected Observer together
```

Setting the marker on the copy lets delegated RECALL pass the sync gate while
retaining Stage 4 direction, risk, busy, selection and random behavior.

Default Extraction parameters guarantee the Portal is not CRITICAL at the
five-second deadline. Do not add an automatic-return bypass for existing
RECALL restrictions.

---

# 16. Close and loss integration

Close during synchronization:

```text
Portal → CLOSED / MANUAL_CLOSE through existing wrapper
Observers remain unchanged, including WAITING_RETURN
later synchronization resolution is a terminal no-op
```

Close during automatic RETURNING:

```text
confirm=false → ErrConfirmationRequired, no mutation
confirm=true  → Portal CLOSED and active Observer LOST at close now
```

Existing `ResolveObserverLifecycle` handles a Portal becoming terminal before
the return deadline. Exact `ClosedAt == PhaseEndsAt` remains successful. Other
waiting Observers remain in the Plane. Add no Extraction-specific LOST path.

---

# 17. Structural validation and errors

Add:

```text
ErrExtractionInvariant
ErrExtractionSynchronizing
```

Reuse:

```text
ErrNoWaitingObserver
ErrNoFreePortalSlot
ErrInsufficientLabEnergy
ErrLabEnergyInvariant
ErrObserverInvariant
ErrPortalNotOpen
ErrPortalBusy
ErrPortalCriticalRisk
ErrConfirmationRequired
```

Reject before mutation/random use:

```text
nil Lab/Portal slice/Plane where required
invalid Lab baseline or backward now
duplicate Observer IDs or noncanonical Observer state
duplicate new Portal ID
duplicate/out-of-range OPEN Slot occupancy
invalid max slots or Extraction ranges/duration/cost
nil random dependency on a path that must draw
wrong Portal kind/destination or malformed marker at sync
```

Do not attach HTTP semantics to domain errors.

---

# 18. Time and randomness

All production APIs accept `now`. No Stage 7 file may call:

```text
time.Now
time.Sleep
time.After
time.NewTicker
```

Opening consumes exactly one TTL int followed by Energy and decay floats.
A successful automatic or manual return consumes one transit int through
`Observer.StartReturning`. Rejections, pre-sync calls, no-wait sync and replay
consume nothing.

---

# 19. No Event semantics

Do not emit:

```text
EXTRACTION_PORTAL_OPENED
EXTRACTION_SYNCHRONIZED
OBSERVER_RETURN_STARTED
OBSERVER_RETURNED
OBSERVER_LOST
ACTION_REJECTED
```

Stage 9 owns durable Events.

---

# 20. Production file map

Create:

```text
internal/domain/extraction.go
```

It owns the four approved Stage 7 APIs and Extraction-specific validation.

Modify:

```text
internal/domain/portal.go             add synchronization marker
internal/domain/errors.go             add two stable errors
internal/domain/observer_commands.go  extract roster validation; add recall gate
```

Do not move Stage 2–6 code or rename its public APIs.

---

# 21. Test file map

Create:

```text
internal/domain/extraction_factory_test.go
internal/domain/extraction_plane_test.go
internal/domain/extraction_open_test.go
internal/domain/extraction_sync_test.go
internal/domain/extraction_auto_return_test.go
internal/domain/extraction_manual_recall_test.go
internal/domain/extraction_close_test.go
internal/domain/extraction_integration_test.go
```

Reuse clear package-level fixtures (`labAt`, `overrideLab`, `stage4Plane`,
`waitingObserver`). Add focused test-only `extractionPortal` and
`waitingObserverInPlane` helpers. Never use production commands to manufacture
the initial state under test.

---

# 22. Fixed fixtures

```text
base time              testutil.BaseTime
selected Plane ID      7
other Plane ID         8
Portal sequence/ID     42
default free Slot      1
Extraction cost        30
sync duration          5 sec
return duration        explicit FakeRandom int in 5..15
Portal Energy bounds   60 / 100
TTL bounds             30 / 60 sec
creatures              0
```

Use exact boundaries, not sleeps or tolerances.

---

# 23. Baseline protocol

Before checkpoint A:

- [ ] Read this complete plan and `AGENTS.md`.
- [ ] Re-read Final Spec §§5, 14, 19, 20, 21, 22 and 37.
- [ ] Verify commits `a6b5643` and `6965d26`.
- [ ] Inspect `git status`, recent log and Stage 6 production diff.
- [ ] Run the complete verification suite and record the green baseline.
- [ ] Confirm no Stage 7 production symbol already exists.

---

# 24. TDD execution protocol

For every checkpoint:

```text
1. Map exact requirement IDs and Final Spec sections.
2. Add only the focused tests for this checkpoint.
3. Run targeted tests and observe genuine RED where behavior is absent.
4. Commit RED evidence before production code.
5. Implement the smallest behavior required by this checkpoint.
6. Run targeted tests and observe GREEN.
7. Run gofmt/vet/build/full tests/race.
8. Refactor only while GREEN.
9. Update traceability honestly.
10. Commit the GREEN implementation/evidence.
```

Do not manufacture RED for already-correct integration characterizations.

---

# 25. Checkpoint dependency graph

```text
A factory/model
    ↓
B Plane eligibility / shared roster validation
    ↓
C opening transaction / Slot / cost
    ↓
D synchronization timing and marker
    ↓
E automatic current-longest return
    ↓
F one-shot and manual RECALL
    ↓
G Close / LOST integration
    ↓
H aggregate regressions and scope
```

Do not combine A–F into one commit: each contains independently observable
missing behavior.

---

# 26. Test helper contracts

Define test-only helpers with explicit state:

```go
func waitingObserverInPlane(
    id, planeID int64,
    waitingSince time.Time,
) domain.Observer

func extractionPortal(now time.Time, cfg config.Config) domain.Portal

func extractionFactoryRandom(cfg config.Config) *testutil.FakeRandom {
    return testutil.NewFakeRandom().
        QueueInt(int(cfg.ExtractionTTLMin.Seconds())).
        QueueFloat(cfg.ExtractionEnergyMin, cfg.PortalDecayMin)
}
```

`waitingObserverInPlane` sets canonical `WAITING_RETURN` fields. The
`extractionPortal` fixture is OPEN/STABLE/INBOUND/creature-free with a nil
marker. Tests needing completed sync explicitly set the marker to
`OpenedAt + cfg.ExtractionSync`.

---

# 27. TDD checkpoint A — Portal marker and pure factory

Requirements:

```text
PORTAL-003
EXTRACTION-004
EXTRACTION-005
EXTRACTION-006
```

Files:

```text
modify internal/domain/portal.go
create internal/domain/extraction.go
create internal/domain/extraction_factory_test.go
```

Required tests:

```text
TestNewExtractionPortal_SetsExtractionKind
TestNewExtractionPortal_StartsOpen
TestNewExtractionPortal_IsStableWithoutHiddenCollapse
TestNewExtractionPortal_StartsInbound
TestNewExtractionPortal_HasNoCreatures
TestNewExtractionPortal_UsesSelectedPlaneAndSlot
TestNewExtractionPortal_UsesSequentialIdentity
TestNewExtractionPortal_AcceptsMinimumEnergy
TestNewExtractionPortal_AcceptsMaximumEnergy
TestNewExtractionPortal_AcceptsMinimumDecay
TestNewExtractionPortal_AcceptsMaximumDecay
TestNewExtractionPortal_AcceptsMinimumTTL
TestNewExtractionPortal_AcceptsMaximumTTL
TestNewExtractionPortal_DrawsTTLThenEnergyThenDecay
TestNewExtractionPortal_InitializesTimestamps
TestNewExtractionPortal_StartsUnsynchronized
TestNewExtractionPortal_DoesNotMutateUnrelatedState
```

Representative test:

```go
func TestNewExtractionPortal_StartsUnsynchronized(t *testing.T) {
    cfg := config.Default()
    rnd := testutil.NewFakeRandom().
        QueueInt(int(cfg.ExtractionTTLMin.Seconds())).
        QueueFloat(cfg.ExtractionEnergyMin, cfg.PortalDecayMin)

    portal := domain.NewExtractionPortal(
        42, 7, 3, testutil.BaseTime, cfg, rnd,
    )

    require.Equal(t, domain.PortalKindExtraction, portal.Kind)
    require.Equal(t, domain.PortalStable, portal.Stability)
    require.Equal(t, domain.PortalFlowInbound, portal.ObserverFlow)
    require.Nil(t, portal.ExtractionSynchronizedAt)
}
```

Execution:

- [ ] Add only checkpoint A tests.
- [ ] Run the targeted package and observe compile RED for the missing field
      and factory.
- [ ] Commit `test(stage7): RED checkpoint A extraction factory`.
- [ ] Add the marker to `Portal` and minimal `NewExtractionPortal`.
- [ ] Keep `NewNaturalPortal` behavior and all existing fixtures unchanged.
- [ ] Run targeted tests and observe GREEN.
- [ ] Run the complete required verification suite.
- [ ] Add factory evidence to PORTAL-003 and EXTRACTION-004..006.
- [ ] Commit `feat(stage7): GREEN checkpoint A extraction factory`.

---

# 28. TDD checkpoint B — Plane eligibility

Requirements:

```text
EXTRACTION-002
OBSERVER-016 reuse
```

Files:

```text
modify internal/domain/extraction.go
modify internal/domain/observer_commands.go
create internal/domain/extraction_plane_test.go
```

Required tests:

```text
TestExtractionPlaneEligible_ReturnsTrueForWaitingObserver
TestExtractionPlaneEligible_ReturnsFalseForEmptyRoster
TestExtractionPlaneEligible_ReturnsFalseWithoutWaitingObservers
TestExtractionPlaneEligible_FiltersBySelectedPlane
TestExtractionPlaneEligible_AcceptsMultipleWaitingObservers
TestExtractionPlaneEligible_IgnoresAvailableObserver
TestExtractionPlaneEligible_IgnoresExploringObserver
TestExtractionPlaneEligible_IgnoresReturningObserver
TestExtractionPlaneEligible_IgnoresLostObserver
TestExtractionPlaneEligible_RejectsWaitingWithoutPlane
TestExtractionPlaneEligible_RejectsWaitingWithoutTimestamp
TestExtractionPlaneEligible_RejectsDuplicateObserverIDs
TestExtractionPlaneEligible_RejectsStaleTransitState
TestExtractionPlaneEligible_DoesNotMutateOrReorderRoster
TestObserverCommandValidation_RefactorPreservesExistingErrors
```

Representative test:

```go
func TestExtractionPlaneEligible_FiltersBySelectedPlane(t *testing.T) {
    observers := []domain.Observer{
        waitingObserverInPlane(1, 8, testutil.BaseTime.Add(-time.Minute)),
    }

    eligible, err := domain.ExtractionPlaneEligible(
        observers, 7, testutil.BaseTime,
    )

    require.NoError(t, err)
    require.False(t, eligible)
}
```

Execution:

- [ ] Add only checkpoint B tests and observe undefined-helper RED.
- [ ] Commit `test(stage7): RED checkpoint B extraction plane eligibility`.
- [ ] Extract `validateObserverRoster` without changing Stage 4 behavior.
- [ ] Implement `ExtractionPlaneEligible` using shared validation and current
      `WAITING_RETURN` state.
- [ ] Run all Stage 4 Observer command tests plus checkpoint B.
- [ ] Run the complete required verification suite.
- [ ] Mark EXTRACTION-002 GREEN and extend OBSERVER-016 evidence.
- [ ] Commit `feat(stage7): GREEN checkpoint B extraction plane eligibility`.

---

# 29. TDD checkpoint C — atomic opening, Slot and cost

Requirements:

```text
EXTRACTION-001
EXTRACTION-003
SLOT-007
LAB-009
LAB-010
EMERGENCY-004
```

Files:

```text
modify internal/domain/extraction.go
modify internal/domain/errors.go
create internal/domain/extraction_open_test.go
```

Required tests:

```text
TestOpenExtractionPortal_SucceedsForSelectedEligiblePlane
TestOpenExtractionPortal_ReturnsCreatedPortalID
TestOpenExtractionPortal_AppendsExactlyOnePortal
TestOpenExtractionPortal_PreservesExistingPortalOrder
TestOpenExtractionPortal_UsesFirstFreeRegularSlot
TestOpenExtractionPortal_ReusesTerminalPortalSlot
TestOpenExtractionPortal_RejectsAllSevenSlotsOccupied
TestOpenExtractionPortal_RejectsNoWaitingObserverInSelectedPlane
TestOpenExtractionPortal_RejectsWaitingObserverOnlyInOtherPlane
TestOpenExtractionPortal_ChargesThirty
TestOpenExtractionPortal_AllowsExactThirty
TestOpenExtractionPortal_UsesRegeneratedEnergy
TestOpenExtractionPortal_ChargesThirtyDuringOverride
TestOpenExtractionPortal_PreservesOverrideDeadline
TestOpenExtractionPortal_RejectsTwentyNine
TestOpenExtractionPortal_InsufficientEnergyLeavesLabUnchanged
TestOpenExtractionPortal_InsufficientEnergyLeavesPortalsUnchanged
TestOpenExtractionPortal_InsufficientEnergyLeavesObserversUnchanged
TestOpenExtractionPortal_RejectionDoesNotConsumeRandom
TestOpenExtractionPortal_InvalidLabIsAtomic
TestOpenExtractionPortal_NilPlaneIsAtomic
TestOpenExtractionPortal_NilPortalCollectionIsAtomic
TestOpenExtractionPortal_RejectsDuplicatePortalID
TestOpenExtractionPortal_RejectsDuplicateOpenSlot
TestOpenExtractionPortal_RejectsOutOfRangeOpenSlot
TestOpenExtractionPortal_InvalidConfigIsAtomic
TestOpenExtractionPortal_NilRandomIsAtomic
TestOpenExtractionPortal_DoesNotSelectOrReserveObserver
```

Representative cost test:

```go
func TestOpenExtractionPortal_ChargesThirtyDuringOverride(t *testing.T) {
    cfg := config.Default()
    cfg.LabRegenPerSec = cfg.ExtractionCost
    lab := overrideLab(0, testutil.BaseTime, cfg)
    deadline := *lab.LeylineOverrideUntil
    now := testutil.BaseTime.Add(time.Second)
    portals := []domain.Portal{}
    plane := stage4Plane()
    observers := []domain.Observer{
        waitingObserverInPlane(1, plane.ID, testutil.BaseTime.Add(-time.Minute)),
    }
    rnd := extractionFactoryRandom(cfg)

    _, err := domain.OpenExtractionPortal(
        &lab, &portals, &plane, observers, 42,
        now, rnd, cfg,
    )

    require.NoError(t, err)
    require.Zero(t, lab.EnergyBase)
    require.Equal(t, deadline, *lab.LeylineOverrideUntil)
}
```

Execution:

- [ ] Add checkpoint C tests and observe undefined-command/error RED.
- [ ] Commit `test(stage7): RED checkpoint C atomic extraction opening`.
- [ ] Implement opening preflight, regular Slot allocation and copy-then-commit.
- [ ] Reuse the factory and Stage 5 positive debit; do not add a discounted
      Extraction cost helper.
- [ ] Ensure every rejection occurs before random consumption.
- [ ] Run targeted tests and observe GREEN.
- [ ] Run the complete required verification suite.
- [ ] Mark EXTRACTION-001/003, SLOT-007, LAB-009/010 and EMERGENCY-004 GREEN.
- [ ] Commit `feat(stage7): GREEN checkpoint C atomic extraction opening`.

---

# 30. TDD checkpoint D — synchronization timing and marker

Requirements:

```text
EXTRACTION-007
EXTRACTION-009 marker primitive
```

Files:

```text
modify internal/domain/extraction.go
create internal/domain/extraction_sync_test.go
```

Required tests:

```text
TestResolveExtractionSynchronization_BeforeDeadlineChangesNothing
TestResolveExtractionSynchronization_AtDeadlineCompletes
TestResolveExtractionSynchronization_AfterDeadlineCompletes
TestResolveExtractionSynchronization_LateResolutionUsesSemanticDeadline
TestResolveExtractionSynchronization_SetsPortalUpdatedAtToDeadline
TestResolveExtractionSynchronization_DefaultDeadlineIsFiveSeconds
TestResolveExtractionSynchronization_EmptyWaitingRosterStillCompletes
TestResolveExtractionSynchronization_EmptyRosterConsumesNoRandom
TestResolveExtractionSynchronization_NaturalPortalRejects
TestResolveExtractionSynchronization_NilPortalRejects
TestResolveExtractionSynchronization_NilPlaneRejects
TestResolveExtractionSynchronization_MismatchedPlaneRejects
TestResolveExtractionSynchronization_NonPositiveDurationRejects
TestResolveExtractionSynchronization_TerminalBeforeSyncIsNoOp
TestResolveExtractionSynchronization_FailureIsAtomic
```

Representative boundary test:

```go
func TestResolveExtractionSynchronization_AtDeadlineCompletes(t *testing.T) {
    cfg := config.Default()
    portal := extractionPortal(testutil.BaseTime, cfg)
    plane := stage4Plane()
    syncAt := portal.OpenedAt.Add(cfg.ExtractionSync)

    observerID, changed, err := domain.ResolveExtractionSynchronization(
        &portal, &plane, nil, syncAt, testutil.NewFakeRandom(), cfg,
    )

    require.NoError(t, err)
    require.True(t, changed)
    require.Zero(t, observerID)
    require.Equal(t, syncAt, *portal.ExtractionSynchronizedAt)
}
```

Execution:

- [ ] Add checkpoint D tests and observe undefined-resolver RED.
- [ ] Commit `test(stage7): RED checkpoint D extraction synchronization`.
- [ ] Implement validation, half-open timing, semantic marker and terminal
      no-op only; automatic return arrives in checkpoint E.
- [ ] Use copy-then-commit even for marker-only completion.
- [ ] Run targeted and complete verification suites.
- [ ] Mark EXTRACTION-007 GREEN and EXTRACTION-009 PARTIAL.
- [ ] Commit `feat(stage7): GREEN checkpoint D extraction synchronization`.

---

# 31. TDD checkpoint E — automatic current-longest return

Requirements:

```text
EXTRACTION-008
OBSERVER-006
OBSERVER-007
OBSERVER-016
FLOW-003
```

Files:

```text
modify internal/domain/extraction.go
create internal/domain/extraction_auto_return_test.go
```

Required tests:

```text
TestResolveExtractionSynchronization_StartsLongestWaitingReturn
TestResolveExtractionSynchronization_FiltersObserversByDestinationPlane
TestResolveExtractionSynchronization_BreaksWaitingTieByLowestID
TestResolveExtractionSynchronization_SelectsCurrentRosterAtSync
TestResolveExtractionSynchronization_OriginalLongestGoneSelectsNextWaiting
TestResolveExtractionSynchronization_IgnoresOriginalLongestReturningElsewhere
TestResolveExtractionSynchronization_SetsReturningStatus
TestResolveExtractionSynchronization_PreservesCurrentPlaneDuringTransit
TestResolveExtractionSynchronization_SetsActivePortalID
TestResolveExtractionSynchronization_UsesSyncDeadlineAsPhaseStart
TestResolveExtractionSynchronization_DrawsConfiguredTransitDuration
TestResolveExtractionSynchronization_DrawsTransitDurationExactlyOnce
TestResolveExtractionSynchronization_CommitsMarkerAndObserverTogether
TestResolveExtractionSynchronization_LateStartUsesSemanticSyncTime
TestResolveExtractionSynchronization_OtherWaitingObserversRemainUnchanged
TestResolveExtractionSynchronization_InvalidObserverStateIsAtomic
TestResolveExtractionSynchronization_BusyPortalIsAtomic
TestResolveExtractionSynchronization_NilRandomWithWaitingObserverIsAtomic
```

Representative changed-roster test:

```go
func TestResolveExtractionSynchronization_OriginalLongestGoneSelectsNextWaiting(t *testing.T) {
    cfg := config.Default()
    portal := extractionPortal(testutil.BaseTime, cfg)
    plane := stage4Plane()
    gone := waitingObserverInPlane(1, plane.ID, testutil.BaseTime.Add(-2*time.Minute))
    gone.Status = domain.ObserverReturning
    otherPortalID := int64(99)
    transitStart := testutil.BaseTime
    transitEnd := testutil.BaseTime.Add(10 * time.Second)
    gone.ActivePortalID = &otherPortalID
    gone.PhaseStartedAt = &transitStart
    gone.PhaseEndsAt = &transitEnd
    remaining := waitingObserverInPlane(2, plane.ID, testutil.BaseTime.Add(-time.Minute))
    observers := []domain.Observer{gone, remaining}
    syncAt := portal.OpenedAt.Add(cfg.ExtractionSync)

    id, changed, err := domain.ResolveExtractionSynchronization(
        &portal, &plane, observers, syncAt,
        testutil.NewFakeRandom().QueueInt(10), cfg,
    )

    require.NoError(t, err)
    require.True(t, changed)
    require.Equal(t, int64(2), id)
    require.Equal(t, domain.ObserverReturning, observers[1].Status)
}
```

Execution:

- [ ] Add checkpoint E tests and run targeted RED: marker completes but no
      Observer starts RETURNING.
- [ ] Commit `test(stage7): RED checkpoint E automatic extraction return`.
- [ ] On copied state, set the marker then delegate through existing RECALL at
      semantic `syncAt`; swallow only `ErrNoWaitingObserver` as marker-only
      success.
- [ ] Commit Portal and Observer together only after the delegated result.
- [ ] Run targeted tests and observe GREEN.
- [ ] Run the complete required verification suite.
- [ ] Mark EXTRACTION-008 GREEN and extend Observer/Flow evidence.
- [ ] Commit `feat(stage7): GREEN checkpoint E automatic extraction return`.

---

# 32. TDD checkpoint F — one-shot and manual-only later returns

Requirements:

```text
EXTRACTION-009
EXTRACTION-010
FLOW-006
FLOW-007
```

Files:

```text
modify internal/domain/errors.go
modify internal/domain/observer_commands.go
create internal/domain/extraction_manual_recall_test.go
```

Required tests:

```text
TestResolveExtractionSynchronization_ReplayDoesNotStartSecondObserver
TestResolveExtractionSynchronization_ReplayDoesNotConsumeRandom
TestResolveExtractionSynchronization_ObserverAddedLaterIsNotAutomatic
TestResolveExtractionSynchronization_MalformedMarkerRejectsAtomically
TestRecallObserver_ExtractionBeforeSyncRejects
TestRecallObserver_ExtractionBeforeSyncRejectionIsAtomic
TestRecallObserver_ExtractionBeforeSyncConsumesNoRandom
TestRecallObserver_ExtractionAtDeadlineBeforeResolverStillRejects
TestRecallObserver_ExtractionAfterSyncAllowsManualReturn
TestRecallObserver_ExtractionAfterSyncSelectsLongestWaiting
TestRecallObserverWithLabEnergy_InheritsExtractionSyncGate
TestRecallObserverWithLabEnergy_AfterSyncRemainsZeroCost
TestExtractionPortal_AutomaticTransitBlocksManualRecallWhileBusy
TestExtractionPortal_AfterAutomaticReturnAllowsNextManualRecall
TestExtractionPortal_OnlyFirstReturnIsAutomatic
TestRecallObserver_NaturalPortalBehaviorUnchangedByExtractionGate
TestRecallObserver_ClosedUnsynchronizedExtractionReportsNotOpen
```

Representative one-shot test:

```go
func TestResolveExtractionSynchronization_ReplayDoesNotStartSecondObserver(t *testing.T) {
    cfg := config.Default()
    portal := extractionPortal(testutil.BaseTime, cfg)
    plane := stage4Plane()
    observers := []domain.Observer{
        waitingObserverInPlane(1, plane.ID, testutil.BaseTime.Add(-2*time.Minute)),
        waitingObserverInPlane(2, plane.ID, testutil.BaseTime.Add(-time.Minute)),
    }
    syncAt := portal.OpenedAt.Add(cfg.ExtractionSync)

    _, changed, err := domain.ResolveExtractionSynchronization(
        &portal, &plane, observers, syncAt,
        testutil.NewFakeRandom().QueueInt(5), cfg,
    )
    require.NoError(t, err)
    require.True(t, changed)

    _, changed, err = domain.ResolveExtractionSynchronization(
        &portal, &plane, observers, syncAt.Add(time.Second),
        testutil.NewFakeRandom(), cfg,
    )
    require.NoError(t, err)
    require.False(t, changed)
    require.Equal(t, domain.ObserverWaitingReturn, observers[1].Status)
}
```

Execution:

- [ ] Add checkpoint F tests and observe genuine RED for the missing manual
      gate; one-shot resolver tests may already be GREEN after D/E.
- [ ] Commit `test(stage7): RED checkpoint F manual recall sync gate`.
- [ ] Add the stable error and minimal Extraction-only guard after OPEN check.
- [ ] Do not change Natural Portal admission or post-sync RECALL behavior.
- [ ] Run all Stage 4/5 RECALL tests plus checkpoint F.
- [ ] Run the complete required verification suite.
- [ ] Mark EXTRACTION-009/010 GREEN and extend FLOW-006/007.
- [ ] Commit `feat(stage7): GREEN checkpoint F manual recall sync gate`.

If every checkpoint F test is already GREEN because the guard was minimally
required in checkpoint E, record one honest GREEN characterization commit
instead of reverting working code to manufacture RED.

---

# 33. TDD checkpoint G — Close and LOST integration

Requirements:

```text
PORTAL-006
OBSERVER-012
OBSERVER-013
EXTRACTION-007..010 integration
```

Files:

```text
create internal/domain/extraction_close_test.go
```

Required tests:

```text
TestExtractionClose_DuringSyncClosesPortal
TestExtractionClose_DuringSyncLeavesWaitingObserverUnchanged
TestExtractionClose_DuringSyncDoesNotSetSynchronizationMarker
TestExtractionClose_ResolverAfterPreSyncCloseIsNoOp
TestExtractionClose_DuringReturningRequiresConfirmation
TestExtractionClose_RejectedConfirmationIsAtomic
TestExtractionClose_ConfirmedReturningBecomesLost
TestExtractionClose_ConfirmedSetsManualCloseReason
TestExtractionClose_ConfirmedUsesSameTimestampForPortalAndLoss
TestExtractionLifecycle_CollapseDuringReturningBecomesLost
TestExtractionLifecycle_OtherWaitingObserversRemainInPlane
TestExtractionLifecycle_CloseAtReturnDeadlineSucceedsReturn
TestExtractionLifecycle_CloseAfterReturnDoesNotRetroactivelyLoseObserver
TestExtractionClose_DuringSyncUsesOrdinaryCloseCost
TestExtractionClose_DuringOverrideUsesFreeCloseRule
```

Representative pre-sync close test:

```go
func TestExtractionClose_DuringSyncLeavesWaitingObserverUnchanged(t *testing.T) {
    cfg := config.Default()
    lab := labAt(100, testutil.BaseTime)
    portal := extractionPortal(testutil.BaseTime, cfg)
    plane := stage4Plane()
    observers := []domain.Observer{
        waitingObserverInPlane(1, plane.ID, testutil.BaseTime.Add(-time.Minute)),
    }
    before := append([]domain.Observer(nil), observers...)

    err := domain.ClosePortalWithLabEnergy(
        &lab, &portal, &plane, observers,
        testutil.BaseTime.Add(time.Second), false, cfg,
    )

    require.NoError(t, err)
    require.Equal(t, before, observers)
    require.Equal(t, domain.PortalStatusClosed, portal.Status)
    require.Nil(t, portal.ExtractionSynchronizedAt)
}
```

Execution:

- [ ] Add checkpoint G integration characterizations.
- [ ] Run targeted tests. They should be GREEN from completed Close, Observer
      lifecycle and synchronization behavior.
- [ ] If a real defect appears, preserve the RED in a separate commit and fix
      only that defect; otherwise manufacture no RED.
- [ ] Run the complete required verification suite.
- [ ] Extend PORTAL-006 and OBSERVER-012/013 evidence.
- [ ] Commit `test(stage7): GREEN checkpoint G extraction close and loss` when
      no production correction is required.

---

# 34. TDD checkpoint H — aggregate regressions and scope

Requirements:

```text
EXTRACTION-001..010
all touched Stage 2–6 invariants
```

Files:

```text
create internal/domain/extraction_integration_test.go
update docs/traceability.md
```

Required tests:

```text
TestExtraction_OpenSyncReturnEndToEnd
TestExtraction_OpenDoesNotMutatePlane
TestExtraction_OpenDoesNotMutateObserverRoster
TestExtraction_OpenAndPortalEnergyRemainIndependent
TestExtraction_OpeningTwoPortalsCreatesDistinctInstances
TestExtraction_TwoPortalsSynchronizeAtMostOnceEach
TestExtraction_DifferentPlanesSelectIndependentObservers
TestExtraction_NoOpeningTimeObserverReservation
TestExtraction_NoWaitingAtSyncKeepsEverythingElseOperational
TestExtraction_NoWaitingAtSyncAllowsLaterManualRecall
TestExtraction_InvalidOpenAggregateIsAtomic
TestExtraction_InvalidSyncAggregateIsAtomic
TestExtraction_ErrorIdentitySurvivesEnergyWrapper
TestExtraction_NaturalFactoryStillStartsFlowNone
TestExtraction_NaturalRecallStillWorksBeforeFiveSeconds
TestExtraction_OverrideWindowSurvivesOpeningDebit
TestExtraction_OpeningRejectionPrecedenceIsDeterministic
```

These should pass after A–G. Manufacture no RED. If they expose an actual
atomicity or regression defect, split it into a genuine RED/GREEN pair and
record the precise failing assertion.

Execution:

- [ ] Add checkpoint H tests.
- [ ] Run targeted and complete suites.
- [ ] Review all EXTRACTION rows and every touched prior-stage row.
- [ ] Confirm `docs/requirements.md` still matches Final Spec.
- [ ] Commit `test(stage7): GREEN checkpoint H extraction regressions` when no
      production correction is required.

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

`gofmt -l .` must print no project Go files. Use a task-specific `GOCACHE`
under `/tmp` if the sandbox cannot write the default cache.

Stage 7 scope checks against production files:

```bash
rg -n 'time\.Sleep|time\.Now\(\)|time\.NewTicker|time\.After' \
  internal/domain/extraction.go internal/domain/observer_commands.go

rg -n 'database/sql|net/http|sync\.|ACTION_REJECTED' \
  internal/domain/extraction.go internal/domain/observer_commands.go

rg -n 'Simulation|SpawnNatural|NeedsAttention|Broadcast|Event\{' \
  internal/domain/extraction.go

rg -n 'WAITING_EXTRACTION|ExtractionReservation|ReservedObserver' \
  internal/domain
```

Expected output: no matches.

---

# 36. Targeted execution commands

Use deterministic package runs:

```bash
go test -count=1 ./internal/domain -run '^TestNewExtractionPortal_'
go test -count=1 ./internal/domain -run '^TestExtractionPlaneEligible_'
go test -count=1 ./internal/domain -run '^TestOpenExtractionPortal_'
go test -count=1 ./internal/domain -run '^TestResolveExtractionSynchronization_'
go test -count=1 ./internal/domain -run '^TestRecallObserver_Extraction|^TestRecallObserverWithLabEnergy_.*Extraction'
go test -count=1 ./internal/domain -run '^TestExtractionClose_|^TestExtractionLifecycle_'
go test -count=1 ./internal/domain -run '^TestExtraction_'
```

Also rerun completed Stage 4–6 focused packages whenever their shared command
or Lab primitives change.

---

# 37. Traceability update expected

Expected statuses after Stage 7:

```text
EXTRACTION-001  GREEN
EXTRACTION-002  GREEN
EXTRACTION-003  GREEN
EXTRACTION-004  GREEN
EXTRACTION-005  GREEN
EXTRACTION-006  GREEN
EXTRACTION-007  GREEN
EXTRACTION-008  GREEN
EXTRACTION-009  GREEN
EXTRACTION-010  GREEN

SLOT-007       GREEN
LAB-009        GREEN
LAB-010        GREEN
EMERGENCY-004  GREEN
```

Prior rows:

```text
PORTAL-001  remains honest about caller-managed global uniqueness
PORTAL-002  remains PARTIAL until Stage 8/11 sequence ownership
PORTAL-003  GREEN with both factories
PORTAL-006  GREEN with pre/post-sync close evidence
SLOT-003/005 remain GREEN and gain Extraction evidence
OBSERVER-006/007/011/012/013/016 remain GREEN and gain auto-return evidence
FLOW-003/006/007 remain GREEN and gain fixed-INBOUND/manual-return evidence
```

Do not mark UI, simulation, Events, persistence or manager requirements GREEN
from pure domain tests.

Every changed row must name concrete tests, implementation symbols and the
exact remaining later-stage boundary. `docs/requirements.md` changes only for
a proven index/spec mismatch.

---

# 38. Worklog update expected

Append Stage 7 to `01_AI_WORKLOG_CURRENT.md` with:

- [ ] execution agent/model and verified starting commit;
- [ ] Final Spec comparison result;
- [ ] approved marker and no-wait decisions;
- [ ] every genuine RED/GREEN pair with hashes;
- [ ] every already-GREEN characterization;
- [ ] exact factory random draw order;
- [ ] opening transaction and error precedence;
- [ ] synchronization semantic-time behavior;
- [ ] no-reservation/current-roster behavior;
- [ ] any actual mistake or corrective pass;
- [ ] traceability status changes;
- [ ] final verification/scope results;
- [ ] explicit statement that Stage 8 was not started.

Never claim a command passed without observing the actual output.

---

# 39. Recommended commit history

```text
test(stage7): RED checkpoint A extraction factory
feat(stage7): GREEN checkpoint A extraction factory
test(stage7): RED checkpoint B extraction plane eligibility
feat(stage7): GREEN checkpoint B extraction plane eligibility
test(stage7): RED checkpoint C atomic extraction opening
feat(stage7): GREEN checkpoint C atomic extraction opening
test(stage7): RED checkpoint D extraction synchronization
feat(stage7): GREEN checkpoint D extraction synchronization
test(stage7): RED checkpoint E automatic extraction return
feat(stage7): GREEN checkpoint E automatic extraction return
test(stage7): RED checkpoint F manual recall sync gate
feat(stage7): GREEN checkpoint F manual recall sync gate
test(stage7): GREEN checkpoint G extraction close and loss
test(stage7): GREEN checkpoint H extraction regressions
docs(stage7): record TDD evidence and verification
```

If G/H reveal missing behavior, replace the single characterization commit
with an honest RED/GREEN pair. If part of F is already correct after E, preserve
the genuine missing gate as RED and describe the already-GREEN subset.

---

# 40. Acceptance checklist — opening and factory

- [ ] The user-selected Plane must currently contain `WAITING_RETURN`.
- [ ] Waiting Observers in another Plane do not enable the selected Plane.
- [ ] Opening does not reserve or mutate an Observer.
- [ ] Extraction uses the first free regular Slot.
- [ ] Seven OPEN Portals reject opening.
- [ ] Terminal Portals release Slots normally.
- [ ] The new Portal is a distinct caller-sequenced instance.
- [ ] Kind is EXTRACTION and status is OPEN.
- [ ] Stability is STABLE with no hidden collapse timestamp.
- [ ] Observer flow is INBOUND immediately at opening.
- [ ] Creature count is exactly zero.
- [ ] Energy draw covers inclusive 60..100.
- [ ] Decay draw covers inclusive 0.1..1.0.
- [ ] TTL draw covers inclusive 30..60 seconds.
- [ ] Factory timestamps use supplied `now`.
- [ ] Synchronization marker starts nil.

---

# 41. Acceptance checklist — cost and atomicity

- [ ] Successful opening charges exactly 30 Laboratory Energy.
- [ ] Exact balance 30 succeeds.
- [ ] Balance 29 rejects.
- [ ] Derived regenerated energy may pay the cost.
- [ ] Active Leyline Override does not discount Extraction.
- [ ] Opening debit does not change Override deadline/window.
- [ ] No rejection appends a Portal.
- [ ] No rejection changes Lab or Observers.
- [ ] No rejection consumes random values.
- [ ] Duplicate Portal ID/open Slot and invalid aggregate state reject.
- [ ] Success appends exactly once without reordering old Portals.
- [ ] Lab Energy and Portal Energy remain independent.

---

# 42. Acceptance checklist — synchronization and return

- [ ] Synchronization lasts exactly configured five seconds.
- [ ] Before the deadline, resolution is a no-op.
- [ ] Exact deadline completes synchronization.
- [ ] Late resolution records the semantic deadline, not late `now`.
- [ ] Selection is repeated from current Observer state at sync time.
- [ ] Current longest-waiting in destination Plane wins.
- [ ] Exact tie uses lowest Observer ID.
- [ ] If the original candidate left, the next current candidate is selected.
- [ ] Selected Observer starts RETURNING at the semantic sync time.
- [ ] Return transit uses exactly one existing 5..15-second draw.
- [ ] Marker and Observer transition commit atomically.
- [ ] No current candidate still completes synchronization.
- [ ] No-candidate completion consumes no random value and gives no refund.
- [ ] Replay cannot start another automatic return.
- [ ] A later waiting Observer is manual-only.

---

# 43. Acceptance checklist — manual RECALL and closure

- [ ] OPEN Extraction Portal rejects manual RECALL before sync.
- [ ] Exact deadline still rejects manual RECALL until resolver runs.
- [ ] Closed pre-sync Portal reports not-open rather than synchronizing.
- [ ] Natural Portal RECALL is unchanged.
- [ ] After sync, ordinary and energy-aware manual RECALL work normally.
- [ ] Manual RECALL remains cost zero.
- [ ] Automatic active transit blocks another transit through the same Portal.
- [ ] After automatic return completes, another Observer can be manually recalled.
- [ ] Close during sync leaves all waiting Observers unchanged.
- [ ] A pre-sync closed Portal never synchronizes later.
- [ ] Close during RETURNING requires confirmation.
- [ ] Confirmed Close makes the active Observer LOST.
- [ ] Collapse before return deadline uses existing LOST behavior.
- [ ] Exact terminal/return tie preserves existing successful-return ordering.
- [ ] Other waiting Observers remain in the Plane.

---

# 44. Acceptance checklist — quality and scope

- [ ] All required checkpoint test names exist.
- [ ] Completed Stage 2–6 tests remain GREEN.
- [ ] No new Observer status or reservation state exists.
- [ ] No wall-clock, sleep, ticker or goroutine is added.
- [ ] No Events or temporary callbacks are added.
- [ ] No persistence, HTTP, WebSocket or frontend code is added.
- [ ] No simulation tick, natural spawning, LabManager or mutex is added.
- [ ] `gofmt -l .` prints nothing.
- [ ] `go vet ./...` passes.
- [ ] `go build ./...` passes.
- [ ] `go test -count=1 ./...` passes.
- [ ] `go test -race -count=1 ./...` passes.
- [ ] Traceability and worklog are updated honestly.
- [ ] Stage 8 was not started.

---

# 45. Resolved Stage 7 decisions

## S7-D1 — explicit completion marker

`Portal.ExtractionSynchronizedAt` records the one completed synchronization
attempt. This technical current-state marker was explicitly approved before
the plan was written.

## S7-D2 — no Observer reservation

Opening checks eligibility but stores no Observer ID and changes no Observer.

## S7-D3 — current roster wins

At sync, choose the current longest-waiting Observer. If the opening-time
candidate left but another waits, the current candidate returns.

## S7-D4 — no-wait completion

If none waits at sync, mark synchronization complete without auto-return,
random draw or refund. Everything else continues normally.

## S7-D5 — synchronization and transit are separate

The five-second sync ends with an atomic marker/RETURNING start. The transit
then independently lasts 5..15 seconds, for a total 10..20 seconds from open
to arrival.

## S7-D6 — only the first attempt is automatic

The persistent marker makes resolver replay a no-op. Later Observers use manual
RECALL only.

## S7-D7 — Close timing

Close before sync affects only the Portal; the Observer remains waiting. Close
during RETURNING retains existing confirmation and LOST behavior.

## S7-D8 — semantic sync timestamp

Late resolution uses `OpenedAt + cfg.ExtractionSync`, matching established
deadline-based lifecycle semantics.

## S7-D9 — Extraction never receives Override discount

The real opening action always charges the configured 30 points.

---

# 46. Self-review checklist for the execution agent

Before implementation:

- [ ] Confirm all API signatures match §6 exactly.
- [ ] Confirm no newer detailed stage plan supersedes this file.
- [ ] Confirm the approved design and this plan agree.
- [ ] Confirm tests assume no Events, manager, persistence or simulation.
- [ ] Confirm shared Observer validation refactor is behavior-preserving.

Before completion:

- [ ] Search every required test name from A–H.
- [ ] Review the diff against plan commit.
- [ ] Verify only Stage 7 production/tests/docs changed.
- [ ] Run all verification and scope commands fresh.
- [ ] Record exact RED/GREEN hashes and actual outputs.
- [ ] Confirm Stage 8 symbols and behavior are absent.

---

# 47. Stop condition

After every Stage 7 acceptance item is verified:

```text
STOP.
```

Do not begin Stage 8 in the same implementation pass.

Final Stage 7 report must include:

- files changed;
- tests added;
- RED/GREEN evidence and already-GREEN characterizations;
- opening/sync/current-roster/no-wait decisions;
- verification and scope results;
- traceability changes and remaining PARTIAL rows;
- commit hashes;
- explicit confirmation that Stage 8 was not started.

---

# 48. Next stage after completion

Stage 8 remains:

```text
Simulation via TDD

Natural spawn delay 0..20 sec
maximum seven OPEN Portals
one-second tick processing
portal/observer/extraction lifecycle orchestration
Needs Attention
state snapshots
```

Stage 8 must reuse the completed Extraction factory/opening/synchronization
primitives. It must not be started during Stage 7 implementation.
