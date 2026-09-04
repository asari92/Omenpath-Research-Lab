# Omenpath Research Lab — Stage 8 Simulation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use
> `superpowers:subagent-driven-development` when explicitly permitted, or
> `superpowers:executing-plans` to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for execution tracking.

**Goal:** implement a deterministic atomic simulation tick, Natural Portal
generation with the fixed seven-OPEN limit, and the backend Needs Attention
selector.

**Architecture:** introduce a pure domain `SimulationState` aggregate and
explicit Natural scheduler baseline. One supplied-time tick reuses completed
Portal, Lab, Observer and Extraction primitives, creates at most one Natural
Portal, then derives Needs Attention; Stage 9–11 infrastructure remains out of
scope.

**Tech Stack:** Go, `time.Time`/`time.Duration`, existing `config.Config`,
`random.Random`, domain lifecycle primitives, `testify/require`, deterministic
`testutil` fixtures, and the repository Git/TDD protocol.

---

# 1. Goal

Stage 8 answers:

```text
What canonical state does the simulation tick own?
In what order are due Portal/Observer/Extraction transitions resolved?
When is the first and each next Natural spawn scheduled?
How does delay 0 behave without producing two Portals in one tick?
How does 7/7 pause generation and how does a freed Slot restart it?
How are destination, Slot and global Portal sequence chosen?
Which one OPEN Portal is Needs Attention?
How are invalid or repeated ticks kept atomic and deterministic?
```

Required behavior:

```text
Natural schedule
  → inclusive random delay 0..20 sec
  → delay 0 is eligible only on a later tick
  → at most one Natural Portal per tick
  → after each spawn, schedule a new delay unless the spawn fills Slot 7

capacity
  → count only OPEN Portals
  → never exceed 7 OPEN
  → 7/7 discards the old timer without a random draw
  → first tick after capacity frees draws a fresh delay but does not spawn

Natural spawn
  → first free regular Slot
  → random destination among exactly 85 canonical Planes
  → repeated destination is allowed
  → one shared sequential Portal identity
  → reuse NewNaturalPortal without changing Stage 2 generation semantics

tick
  → Portal lifecycle and Collapse/Leyline effects
  → Observer transit and research
  → Extraction synchronization
  → Natural timer/spawn
  → Needs Attention from final OPEN state
  → copy-then-commit atomicity

Needs Attention
  → highest risk score
  → exact tie: UNSTABLE first
  → tie: lower effective lifetime
  → tie: older opened_at
  → complete tie: lower Portal ID
```

---

# 2. Source-of-truth order

Use the repository hierarchy fixed in `AGENTS.md`:

```text
00_FINAL_SPEC_v5.md
→ 01_AI_WORKLOG_CURRENT.md (history only)
→ 02_IMPLEMENTATION_ROADMAP_TDD.md
→ docs/superpowers/specs/2026-09-04-stage-8-simulation-design.md
→ this Stage 8 detailed plan
→ docs/requirements.md
→ docs/traceability.md
→ tests
→ implementation
```

Final Spec §§3, 5–13, 22, 31–34 and 37 control product behavior. The approved
design resolves only scheduler, deterministic tie and pure-domain ownership
details absent from the product text. Stop on a genuine conflict instead of
changing completed Stage 2–7 semantics.

---

# 3. Verified starting point

Stage 7 implementation ended at `e6ea0e2`. The approved Stage 8 design is
`0224ef9`. Before implementation, independently verify both commits, current
`git status`, relevant diffs, all stage documents and a complete green test
baseline.

Existing APIs to reuse exactly:

```go
func NewNaturalPortal(
    seq, planeID int64,
    slot int,
    now time.Time,
    cfg config.Config,
    rnd random.Random,
) Portal

func FirstFreeSlot(portals []Portal, maxSlots int) (slot int, ok bool)

func (p *Portal) ResolveLifecycle(now time.Time) (changed bool, err error)

func ResolvePortalLifecycleWithLabEmergency(
    lab *LabState,
    portal *Portal,
    now time.Time,
    cfg config.Config,
) (changed bool, err error)

func ResolveObserverLifecycle(
    observer *Observer,
    plane *Plane,
    portal *Portal,
    now time.Time,
    cfg config.Config,
) error

func ResolveExtractionSynchronization(
    portal *Portal,
    plane *Plane,
    observers []Observer,
    now time.Time,
    rnd random.Random,
    cfg config.Config,
) (observerID int64, changed bool, err error)
```

Existing completed behavior to preserve:

```text
Portal.ResolveLifecycle chooses one earliest semantic terminal outcome
Natural Close wins its exact ties; energy depletion wins instability-only tie
Collapse activates/restarts Leyline Override at Portal.ClosedAt
Observer resolver catches up phases using semantic deadlines
portal closed strictly before transit end makes the Observer LOST
Extraction sync reselects current longest waiter exactly once
NewNaturalPortal draw order and all generated balance ranges are fixed
FirstFreeSlot counts only OPEN occupants and does not reorder input
all derived realtime fields remain unpersisted
```

No simulation aggregate, Natural scheduler, atomic tick or Needs Attention
selector exists at the starting commit. `internal/engine/manager.go` remains a
Stage 1 wiring placeholder and must not receive Stage 8 behavior.

---

# 4. Requirements covered

Before checkpoint A tests, add these catalog rows to `docs/requirements.md` as
direct mappings of existing Final Spec rules:

```text
SPAWN-001       Natural delay is inclusive random 0..20 sec
SPAWN-002       successful spawn creates the next fresh delay
SPAWN-003       7/7 pauses; a free Slot starts a fresh delay
SPAWN-004       destination is random among the canonical 85 Planes
SPAWN-005       multiple OPEN Portals may target the same Plane
SPAWN-006       at most one Natural spawn per tick; delay 0 waits a later tick

SIMULATION-001  one-second tick has a supplied-time deterministic domain step
SIMULATION-002  transition order follows Final Spec §33 Stage 8 subset
SIMULATION-003  tick is atomic, monotonic and same-timestamp idempotent
SIMULATION-004  realtime Lab/Portal/creature/risk values remain derived

ATTENTION-001   highest risk_score wins
ATTENTION-002   exact risk tie prefers UNSTABLE
ATTENTION-003   next tie prefers lower effective_lifetime
ATTENTION-004   next tie prefers older opened_at
ATTENTION-005   only OPEN Portals participate
ATTENTION-006   complete tie uses lower Portal ID deterministically
ATTENTION-007   selector does not sort or mutate Portal state
```

Rows enhanced without weakening their existing boundaries:

```text
PLANE-002/003/004
PORTAL-001/002/005/007/008/009
SLOT-002/003/004/005/006
ENERGY-003/005
CREATURE-006
RISK-001..010
LAB-003/004
OBSERVER-004/008/010..013
EXTRACTION-007..009
EMERGENCY-001/002/005/006
UI-007
WS-002
```

Expected after Stage 8:

```text
SPAWN-001..006       GREEN
SIMULATION-002..004  GREEN
SIMULATION-001       PARTIAL: domain step complete; actual 1-sec ticker Stage 11
ATTENTION-001..007   GREEN at backend selector boundary
SLOT-006             GREEN for backend Natural Generator behavior
PLANE-003            GREEN through generated duplicate destinations
PORTAL-001/002       remain PARTIAL until Stage 11 serializes every opening
PLANE-004            becomes PARTIAL for cardinality validation; seed Stage 10
UI-007                remains PARTIAL until frontend rendering
WS-002                remains PLANNED until WebSocket loop
```

---

# 5. Explicit stage boundary

Stage 8 owns:

```text
pure SimulationState current-state aggregate
Natural scheduler baseline and pause state
one supplied-time atomic ResolveTick operation
reuse/orchestration of due Stage 2–7 lifecycle primitives
one Natural spawn at most per tick
hard seven-OPEN admission
random destination from the supplied canonical 85-Plane roster
shared next Portal sequence consumption for Natural spawn
pure Needs Attention backend selector
deterministic traversal/random draw ordering
```

Stage 8 does not own:

```text
Event creation, risk-change Events, history or ACTION_REJECTED
SQLite schema, 85-Plane seed storage or restart recovery
LabManager locks, command routing or REST/simulation races
actual time.Ticker, goroutine or wall-clock loop
state DTOs, WebSocket broadcast or frontend rendering
Tutorial/Live mode transitions
balance tuning
```

Do not start Stage 9.

---

# 6. Approved domain model and APIs

Add to `internal/domain/simulation.go`:

```go
type NaturalSpawnState struct {
    ScheduledAt *time.Time
    DueAt       *time.Time
    Paused      bool
}

type SimulationState struct {
    Lab          LabState
    Portals      []Portal
    Planes       []Plane
    Observers    []Observer
    NextPortalID int64
    NaturalSpawn NaturalSpawnState
    LastTickAt   *time.Time
}

type NaturalSpawnResult struct {
    Changed  bool
    Spawned  bool
    PortalID int64
}

type SimulationTickResult struct {
    Changed                bool
    Spawned                bool
    SpawnedPortalID        int64
    HasNeedsAttention      bool
    NeedsAttentionPortalID int64
    LabEnergy              int
}

func NewNaturalSpawnState(
    now time.Time,
    cfg config.Config,
    rnd random.Random,
) (NaturalSpawnState, error)

func NeedsAttentionPortalIndex(
    portals []Portal,
    now time.Time,
    cfg config.Config,
) (index int, ok bool, err error)

func ResolveNaturalSpawn(
    state *SimulationState,
    now time.Time,
    rnd random.Random,
    cfg config.Config,
) (NaturalSpawnResult, error)

func (state *SimulationState) ResolveTick(
    now time.Time,
    rnd random.Random,
    cfg config.Config,
) (SimulationTickResult, error)
```

Add to `internal/domain/errors.go`:

```go
ErrSimulationInvariant = errors.New("simulation invariant violated")
```

Do not add Event slices, persistence IDs, mutexes, callbacks or transport
snapshot types to these structures.

---

# 7. State invariants

`SimulationState` is valid only when:

```text
state is non-nil
cfg.MaxActivePortals == 7
spawn delay bounds are non-negative, whole-second and ordered
all Stage 2–7 duration/range values required by a tick are valid
Lab baseline is canonical at now
there are exactly 85 Planes with unique positive IDs
Portal IDs are unique and positive
every Portal destination references one of those Planes
every OPEN Portal slot is unique and inside 1..7
Portal kind/status/stability/flow enums and lifecycle timestamps are canonical
OPEN Portals have no terminal fields; terminal Portals have ClosedAt/reason
NextPortalID is positive and greater than every existing Portal ID
Observer IDs are unique and their status fields are canonical
Observer count equals cfg.ObserverCount (default 10)
Observer Plane/Portal references resolve when their status requires them
scheduler is either scheduled (both timestamps non-nil, Paused=false)
or paused (both timestamps nil, Paused=true)
scheduled DueAt is not before ScheduledAt
LastTickAt, when set, is not after requested now
```

Terminal historical Portals may share a slot with a later OPEN instance.
Stored slice order is not an invariant and must never be changed by a tick.

Validate the complete aggregate before any random draw. A malformed state,
configuration or reverse-time call returns `ErrSimulationInvariant` without
state mutation or random consumption.

---

# 8. Tick ordering and atomicity

The production method must use this high-level shape:

```go
func (state *SimulationState) ResolveTick(
    now time.Time,
    rnd random.Random,
    cfg config.Config,
) (SimulationTickResult, error) {
    if err := validateSimulationState(state, now, cfg); err != nil {
        return SimulationTickResult{}, err
    }
    if state.LastTickAt != nil && state.LastTickAt.Equal(now) {
        return deriveSimulationTickResult(*state, now, cfg), nil
    }

    next := cloneSimulationState(*state)
    if err := resolvePortalStage(&next, now, cfg); err != nil {
        return SimulationTickResult{}, err
    }
    if err := resolveObserverStage(&next, now, cfg); err != nil {
        return SimulationTickResult{}, err
    }
    if err := resolveExtractionStage(&next, now, rnd, cfg); err != nil {
        return SimulationTickResult{}, err
    }
    spawn, err := resolveNaturalSpawnPrepared(&next, now, rnd, cfg)
    if err != nil {
        return SimulationTickResult{}, err
    }

    tickAt := now
    next.LastTickAt = &tickAt
    result := deriveSimulationTickResult(next, now, cfg)
    result.Changed = !simulationStateEqual(*state, next)
    result.Spawned = spawn.Spawned
    result.SpawnedPortalID = spawn.PortalID
    *state = next
    return result, nil
}
```

The exact comparison helper may use explicit field comparison in production;
do not add `reflect` to production merely to compute `Changed`. Each stage can
return a changed flag and the method can OR those flags with the last-tick
baseline change.

`cloneSimulationState` copies all top-level slices before mutation. Existing
timestamp pointers are immutable values; transitions replace pointers rather
than mutate pointed timestamps. Plane aliases are not changed by Stage 8.

---

# 9. Cross-Portal lifecycle ordering

Never let slice order decide which Collapse resets Lab Energy last. Preflight
all due transitions on Portal copies, then order changed instances by:

```text
Portal.ClosedAt ascending
exact timestamp tie → Portal.ID ascending
```

Apply each ordered transition to the working Lab and Portal by reusing
`ResolvePortalLifecycleWithLabEmergency`. This preserves each Portal's Stage 2
winner and makes repeated Collapse behavior match Stage 6 chronological rules.

Natural Close does not activate Override. Each COLLAPSED transition resets Lab
at its semantic `ClosedAt`; the chronologically last Collapse supplies the
final zero baseline/deadline. Already-terminal replay is unchanged.

---

# 10. Observer and Extraction ordering

Build index maps without reordering slices:

```text
Plane ID  → Plane slice index
Portal ID → Portal slice index
```

Visit Observers by ascending Observer ID. AVAILABLE/LOST need no related
entity. EXPLORING/WAITING_RETURN resolve against their current Plane.
OUTBOUND/RETURNING resolve against both referenced Portal and its destination
Plane. Any unresolved reference is a preflight invariant error.

After all Observer phases catch up, visit OPEN Extraction Portals by ascending
Portal ID and call `ResolveExtractionSynchronization`. This lets research that
ends exactly at `now` produce a WAITING_RETURN candidate for same-time sync.
Terminal Extraction Portals remain resolver no-ops.

---

# 11. Natural scheduler contract

Delay creation:

```go
delaySeconds := rnd.IntInclusive(
    int(cfg.SpawnDelayMin/time.Second),
    int(cfg.SpawnDelayMax/time.Second),
)
scheduledAt := now
dueAt := now.Add(time.Duration(delaySeconds) * time.Second)
```

A deadline is due only when both conditions hold:

```go
now.After(*spawn.ScheduledAt) && !now.Before(*spawn.DueAt)
```

This makes zero a real configured draw while forbidding a second same-tick
spawn.

At each Natural stage:

```text
OPEN count == 7
  → Paused=true, timestamps=nil, no draw

Paused and OPEN count < 7
  → create fresh schedule from now, no spawn this tick

scheduled but not due
  → no-op

scheduled and due
  → re-check capacity
  → first free Slot
  → destination index draw among 85 Planes
  → NewNaturalPortal(NextPortalID, planeID, slot, now, cfg, rnd)
  → append, increment NextPortalID
  → if OPEN count is now 7: pause without delay draw
  → otherwise schedule next delay from now
```

Natural destination selection uses the stored Plane slice as the canonical
uniform index set. Do not sort the slice before the random index draw. Duplicate
destinations across Portal instances remain legal.

---

# 12. Deterministic random order

Within one tick:

```text
1. Extraction return duration draws, Extraction Portal ID ascending
2. Natural destination index, only when a spawn is due
3. NewNaturalPortal existing draws:
   TTL, energy, decay, stability, optional hidden offset, creatures
4. next Natural delay, only if fewer than seven OPEN remain
```

Initial scheduler construction consumes exactly one delay draw. Full-capacity,
not-due, same-timestamp replay, malformed aggregate and terminal lifecycle
paths consume none.

---

# 13. Needs Attention contract

`NeedsAttentionPortalIndex` considers only OPEN Portals. Compare candidates in
this exact order:

```text
higher Portal.RiskScore(now, cfg)
UNSTABLE over STABLE on exact numeric score tie
lower Portal.EffectiveLifetime(now)
older Portal.OpenedAt
lower Portal.ID as final technical tie
```

Return `(0, false, nil)` for no OPEN candidate. Preserve the original Portal
slice byte-for-byte. Hidden instability timestamp is not an input except via
the visible STABLE/UNSTABLE enum; it must not enter the numeric risk formula.

---

# 14. File map

Production:

```text
create internal/domain/simulation.go
create internal/domain/simulation_spawn.go
create internal/domain/simulation_attention.go
modify internal/domain/errors.go
```

Tests:

```text
create internal/domain/simulation_state_test.go
create internal/domain/simulation_attention_test.go
create internal/domain/simulation_scheduler_test.go
create internal/domain/simulation_spawn_test.go
create internal/domain/simulation_portal_lifecycle_test.go
create internal/domain/simulation_observer_test.go
create internal/domain/simulation_extraction_test.go
create internal/domain/simulation_integration_test.go
```

Documentation during implementation:

```text
modify docs/requirements.md
modify docs/traceability.md
modify 01_AI_WORKLOG_CURRENT.md
```

Do not modify `internal/engine/manager.go`, `internal/domain/event.go`, any
persistence/transport package, or Final Spec.

---

# 15. Fixed test fixtures

Use `testutil.BaseTime` and local test helpers:

```go
func simulationPlanes() []domain.Plane {
    planes := make([]domain.Plane, 85)
    for i := range planes {
        planes[i] = domain.Plane{
            ID:   int64(i + 1),
            Name: fmt.Sprintf("Plane %02d", i+1),
        }
    }
    return planes
}

func scheduledSpawn(scheduledAt time.Time, delay time.Duration) domain.NaturalSpawnState {
    dueAt := scheduledAt.Add(delay)
    return domain.NaturalSpawnState{
        ScheduledAt: &scheduledAt,
        DueAt:       &dueAt,
    }
}

func simulationState(now time.Time) domain.SimulationState {
    return domain.SimulationState{
        Lab: domain.LabState{
            EnergyBase:   100,
            EnergyBaseAt: now,
        },
        Planes:       simulationPlanes(),
        Observers:    domain.NewObserverRoster(10, now),
        NextPortalID: 1,
        NaturalSpawn: scheduledSpawn(now, 10*time.Second),
    }
}
```

Additional local helpers may create canonical waiting/transit Observers and
OPEN/terminal Portals, but must assign explicit IDs, slots, destination IDs and
timestamps. Do not execute production commands merely to arrange a test
fixture.

Representative shared helper implementations:

```go
func openPortals(n int) []domain.Portal {
    portals := make([]domain.Portal, n)
    for i := range portals {
        portal := testutil.NewPortalBuilder().Slot(i + 1).Build()
        portal.ID = int64(i + 1)
        portal.Name = fmt.Sprintf("Omenpath #%04d", i+1)
        portal.DestinationPlaneID = int64(i + 1)
        portals[i] = portal
    }
    return portals
}

func sevenOpenPortals() []domain.Portal {
    return openPortals(7)
}

func sixOpenAndOneTerminalPortals() []domain.Portal {
    portals := openPortals(6)
    closedAt := testutil.BaseTime
    terminal := testutil.NewPortalBuilder().Slot(7).Build()
    terminal.ID = 7
    terminal.Name = "Omenpath #0007"
    terminal.DestinationPlaneID = 7
    terminal.Status = domain.PortalStatusClosed
    terminal.TerminationReason = domain.TerminationManualClose
    terminal.ClosedAt = &closedAt
    return append(portals, terminal)
}

func cloneForTest(state domain.SimulationState) domain.SimulationState {
    state.Portals = append([]domain.Portal(nil), state.Portals...)
    state.Planes = append([]domain.Plane(nil), state.Planes...)
    state.Observers = append([]domain.Observer(nil), state.Observers...)
    return state
}

func countOpenForTest(portals []domain.Portal) int {
    count := 0
    for _, portal := range portals {
        if portal.Status == domain.PortalStatusOpen {
            count++
        }
    }
    return count
}

func collapsingPortal(id int64, slot int, collapseAt time.Time) domain.Portal {
    portal := testutil.NewPortalBuilder().Slot(slot).Decay(1).Build()
    portal.ID = id
    portal.Name = fmt.Sprintf("Omenpath #%04d", id)
    portal.DestinationPlaneID = id
    portal.EnergyBase = collapseAt.Sub(testutil.BaseTime).Seconds()
    portal.EnergyBaseAt = testutil.BaseTime
    portal.ScheduledCloseAt = testutil.BaseTime.Add(time.Minute)
    return portal
}

func outboundPortalClosingAt(closesAt time.Time) domain.Portal {
    portal := testutil.NewPortalBuilder().TTL(closesAt.Sub(testutil.BaseTime)).Build()
    portal.ID = 1
    portal.DestinationPlaneID = 1
    portal.ObserverFlow = domain.PortalFlowOutbound
    return portal
}

func outboundObserverEndingAt(endsAt time.Time) domain.Observer {
    portalID := int64(1)
    startedAt := testutil.BaseTime
    return domain.Observer{
        ID:             1,
        Status:         domain.ObserverOutbound,
        ActivePortalID: &portalID,
        PhaseStartedAt: &startedAt,
        PhaseEndsAt:    &endsAt,
        CreatedAt:      testutil.BaseTime,
        UpdatedAt:      testutil.BaseTime,
    }
}

func reassignObserverIDs(observers []domain.Observer) {
    for i := range observers {
        observers[i].ID = int64(i + 1)
    }
}

func observerRosterWithResearchEnding(
    observerID, planeID int64,
    endsAt time.Time,
) []domain.Observer {
    observers := domain.NewObserverRoster(10, testutil.BaseTime)
    startedAt := endsAt.Add(-config.Default().ResearchDuration)
    observers[observerID-1] = domain.Observer{
        ID:             observerID,
        Status:         domain.ObserverExploring,
        CurrentPlaneID: &planeID,
        PhaseStartedAt: &startedAt,
        PhaseEndsAt:    &endsAt,
        CreatedAt:      testutil.BaseTime,
        UpdatedAt:      startedAt,
    }
    return observers
}

func dueNaturalSimulation(now time.Time, cfg config.Config) domain.SimulationState {
    state := simulationState(now)
    state.NaturalSpawn = scheduledSpawn(now, 0)
    return state
}

func naturalSpawnRandom(
    cfg config.Config,
    destinationIndex, nextDelay int,
) *testutil.FakeRandom {
    return testutil.NewFakeRandom().
        QueueInt(
            destinationIndex,
            int(cfg.NaturalTTLMin/time.Second),
            0,
            nextDelay,
        ).
        QueueFloat(cfg.PortalEnergyMin, cfg.PortalDecayMin, 0.9)
}
```

Tests that assert cross-method draw order should define a local
`simulationRecordingRandom` implementing `random.Random`; it records each
`IntInclusive`/`FloatRange` call before returning its preloaded value. Keep it
inside `simulation_spawn_test.go` rather than expanding production interfaces.

---

# 16. Baseline protocol

Before checkpoint A:

- [ ] Read this complete plan and `AGENTS.md`.
- [ ] Re-read Final Spec §§3, 5–13, 22, 31–34 and 37.
- [ ] Re-read the Stage 8 approved design and Stage 2–7 plans.
- [ ] Verify commits `e6ea0e2` and `0224ef9`.
- [ ] Inspect current Go structure, tests, `git log`, relevant recent diffs and
      `git status`.
- [ ] Run the complete required verification suite and record a green baseline.
- [ ] Confirm no Stage 8 production symbols already exist.
- [ ] Confirm `internal/engine/manager.go` is still untouched placeholder scope.

---

# 17. TDD execution protocol

For every checkpoint:

```text
1. Map exact requirement IDs and Final Spec sections.
2. Add only focused tests for that checkpoint.
3. Run the targeted command and observe genuine RED.
4. Commit RED evidence before production code.
5. Implement the smallest behavior required by the checkpoint.
6. Run targeted tests and observe GREEN.
7. Run gofmt/vet/build/full tests/race.
8. Refactor only while GREEN.
9. Update traceability honestly.
10. Commit GREEN implementation/evidence.
```

Do not fake RED for characterization tests already satisfied by Stage 2–7.

---

# 18. Checkpoint dependency graph

```text
A state / scheduler model / invariants
        ↓
B Needs Attention selector
        ↓
C pause / resume / delay scheduler
        ↓
D due Natural spawn / destination / sequence
        ↓
E Portal lifecycle / Collapse ordering
        ↓
F Observer lifecycle / exploration
        ↓
G Extraction synchronization ordering
        ↓
H full tick atomicity / idempotence / regressions
```

Checkpoints A–G expose independent behavior and must retain separate commits.

---

# 19. Checkpoint A — simulation state and scheduler invariants

Requirements:

```text
SPAWN-001 scheduler baseline
SIMULATION-001 pure supplied-time boundary
SIMULATION-003 structural atomicity foundation
```

Files:

```text
create internal/domain/simulation.go
create internal/domain/simulation_state_test.go
modify internal/domain/errors.go
modify docs/requirements.md
modify docs/traceability.md
```

Required tests:

```text
TestNewNaturalSpawnState_AcceptsZeroDelay
TestNewNaturalSpawnState_AcceptsMaximumDelay
TestNewNaturalSpawnState_DrawsExactlyOnce
TestNewNaturalSpawnState_StoresSchedulingOrigin
TestNewNaturalSpawnState_RejectsNegativeMinimum
TestNewNaturalSpawnState_RejectsReversedBounds
TestNewNaturalSpawnState_RejectsSubsecondBounds
TestNewNaturalSpawnState_RejectsNilRandom
TestSimulationState_RejectsNilState
TestSimulationState_RequiresExactlyEightyFivePlanes
TestSimulationState_RejectsDuplicatePlaneIDs
TestSimulationState_RejectsDuplicatePortalIDs
TestSimulationState_RejectsDuplicateOpenSlots
TestSimulationState_AllowsTerminalSlotReuse
TestSimulationState_RejectsUnknownPortalStatus
TestSimulationState_RejectsOpenPortalWithClosedAt
TestSimulationState_RejectsTerminalPortalWithoutClosedAt
TestSimulationState_RejectsInvalidNextPortalID
TestSimulationState_RequiresConfiguredObserverCount
TestSimulationState_RejectsMalformedSpawnSchedule
TestSimulationState_InvalidAggregateDoesNotConsumeRandom
TestSimulationState_InvalidAggregateIsAtomic
```

Representative RED test:

```go
func TestNewNaturalSpawnState_AcceptsZeroDelay(t *testing.T) {
    cfg := config.Default()
    rnd := testutil.NewFakeRandom().QueueInt(0)

    spawn, err := domain.NewNaturalSpawnState(testutil.BaseTime, cfg, rnd)

    require.NoError(t, err)
    require.False(t, spawn.Paused)
    require.Equal(t, testutil.BaseTime, *spawn.ScheduledAt)
    require.Equal(t, testutil.BaseTime, *spawn.DueAt)
}
```

Minimal implementation shape:

```go
func NewNaturalSpawnState(
    now time.Time,
    cfg config.Config,
    rnd random.Random,
) (NaturalSpawnState, error) {
    if rnd == nil || cfg.SpawnDelayMin < 0 ||
        cfg.SpawnDelayMin > cfg.SpawnDelayMax ||
        cfg.SpawnDelayMin%time.Second != 0 ||
        cfg.SpawnDelayMax%time.Second != 0 {
        return NaturalSpawnState{}, ErrSimulationInvariant
    }
    delay := time.Duration(rnd.IntInclusive(
        int(cfg.SpawnDelayMin/time.Second),
        int(cfg.SpawnDelayMax/time.Second),
    )) * time.Second
    scheduledAt := now
    dueAt := now.Add(delay)
    return NaturalSpawnState{
        ScheduledAt: &scheduledAt,
        DueAt:       &dueAt,
    }, nil
}
```

Execution:

- [ ] Add Stage 8 requirement/trace rows as PLANNED.
- [ ] Add only checkpoint A tests and fixture helpers.
- [ ] Run `go test -count=1 ./internal/domain -run '^TestNewNaturalSpawnState_|^TestSimulationState_'`.
- [ ] Observe compile RED for missing simulation types/functions/error.
- [ ] Commit `test(stage8): RED checkpoint A simulation state`.
- [ ] Add the approved structures, error, scheduler constructor, clone and
      structural validation helpers.
- [ ] Run the targeted command and observe GREEN.
- [ ] Run the complete required verification suite.
- [ ] Mark only the proven scheduler/model subset GREEN/PARTIAL.
- [ ] Commit `feat(stage8): GREEN checkpoint A simulation state`.

---

# 20. Checkpoint B — Needs Attention selector

Requirements:

```text
ATTENTION-001..007
RISK-001..010 reuse
UI-007 backend subset
```

Files:

```text
create internal/domain/simulation_attention.go
create internal/domain/simulation_attention_test.go
modify docs/traceability.md
```

Required tests:

```text
TestNeedsAttentionPortalIndex_ReturnsNoneWithoutOpenPortals
TestNeedsAttentionPortalIndex_IgnoresClosedPortal
TestNeedsAttentionPortalIndex_IgnoresCollapsedPortal
TestNeedsAttentionPortalIndex_SelectsHighestRiskScore
TestNeedsAttentionPortalIndex_RiskScoreBeatsInstabilityTieBreaker
TestNeedsAttentionPortalIndex_ExactRiskTiePrefersUnstable
TestNeedsAttentionPortalIndex_ExactRiskAndStabilityTiePrefersLowerLifetime
TestNeedsAttentionPortalIndex_LifetimeTiePrefersOlderOpening
TestNeedsAttentionPortalIndex_CompleteTiePrefersLowerPortalID
TestNeedsAttentionPortalIndex_ReturnsOriginalSliceIndex
TestNeedsAttentionPortalIndex_DoesNotReorderInput
TestNeedsAttentionPortalIndex_DoesNotMutatePortals
TestNeedsAttentionPortalIndex_UsesCurrentTimeDerivedRisk
TestNeedsAttentionPortalIndex_HiddenTimestampDoesNotAffectScore
TestNeedsAttentionPortalIndex_RejectsInvalidRiskConfig
TestNeedsAttentionPortalIndex_RejectionIsAtomic
```

Representative test:

```go
func TestNeedsAttentionPortalIndex_ExactRiskTiePrefersUnstable(t *testing.T) {
    cfg := config.Default()
    stable := testutil.NewPortalBuilder().Stable().TTL(36 * time.Second).Build()
    unstable := testutil.NewPortalBuilder().
        Unstable(testutil.BaseTime.Add(40 * time.Second)).
        Build()
    stable.ID = 2
    unstable.ID = 1
    portals := []domain.Portal{stable, unstable}

    index, ok, err := domain.NeedsAttentionPortalIndex(
        portals, testutil.BaseTime, cfg,
    )

    require.NoError(t, err)
    require.True(t, ok)
    require.Equal(t, 1, index)
    require.Equal(t, unstable.RiskScore(testutil.BaseTime, cfg),
        stable.RiskScore(testutil.BaseTime, cfg))
}
```

Implementation comparison order:

```go
func attentionBefore(a, b Portal, now time.Time, cfg config.Config) bool {
    aScore, bScore := a.RiskScore(now, cfg), b.RiskScore(now, cfg)
    if aScore != bScore {
        return aScore > bScore
    }
    if a.Stability != b.Stability {
        return a.Stability == PortalUnstable
    }
    aLife, bLife := a.EffectiveLifetime(now), b.EffectiveLifetime(now)
    if aLife != bLife {
        return aLife < bLife
    }
    if !a.OpenedAt.Equal(b.OpenedAt) {
        return a.OpenedAt.Before(b.OpenedAt)
    }
    return a.ID < b.ID
}
```

Execution:

- [ ] Add only checkpoint B tests.
- [ ] Run `go test -count=1 ./internal/domain -run '^TestNeedsAttentionPortalIndex_'`.
- [ ] Observe compile RED for missing selector.
- [ ] Commit `test(stage8): RED checkpoint B needs attention`.
- [ ] Implement the pure selector without sorting or mutation.
- [ ] Run the targeted command and observe GREEN.
- [ ] Run the complete required verification suite.
- [ ] Mark ATTENTION-001..007 GREEN; keep UI-007 PARTIAL.
- [ ] Commit `feat(stage8): GREEN checkpoint B needs attention`.

---

# 21. Checkpoint C — Natural scheduler pause and resume

Requirements:

```text
SPAWN-001
SPAWN-003
SPAWN-006 zero-delay later-tick half
SLOT-005/006
```

Files:

```text
create internal/domain/simulation_spawn.go
create internal/domain/simulation_scheduler_test.go
modify docs/traceability.md
```

Required tests:

```text
TestResolveNaturalSpawn_BeforeDeadlineIsNoOp
TestResolveNaturalSpawn_AtSchedulingOriginIsNoOpForZeroDelay
TestResolveNaturalSpawn_FullCapacityPausesImmediately
TestResolveNaturalSpawn_FullCapacityClearsExistingSchedule
TestResolveNaturalSpawn_FullCapacityConsumesNoRandom
TestResolveNaturalSpawn_AlreadyPausedAndFullIsIdempotent
TestResolveNaturalSpawn_CountsOnlyOpenPortals
TestResolveNaturalSpawn_TerminalSlotAllowsFreshSchedule
TestResolveNaturalSpawn_FirstTickAfterFreeOnlySchedules
TestResolveNaturalSpawn_FirstTickAfterFreeDoesNotSpawnAtZeroDelay
TestResolveNaturalSpawn_FirstTickAfterFreeDrawsExactlyOnce
TestResolveNaturalSpawn_FreshScheduleUsesCurrentTickAsOrigin
TestResolveNaturalSpawn_FreshScheduleUsesInclusiveMaximum
TestResolveNaturalSpawn_PauseDoesNotChangePortalRecords
TestResolveNaturalSpawn_ResumeDoesNotChangePortalRecords
TestResolveNaturalSpawn_InvalidSchedulerIsAtomic
TestResolveNaturalSpawn_InvalidSchedulerConsumesNoRandom
TestResolveNaturalSpawn_RejectsNilState
```

Representative tests:

```go
func TestResolveNaturalSpawn_FirstTickAfterFreeDoesNotSpawnAtZeroDelay(t *testing.T) {
    cfg := config.Default()
    state := simulationState(testutil.BaseTime)
    state.Portals = sixOpenAndOneTerminalPortals()
    state.NextPortalID = 8
    state.NaturalSpawn = domain.NaturalSpawnState{Paused: true}
    beforePortals := append([]domain.Portal(nil), state.Portals...)

    result, err := domain.ResolveNaturalSpawn(
        &state,
        testutil.BaseTime.Add(time.Second),
        testutil.NewFakeRandom().QueueInt(0),
        cfg,
    )

    require.NoError(t, err)
    require.True(t, result.Changed)
    require.False(t, result.Spawned)
    require.Equal(t, beforePortals, state.Portals)
    require.False(t, state.NaturalSpawn.Paused)
    require.Equal(t, testutil.BaseTime.Add(time.Second),
        *state.NaturalSpawn.ScheduledAt)
    require.Equal(t, *state.NaturalSpawn.ScheduledAt,
        *state.NaturalSpawn.DueAt)
}

func TestResolveNaturalSpawn_FullCapacityConsumesNoRandom(t *testing.T) {
    cfg := config.Default()
    state := simulationState(testutil.BaseTime)
    state.Portals = sevenOpenPortals()
    state.NextPortalID = 8

    result, err := domain.ResolveNaturalSpawn(
        &state, testutil.BaseTime.Add(time.Second), nil, cfg,
    )

    require.NoError(t, err)
    require.True(t, result.Changed)
    require.False(t, result.Spawned)
    require.True(t, state.NaturalSpawn.Paused)
}
```

Minimal branch order:

```go
openCount := countOpenPortals(state.Portals)
switch {
case openCount >= cfg.MaxActivePortals:
    changed := !state.NaturalSpawn.Paused ||
        state.NaturalSpawn.ScheduledAt != nil ||
        state.NaturalSpawn.DueAt != nil
    state.NaturalSpawn = NaturalSpawnState{Paused: true}
    return NaturalSpawnResult{Changed: changed}, nil

case state.NaturalSpawn.Paused:
    next, err := NewNaturalSpawnState(now, cfg, rnd)
    if err != nil {
        return NaturalSpawnResult{}, err
    }
    state.NaturalSpawn = next
    return NaturalSpawnResult{Changed: true}, nil

case !naturalSpawnDue(state.NaturalSpawn, now):
    return NaturalSpawnResult{}, nil
}
```

Checkpoint C stops before the due/free spawn branch, which checkpoint D adds.
The temporary checkpoint C implementation leaves a due/free schedule unchanged;
checkpoint D tests then provide the genuine RED for missing spawn behavior.

Execution:

- [ ] Add only checkpoint C tests.
- [ ] Run `go test -count=1 ./internal/domain -run '^TestResolveNaturalSpawn_.*(Deadline|SchedulingOrigin|Capacity|Pause|Free|Schedule|Invalid|Nil)'`.
- [ ] Observe compile RED for missing Natural resolver.
- [ ] Commit `test(stage8): RED checkpoint C natural scheduler`.
- [ ] Implement count, due predicate, pause, idempotent full and fresh-resume
      scheduling branches atomically.
- [ ] Run the targeted command and observe GREEN.
- [ ] Run the complete required verification suite.
- [ ] Mark SPAWN-001/003 PARTIAL and extend SLOT-005/006 evidence.
- [ ] Commit `feat(stage8): GREEN checkpoint C natural scheduler`.

---

# 22. Checkpoint D — due Natural Portal spawn

Requirements:

```text
SPAWN-002/004/005/006
PLANE-003
PORTAL-001/002/003
SLOT-002/003/004/006
ENERGY-001
STABILITY-001..003
CREATURE-001..006
```

Files:

```text
modify internal/domain/simulation_spawn.go
create internal/domain/simulation_spawn_test.go
modify docs/traceability.md
```

Required tests:

```text
TestResolveNaturalSpawn_AtDeadlineSpawnsOnePortal
TestResolveNaturalSpawn_LateTickSpawnsOnlyOnePortal
TestResolveNaturalSpawn_UsesTickTimeAsOpenedAt
TestResolveNaturalSpawn_UsesFirstFreeSlot
TestResolveNaturalSpawn_ReusesTerminalPortalSlot
TestResolveNaturalSpawn_SelectsFirstPlaneIndex
TestResolveNaturalSpawn_SelectsLastPlaneIndex
TestResolveNaturalSpawn_AllowsRepeatedDestination
TestResolveNaturalSpawn_UsesNextPortalID
TestResolveNaturalSpawn_FormatsSequentialPortalName
TestResolveNaturalSpawn_IncrementsSequenceExactlyOnce
TestResolveNaturalSpawn_PreservesExistingPortalOrder
TestResolveNaturalSpawn_AppendsWithoutRemovingTerminalHistory
TestResolveNaturalSpawn_ReusesNaturalFactoryRanges
TestResolveNaturalSpawn_ReusesNaturalFactoryStability
TestResolveNaturalSpawn_ReusesNaturalFactoryCreatures
TestResolveNaturalSpawn_SchedulesNextDelayAfterSuccess
TestResolveNaturalSpawn_NextScheduleUsesSpawnTick
TestResolveNaturalSpawn_NextZeroDelayDoesNotSpawnTwice
TestResolveNaturalSpawn_SeventhPortalPausesWithoutNextDelayDraw
TestResolveNaturalSpawn_SixthPortalDrawsNextDelay
TestResolveNaturalSpawn_DestinationDrawPrecedesFactoryDraws
TestResolveNaturalSpawn_FactoryDrawsPrecedeNextDelay
TestResolveNaturalSpawn_NoDueSpawnConsumesNoRandom
TestResolveNaturalSpawn_SuccessDoesNotMutatePlanes
TestResolveNaturalSpawn_SuccessDoesNotMutateObserversOrLab
```

Representative test:

```go
func TestResolveNaturalSpawn_AtDeadlineSpawnsOnePortal(t *testing.T) {
    cfg := config.Default()
    state := simulationState(testutil.BaseTime)
    state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 10*time.Second)
    rnd := testutil.NewFakeRandom().
        QueueInt(
            6, // destination index
            int(cfg.NaturalTTLMin/time.Second),
            0, // creatures
            12, // next delay
        ).
        QueueFloat(
            cfg.PortalEnergyMin,
            cfg.PortalDecayMin,
            0.9, // STABLE roll
        )

    result, err := domain.ResolveNaturalSpawn(
        &state, testutil.BaseTime.Add(10*time.Second), rnd, cfg,
    )

    require.NoError(t, err)
    require.True(t, result.Changed)
    require.True(t, result.Spawned)
    require.Equal(t, int64(1), result.PortalID)
    require.Len(t, state.Portals, 1)
    require.Equal(t, int64(7), state.Portals[0].DestinationPlaneID)
    require.Equal(t, 1, state.Portals[0].SlotIndex)
    require.Equal(t, int64(2), state.NextPortalID)
}
```

Due branch implementation shape:

```go
slot, ok := FirstFreeSlot(state.Portals, cfg.MaxActivePortals)
if !ok {
    return NaturalSpawnResult{}, ErrSimulationInvariant
}
planeIndex := rnd.IntInclusive(0, len(state.Planes)-1)
id := state.NextPortalID
portal := NewNaturalPortal(
    id, state.Planes[planeIndex].ID, slot, now, cfg, rnd,
)
state.Portals = append(state.Portals, portal)
state.NextPortalID++

if openCount+1 == cfg.MaxActivePortals {
    state.NaturalSpawn = NaturalSpawnState{Paused: true}
} else {
    next, err := NewNaturalSpawnState(now, cfg, rnd)
    if err != nil {
        return NaturalSpawnResult{}, err
    }
    state.NaturalSpawn = next
}
return NaturalSpawnResult{Changed: true, Spawned: true, PortalID: id}, nil
```

Apply the branch to a cloned `SimulationState` and commit only after all
factory/scheduler work succeeds. This preserves atomic state even if the
aggregate is malformed.

Execution:

- [ ] Add only checkpoint D tests.
- [ ] Run `go test -count=1 ./internal/domain -run '^TestResolveNaturalSpawn_'`.
- [ ] Observe RED because due schedules do not create a Portal.
- [ ] Commit `test(stage8): RED checkpoint D natural spawn`.
- [ ] Add the due branch, explicit draw order and copy-then-commit.
- [ ] Run the targeted command and observe GREEN.
- [ ] Run the complete required verification suite.
- [ ] Mark SPAWN-001..006 GREEN and update Plane/Portal/Slot evidence honestly.
- [ ] Commit `feat(stage8): GREEN checkpoint D natural spawn`.

---

# 23. Checkpoint E — Portal lifecycle and Leyline orchestration

Requirements:

```text
SIMULATION-002 Portal phases
SIMULATION-003 atomic tick
SIMULATION-004 derived state
PORTAL-005/007/008/009
EMERGENCY-001/002/005/006
```

Files:

```text
modify internal/domain/simulation.go
create internal/domain/simulation_portal_lifecycle_test.go
modify docs/traceability.md
```

Required tests:

```text
TestSimulationTick_ResolvesNaturalClose
TestSimulationTick_ResolvesEnergyCollapse
TestSimulationTick_ResolvesInstabilityCollapse
TestSimulationTick_UsesSemanticPortalClosedAt
TestSimulationTick_NaturalCloseDoesNotActivateOverride
TestSimulationTick_EnergyCollapseResetsLabEnergy
TestSimulationTick_InstabilityCollapseResetsLabEnergy
TestSimulationTick_MultipleCollapsesApplyChronologically
TestSimulationTick_MultipleCollapsesIgnorePortalSliceOrder
TestSimulationTick_EqualCollapseTimesUsePortalIDOrder
TestSimulationTick_AlreadyTerminalPortalIsIdempotent
TestSimulationTick_TerminalPortalReleasesSlotBeforeNaturalStage
TestSimulationTick_DerivedPortalEnergyIsNotPersisted
TestSimulationTick_DerivedCreaturesAreNotPersisted
TestSimulationTick_DerivedLabEnergyIsNotRebased
TestSimulationTick_ResultReportsCurrentLabEnergy
TestSimulationTick_PortalStageFailureIsAtomic
TestSimulationTick_PortalStageFailureConsumesNoRandom
```

Representative test:

```go
func TestSimulationTick_MultipleCollapsesIgnorePortalSliceOrder(t *testing.T) {
    cfg := config.Default()
    state := simulationState(testutil.BaseTime)
    early := collapsingPortal(1, 1, testutil.BaseTime.Add(2*time.Second))
    late := collapsingPortal(2, 2, testutil.BaseTime.Add(4*time.Second))
    state.Portals = []domain.Portal{late, early}
    state.NextPortalID = 3
    state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 20*time.Second)

    result, err := state.ResolveTick(
        testutil.BaseTime.Add(5*time.Second), nil, cfg,
    )

    require.NoError(t, err)
    require.True(t, result.Changed)
    require.Equal(t, 1, result.LabEnergy)
    require.Equal(t, testutil.BaseTime.Add(4*time.Second),
        state.Lab.EnergyBaseAt)
    require.Equal(t, 0, state.Lab.EnergyBase)
    require.Equal(t, testutil.BaseTime.Add(24*time.Second),
        *state.Lab.LeylineOverrideUntil)
    require.Equal(t, int64(2), state.Portals[0].ID)
    require.Equal(t, int64(1), state.Portals[1].ID)
}
```

Note: with default `+1/sec`, `result.LabEnergy` at tick `T+5` after the last
Collapse at `T+4` is exactly 1. The stored baseline remains zero at `T+4`.

Execution:

- [ ] Add only checkpoint E tests with Natural schedules safely not due.
- [ ] Run `go test -count=1 ./internal/domain -run '^TestSimulationTick_.*(Close|Collapse|Portal|LabEnergy|Creatures)'`.
- [ ] Observe compile RED for missing `ResolveTick` and lifecycle stage.
- [ ] Commit `test(stage8): RED checkpoint E portal tick`.
- [ ] Implement atomic clone, due-transition preflight/sort, lifecycle apply,
      result derivation and LastTickAt commit.
- [ ] Do not add Events for transitions.
- [ ] Run the targeted command and observe GREEN.
- [ ] Run the complete required verification suite.
- [ ] Mark the proven Portal/Emergency/SIMULATION subsets.
- [ ] Commit `feat(stage8): GREEN checkpoint E portal tick`.

---

# 24. Checkpoint F — Observer lifecycle inside the tick

Requirements:

```text
SIMULATION-002 Observer transit/research phase
OBSERVER-004/008/010/011/012/013
PLANE-005..009
```

Files:

```text
modify internal/domain/simulation.go
create internal/domain/simulation_observer_test.go
modify docs/traceability.md
```

Required tests:

```text
TestSimulationTick_OutboundBeforeDeadlineRemainsOutbound
TestSimulationTick_OutboundAtDeadlineStartsResearch
TestSimulationTick_LateOutboundCatchesUpToWaitingReturn
TestSimulationTick_ResearchAtDeadlineBecomesWaitingReturn
TestSimulationTick_ReturningAtDeadlineBecomesAvailable
TestSimulationTick_SuccessfulReturnExploresPlane
TestSimulationTick_RepeatReturnPreservesExploredAt
TestSimulationTick_PortalClosedBeforeTransitMakesObserverLost
TestSimulationTick_PortalCollapsedBeforeTransitMakesObserverLost
TestSimulationTick_CloseAtTransitDeadlineAllowsArrival
TestSimulationTick_PortalLifecycleRunsBeforeObserverLifecycle
TestSimulationTick_ObserversAreVisitedByID
TestSimulationTick_ObserverTraversalDoesNotReorderSlice
TestSimulationTick_MultipleObserversResolveIndependently
TestSimulationTick_WaitingObserverRemainsWaiting
TestSimulationTick_AvailableAndLostObserversRemainUnchanged
TestSimulationTick_RejectsUnknownObserverPlaneReference
TestSimulationTick_RejectsUnknownActivePortalReference
TestSimulationTick_DueObserverStateIsValidPreflight
TestSimulationTick_ObserverStageFailureIsAtomic
```

Representative ordering test:

```go
func TestSimulationTick_PortalLifecycleRunsBeforeObserverLifecycle(t *testing.T) {
    cfg := config.Default()
    state := simulationState(testutil.BaseTime)
    portal := outboundPortalClosingAt(testutil.BaseTime.Add(4 * time.Second))
    observer := outboundObserverEndingAt(testutil.BaseTime.Add(8 * time.Second))
    state.Portals = []domain.Portal{portal}
    state.Observers = append(
        []domain.Observer{observer},
        domain.NewObserverRoster(9, testutil.BaseTime)...,
    )
    reassignObserverIDs(state.Observers)
    state.NextPortalID = 2
    state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 20*time.Second)

    _, err := state.ResolveTick(
        testutil.BaseTime.Add(5*time.Second), nil, cfg,
    )

    require.NoError(t, err)
    require.Equal(t, domain.PortalStatusClosed, state.Portals[0].Status)
    require.Equal(t, domain.ObserverLost, state.Observers[0].Status)
}
```

Do not use the existing command-time `validateObserverRoster(observers, now)`
as tick preflight: it intentionally rejects transit that is already due at
`now`, while the tick exists to resolve that state. Add a Stage 8 structural
validator that requires canonical field combinations and ordered phase
timestamps but permits `PhaseEndsAt <= now`. Keep Stage 4 command validation
unchanged.

Execution:

- [ ] Add only checkpoint F tests.
- [ ] Run `go test -count=1 ./internal/domain -run '^TestSimulationTick_.*(Outbound|Research|Returning|Return|Observer|Transit|Explores|Waiting|Available|Lost)'`.
- [ ] Observe RED because tick does not advance Observer phases.
- [ ] Commit `test(stage8): RED checkpoint F observer tick`.
- [ ] Add reference maps, ID-ordered index traversal and Observer resolver
      calls after the Portal stage.
- [ ] Preserve slice order and Stage 3 semantic timestamps.
- [ ] Run the targeted command and observe GREEN.
- [ ] Run the complete required verification suite.
- [ ] Extend Observer/Plane/SIMULATION trace evidence.
- [ ] Commit `feat(stage8): GREEN checkpoint F observer tick`.

---

# 25. Checkpoint G — Extraction synchronization inside the tick

Requirements:

```text
SIMULATION-002 Extraction phase
EXTRACTION-007/008/009
OBSERVER-010/016
FLOW-003/006/007
```

Files:

```text
modify internal/domain/simulation.go
create internal/domain/simulation_extraction_test.go
modify docs/traceability.md
```

Required tests:

```text
TestSimulationTick_ExtractionBeforeSyncRemainsPending
TestSimulationTick_ExtractionAtSyncCompletes
TestSimulationTick_ResearchCompletionFeedsSameTickExtraction
TestSimulationTick_ExtractionSelectsCurrentLongestWaiting
TestSimulationTick_ExtractionOriginalCandidateGoneSelectsNext
TestSimulationTick_ExtractionWithoutWaitingCompletesWithoutReturn
TestSimulationTick_CompletedExtractionDoesNotReturnSecondObserver
TestSimulationTick_TerminalExtractionDoesNotSynchronize
TestSimulationTick_PortalCloseAtSyncWinsBeforeExtraction
TestSimulationTick_MultipleExtractionsVisitPortalIDOrder
TestSimulationTick_MultipleExtractionsReturnDifferentObservers
TestSimulationTick_ExtractionDrawsTransitDurationBeforeNaturalSpawn
TestSimulationTick_ExtractionNoWaitConsumesNoTransitRandom
TestSimulationTick_ExtractionMarkerUsesSemanticDeadline
TestSimulationTick_ExtractionTransitUsesSemanticDeadline
TestSimulationTick_ExtractionDoesNotReorderPortals
TestSimulationTick_ExtractionDoesNotReorderObservers
TestSimulationTick_InvalidExtractionAggregateIsAtomic
```

Representative same-tick test:

```go
func TestSimulationTick_ResearchCompletionFeedsSameTickExtraction(t *testing.T) {
    cfg := config.Default()
    state := simulationState(testutil.BaseTime)
    extraction := extractionPortal(testutil.BaseTime, cfg)
    extraction.ID = 2
    extraction.SlotIndex = 2
    state.Portals = []domain.Portal{extraction}
    state.NextPortalID = 3
    state.Observers = observerRosterWithResearchEnding(
        1, extraction.DestinationPlaneID,
        testutil.BaseTime.Add(cfg.ExtractionSync),
    )
    state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 20*time.Second)

    result, err := state.ResolveTick(
        testutil.BaseTime.Add(cfg.ExtractionSync),
        testutil.NewFakeRandom().QueueInt(5),
        cfg,
    )

    require.NoError(t, err)
    require.True(t, result.Changed)
    require.Equal(t, domain.ObserverReturning, state.Observers[0].Status)
    require.Equal(t, testutil.BaseTime.Add(cfg.ExtractionSync),
        *state.Observers[0].PhaseStartedAt)
    require.NotNil(t, state.Portals[0].ExtractionSynchronizedAt)
}
```

Extraction traversal shape:

```go
indices := extractionPortalIndicesByID(state.Portals)
for _, index := range indices {
    portal := &state.Portals[index]
    planeIndex, ok := planeIndexByID[portal.DestinationPlaneID]
    if !ok {
        return ErrSimulationInvariant
    }
    if _, _, err := ResolveExtractionSynchronization(
        portal,
        &state.Planes[planeIndex],
        state.Observers,
        now,
        rnd,
        cfg,
    ); err != nil {
        return err
    }
}
```

Run this only after the Observer stage and before Natural generation. Include
terminal Extraction records in deterministic visitation only if needed for
validation; their resolver path must stay no-op and draw-free.

Execution:

- [ ] Add only checkpoint G tests.
- [ ] Run `go test -count=1 ./internal/domain -run '^TestSimulationTick_Extraction|^TestSimulationTick_ResearchCompletionFeeds'`.
- [ ] Observe RED because the tick leaves Extraction synchronization pending.
- [ ] Commit `test(stage8): RED checkpoint G extraction tick`.
- [ ] Add deterministic Extraction stage after Observer catch-up.
- [ ] Preserve Stage 7 one-shot marker and manual-only subsequent returns.
- [ ] Run the targeted command and observe GREEN.
- [ ] Run the complete required verification suite.
- [ ] Extend Extraction/Observer/Flow/SIMULATION evidence.
- [ ] Commit `feat(stage8): GREEN checkpoint G extraction tick`.

---

# 26. Checkpoint H — full tick integration and regressions

Requirements:

```text
SIMULATION-001..004
SPAWN-001..006 integration
ATTENTION-001..007 integration
all touched Stage 2–7 rows remain GREEN at their promised boundaries
```

Files:

```text
create internal/domain/simulation_integration_test.go
modify docs/traceability.md
```

Required tests:

```text
TestSimulationTick_FullOrderPortalObserverExtractionSpawnAttention
TestSimulationTick_NeedsAttentionUsesPostSpawnState
TestSimulationTick_TerminalReleaseAllowsOnlyFreshSchedule
TestSimulationTick_ZeroDelayProducesAtMostOneSpawn
TestSimulationTick_SevenOpenPortalsNeverBecomeEight
TestSimulationTick_SameTimestampReplayIsNoOp
TestSimulationTick_SameTimestampReplayConsumesNoRandom
TestSimulationTick_BackwardTimeRejectsAtomically
TestSimulationTick_LargeJumpDoesNotBackfillNaturalSpawns
TestSimulationTick_LargeJumpStillCatchesUpDomainLifecycles
TestSimulationTick_RandomOrderExtractionBeforeNatural
TestSimulationTick_InvalidStateConsumesNoRandom
TestSimulationTick_InvalidStateDoesNotPartiallyResolvePortal
TestSimulationTick_InvalidStateDoesNotPartiallyResolveObserver
TestSimulationTick_InvalidStateDoesNotPartiallySynchronizeExtraction
TestSimulationTick_InvalidStateDoesNotPartiallySpawn
TestSimulationTick_DoesNotPersistDerivedRealtimeValues
TestSimulationTick_DoesNotReorderAnyAggregateSlice
TestSimulationTick_DoesNotCreateEvents
TestSimulationTick_NaturalPortalFactoryBehaviorUnchanged
TestSimulationTick_ManualCommandsRemainOutsideTick
TestSimulationTick_ResultIsDerivedFromCommittedState
```

Representative cap regression:

```go
func TestSimulationTick_SevenOpenPortalsNeverBecomeEight(t *testing.T) {
    cfg := config.Default()
    state := simulationState(testutil.BaseTime)
    state.Portals = sevenOpenPortals()
    state.NextPortalID = 8
    state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 0)
    before := cloneForTest(state)

    result, err := state.ResolveTick(
        testutil.BaseTime.Add(time.Second), nil, cfg,
    )

    require.NoError(t, err)
    require.False(t, result.Spawned)
    require.Len(t, state.Portals, len(before.Portals))
    require.Equal(t, 7, countOpenForTest(state.Portals))
    require.True(t, state.NaturalSpawn.Paused)
}
```

Representative idempotence test:

```go
func TestSimulationTick_SameTimestampReplayConsumesNoRandom(t *testing.T) {
    cfg := config.Default()
    state := dueNaturalSimulation(testutil.BaseTime, cfg)
    firstRandom := naturalSpawnRandom(cfg, 0, 5)
    now := testutil.BaseTime.Add(time.Second)

    _, err := state.ResolveTick(now, firstRandom, cfg)
    require.NoError(t, err)
    afterFirst := cloneForTest(state)

    result, err := state.ResolveTick(now, nil, cfg)
    require.NoError(t, err)
    require.False(t, result.Changed)
    require.Equal(t, afterFirst, state)
}
```

Characterization rule:

```text
If a required H test passes before any H production change, record it as
GREEN characterization. Do not weaken assertions or manufacture a failure.
If a real integration defect appears, commit the failing test as a genuine
RED, make the smallest correction, rerun all shared-stage tests, and record
the extra RED/GREEN pair in the worklog.
```

Execution:

- [ ] Add all checkpoint H tests.
- [ ] Run `go test -count=1 ./internal/domain -run '^TestSimulationTick_'`.
- [ ] Record which tests are characterization GREEN and any genuine failures.
- [ ] Split each genuine integration defect into honest RED/GREEN commits.
- [ ] Run completed Stage 2–7 focused suites after every shared-domain fix.
- [ ] Run the complete required verification suite.
- [ ] Review every new Stage 8 row and every touched earlier-stage row.
- [ ] Commit `test(stage8): GREEN checkpoint H simulation regressions` when no
      production correction remains.

---

# 27. Required verification commands

At every meaningful GREEN checkpoint and final completion:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
go test -race -count=1 ./...
```

`gofmt -l .` must print no project Go files. Use a task-specific `GOCACHE`
under `/tmp` when the sandbox cannot write the default cache.

Never replace the full suite with targeted runs. Targeted tests prove the
checkpoint; the full suite proves completed-stage compatibility.

---

# 28. Targeted execution commands

```bash
go test -count=1 ./internal/domain -run '^TestNewNaturalSpawnState_|^TestSimulationState_'
go test -count=1 ./internal/domain -run '^TestNeedsAttentionPortalIndex_'
go test -count=1 ./internal/domain -run '^TestResolveNaturalSpawn_'
go test -count=1 ./internal/domain -run '^TestSimulationTick_.*(Portal|Close|Collapse|LabEnergy|Creatures)'
go test -count=1 ./internal/domain -run '^TestSimulationTick_.*(Observer|Outbound|Research|Returning|Return|Transit|Explores|Waiting|Lost)'
go test -count=1 ./internal/domain -run '^TestSimulationTick_Extraction|^TestSimulationTick_ResearchCompletionFeeds'
go test -count=1 ./internal/domain -run '^TestSimulationTick_'
```

Also rerun focused Stage 2–7 tests whenever a shared primitive must change.
A shared primitive change requires a demonstrated Stage 8 integration defect,
not convenience refactoring.

---

# 29. Scope verification commands

Run against production Stage 8 files:

```bash
rg -n 'time\.Sleep|time\.Now\(\)|time\.NewTicker|time\.After' \
  internal/domain/simulation*.go

rg -n 'database/sql|net/http|sync\.|websocket|Broadcast|ACTION_REJECTED' \
  internal/domain/simulation*.go

rg -n 'Event\{|\[\]Event|EventType|EventRiskLevelChanged' \
  internal/domain/simulation*.go

rg -n 'gorm|sqlite|migration|repository|transaction' \
  internal/domain/simulation*.go

git diff --name-only e6ea0e2..HEAD
git diff --check
```

The first four searches must return no matches. The diff may contain only the
Stage 8 production/tests/docs listed in §14 plus the approved design and this
plan. Confirm `internal/engine/manager.go` and `internal/domain/event.go` have
no diff.

---

# 30. Traceability update expected

Final Stage 8 rows:

```text
SPAWN-001       GREEN
SPAWN-002       GREEN
SPAWN-003       GREEN
SPAWN-004       GREEN
SPAWN-005       GREEN
SPAWN-006       GREEN

SIMULATION-001  PARTIAL — deterministic domain step complete;
                         actual one-second ticker owned by Stage 11
SIMULATION-002  GREEN
SIMULATION-003  GREEN
SIMULATION-004  GREEN

ATTENTION-001   GREEN
ATTENTION-002   GREEN
ATTENTION-003   GREEN
ATTENTION-004   GREEN
ATTENTION-005   GREEN
ATTENTION-006   GREEN
ATTENTION-007   GREEN
```

Existing rows:

```text
SLOT-006       GREEN for pause/resume backend generator
PLANE-003      GREEN for repeated generated destinations
PLANE-004      PARTIAL for exactly-85 validation; seed/bootstrap Stage 10
PORTAL-001     PARTIAL until all opening commands share Stage 11 serialization
PORTAL-002     PARTIAL until Extraction and Natural allocate under one manager
SLOT-004       remains PARTIAL until LabManager owns lifecycle storage
UI-007         PARTIAL: backend selection GREEN, frontend Stage 15/16 absent
WS-002         PLANNED: no ticker/broadcast in Stage 8
EVENT-001..005 PLANNED
PERSIST-*      PLANNED
```

Every changed row must name concrete tests and production symbols. Do not mark
actual cadence, UI, Event, persistence or concurrency requirements GREEN from
pure domain tests.

---

# 31. Requirements catalog update

Add three sections to `docs/requirements.md` during checkpoint A using the IDs
and exact meanings from §4 of this plan:

```text
## SPAWN
SPAWN-001..006 → Final Spec §§5, 7, 33, 37

## SIMULATION
SIMULATION-001..004 → Final Spec §§31–34

## ATTENTION
ATTENTION-001..007 → Final Spec §6
```

This is catalog completion, not a product-spec change. Keep the existing
`SpawnDelayMin=0` and `SpawnDelayMax=20s`; do not edit Final Spec or balance
defaults.

---

# 32. Worklog update

After checkpoint H and final verification, append Stage 8 to
`01_AI_WORKLOG_CURRENT.md` with:

```text
verified starting commit and clean baseline
Final Spec/design conflict result
implemented domain state and API list
each checkpoint RED hash and GREEN hash
characterization checkpoints explicitly identified
actual test count and required-name audit
random draw order and zero-delay decision
7/7 pause/resume semantics
tick ordering and atomicity decisions
traceability changes and remaining PARTIAL/PLANNED boundaries
real operational mistakes or corrective passes
fresh gofmt/vet/build/test/race outputs
scope-search results
explicit statement that Stage 9 was not started
```

Commit documentation separately after implementation evidence is final:

```bash
git add 01_AI_WORKLOG_CURRENT.md docs/requirements.md docs/traceability.md
git commit -m "docs(stage8): record TDD evidence and verification"
```

If requirements/traceability rows were already committed checkpoint-by-
checkpoint, stage only the files still changed. Never use broad `git add .`.

---

# 33. Commit protocol

Use exact paths and separate staging/commit commands. Expected history shape:

```text
test(stage8): RED checkpoint A simulation state
feat(stage8): GREEN checkpoint A simulation state
test(stage8): RED checkpoint B needs attention
feat(stage8): GREEN checkpoint B needs attention
test(stage8): RED checkpoint C natural scheduler
feat(stage8): GREEN checkpoint C natural scheduler
test(stage8): RED checkpoint D natural spawn
feat(stage8): GREEN checkpoint D natural spawn
test(stage8): RED checkpoint E portal tick
feat(stage8): GREEN checkpoint E portal tick
test(stage8): RED checkpoint F observer tick
feat(stage8): GREEN checkpoint F observer tick
test(stage8): RED checkpoint G extraction tick
feat(stage8): GREEN checkpoint G extraction tick
test(stage8): GREEN checkpoint H simulation regressions
docs(stage8): record TDD evidence and verification
```

If checkpoint H finds a defect, insert its genuine RED and corrective GREEN
commits before the final H/docs commits. Do not squash RED evidence.

---

# 34. Acceptance checklist — model and scheduler

- [ ] `NaturalSpawnState` has only scheduler baseline/pause state.
- [ ] Initial delay is inclusive 0..20 seconds through config.
- [ ] Subsecond/negative/reversed delay config is rejected before a draw.
- [ ] Delay zero is eligible only on a later tick.
- [ ] One tick creates at most one Natural Portal.
- [ ] Same-timestamp replay creates none and consumes no randomness.
- [ ] Seven OPEN Portals always pause generation.
- [ ] Full-state pause clears the obsolete deadline without drawing.
- [ ] First tick after a Slot frees schedules only; it never spawns.
- [ ] Terminal Portal records do not occupy Slots.
- [ ] A seventh successful spawn enters paused state without a throwaway draw.

---

# 35. Acceptance checklist — Natural spawn

- [ ] Every spawn re-checks OPEN count immediately before append.
- [ ] OPEN count never exceeds `cfg.MaxActivePortals == 7`.
- [ ] First free regular Slot is used.
- [ ] Destination index is drawn across exactly 85 validated Planes.
- [ ] Multiple OPEN Portals to the same Plane are allowed.
- [ ] Stored Plane order is not changed.
- [ ] Existing Portal order/history is not changed or removed.
- [ ] `NextPortalID` is consumed and incremented exactly once.
- [ ] `NewNaturalPortal` is reused without changing its draw order/ranges.
- [ ] Successful non-seventh spawn schedules one fresh next delay.
- [ ] Late tick produces one current-time Portal, not catch-up bursts.

---

# 36. Acceptance checklist — tick orchestration

- [ ] Tick receives `now` and `random.Random`; it has no wall-clock call.
- [ ] Portal transitions precede Observer resolution.
- [ ] Multiple Portal transitions apply in semantic-time/ID order.
- [ ] Collapse/Leyline behavior reuses Stage 6 semantics.
- [ ] Observer transit/research precedes Extraction synchronization.
- [ ] Same-time research completion may feed Extraction sync.
- [ ] Extraction Portals are visited by ID and keep one-shot semantics.
- [ ] Natural spawn follows lifecycle and Extraction work.
- [ ] Needs Attention is derived from final post-spawn OPEN state.
- [ ] Existing slices retain storage order.
- [ ] Reverse time rejects atomically.
- [ ] Same timestamp is an unchanged, draw-free replay.
- [ ] Structural/config error commits no partial state.
- [ ] Due Observer phases are accepted by structural preflight.
- [ ] Derived Lab/Portal/creature/risk values are not stored each tick.

---

# 37. Acceptance checklist — Needs Attention

- [ ] No OPEN Portal returns no selection.
- [ ] CLOSED and COLLAPSED Portals are excluded.
- [ ] Highest current numeric risk wins.
- [ ] Exact score tie prefers UNSTABLE.
- [ ] Remaining tie prefers lower effective lifetime.
- [ ] Remaining tie prefers older opening.
- [ ] Complete tie prefers lower Portal ID.
- [ ] Hidden instability timestamp does not alter numeric risk.
- [ ] Selector returns original slice index and does not sort/mutate.
- [ ] Tick result exposes only identity, not numeric risk.
- [ ] UI-007 remains PARTIAL until frontend work.

---

# 38. Acceptance checklist — quality and scope

- [ ] All required checkpoint test names exist.
- [ ] Completed Stage 2–7 tests remain GREEN.
- [ ] Requirement catalog contains SPAWN/SIMULATION/ATTENTION IDs.
- [ ] Traceability statuses match §30 exactly or explain proven deviations.
- [ ] No Event creation or ACTION_REJECTED exists in Stage 8 files.
- [ ] No persistence or restart recovery exists.
- [ ] No `LabManager`, mutex or concurrent command routing change exists.
- [ ] No actual ticker, goroutine, sleep or wall-clock read exists.
- [ ] No state DTO, WebSocket broadcast or frontend code exists.
- [ ] `gofmt -l .` prints nothing.
- [ ] `go vet ./...` passes.
- [ ] `go build ./...` passes.
- [ ] `go test -count=1 ./...` passes.
- [ ] `go test -race -count=1 ./...` passes.
- [ ] Worklog contains exact RED/GREEN evidence.
- [ ] Stage 9 was not started.

---

# 39. Plan self-review checklist for the execution agent

Before implementation:

- [ ] Confirm no newer detailed stage plan supersedes this file.
- [ ] Confirm all APIs in §6 use the same field/type names throughout.
- [ ] Confirm the approved design and this plan agree on delay zero and 7/7.
- [ ] Confirm Stage 4 command-time Observer validation remains untouched.
- [ ] Confirm tests contain no Event, persistence, manager or transport needs.

Before completion:

- [ ] Search every required test name from checkpoints A–H.
- [ ] Review the full diff against commits `e6ea0e2` and `0224ef9`.
- [ ] Verify only files listed in §14 plus required docs changed.
- [ ] Run every verification and scope command fresh.
- [ ] Record exact RED/GREEN hashes and actual command outputs.
- [ ] Confirm no Stage 9 symbols or behavior were added.

---

# 40. Resolved Stage 8 decisions

```text
S8-D1   Spawn delay remains inclusive 0..20 seconds.
S8-D2   Maximum one Natural spawn per tick; zero waits a later tick.
S8-D3   Every spawn re-checks the hard seven-OPEN limit.
S8-D4   Full capacity discards timer; freed capacity creates a fresh delay.
S8-D5   Pure domain aggregate now; ticker/mutex ownership stays Stage 11.
S8-D6   Existing semantic Portal lifecycle ordering remains authoritative.
S8-D7   Observer lifecycle precedes Extraction synchronization.
S8-D8   Complete Needs Attention tie uses lower Portal ID.
S8-D9   Tick is deterministic copy-then-commit under supplied time/randomness.
S8-D10  New requirement IDs map existing Final Spec rules only.
```

The approved rationale and rejected alternatives live in
`docs/superpowers/specs/2026-09-04-stage-8-simulation-design.md`.

---

# 41. Deferred Stage 9+ work

Stage 9 owns:

```text
meaningful domain Events
RISK_LEVEL_CHANGED only on level transition
ACTION_REJECTED
Portal History and global Event Log source
```

Stage 10 owns SQLite, canonical Plane seed persistence and recovery. Stage 11
owns the real one-second ticker, LabManager lock, command/tick serialization
and global Portal sequence allocation across both Natural and Extraction
openings. Stage 12+ owns snapshots, WebSocket transport and UI.

Stage 8 must expose a clean aggregate transition boundary for these consumers
without implementing any of them.

---

# 42. Stop condition

After every Stage 8 acceptance item is verified:

```text
STOP.
```

Do not begin Stage 9 in the same implementation pass.

Final Stage 8 report must include:

```text
implemented domain behavior and API list
all changed files
test count and required-name audit
RED/GREEN commit hashes by checkpoint
gofmt/vet/build/test/race outputs
requirements and traceability changes
scope-search results
clean git status
explicit confirmation that Stage 9 was not started
```
