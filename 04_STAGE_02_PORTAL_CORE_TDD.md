# Omenpath Research Lab — Stage 2
## Portal Core via TDD

> Depends on:
> - `00_FINAL_SPEC_v5.md`
> - `03_STAGE_00_01_TDD_FOUNDATION.md`
>
> This stage implements only Portal-domain behavior. No HTTP, WebSocket, SQLite repositories, real simulation goroutine, Observer lifecycle, or Laboratory Energy orchestration yet.

---

# 1. Goal

Implement the complete **Portal Core** as deterministic Go domain logic using TDD:

```text
Portal State
→ Natural Close
→ Portal Energy
→ Energy Collapse
→ Stability
→ Hidden Instability Collapse
→ Manual Close primitive
→ Stabilize primitive
→ Creatures
→ Risk
→ Portal Slots
```

At the end of Stage 2, Portal behavior must be testable entirely in memory with:

- `FakeClock`;
- deterministic test fixtures;
- no `time.Sleep`;
- no real random in tests;
- no database;
- no HTTP.

---

# 2. Requirements covered

Primary requirement families:

```text
PORTAL-001..009
SLOT-001..007
ENERGY-001..010
STABILITY-001..007
CREATURE-001..008
RISK-001..013
```

Related requirements partially prepared but not fully orchestrated here:

```text
LAB-*       # action costs are integrated later
OBSERVER-*  # active Observer close-confirmation added later
EVENT-*     # events integrated later
```

---

# 3. Important stage boundary

`STABILIZE` has two layers:

## Portal primitive — Stage 2

Portal-level validation and mutation:

```text
Portal OPEN
Portal UNSTABLE
Portal Energy <= 85
→ STABLE
→ hidden collapse cleared
→ Energy +15
```

## Laboratory command — later stage

Adds:

```text
Lab Energy >= 20
Lab Energy -= 20
Emergency cost override
Event persistence/broadcast
```

Stage 2 MUST NOT couple `Portal` directly to `LabState`.

The same rule applies to `CLOSE`:

- Stage 2 implements Portal state transition and creature confirmation semantics.
- Lab Energy cost `5` is added at Laboratory orchestration stage.
- Observer-in-transit close confirmation is added after Observer lifecycle exists.

This keeps Portal domain logic independent from resource orchestration.

---

# 4. Proposed domain types

Initial enums:

```go
type PortalKind string

const (
    PortalKindNatural    PortalKind = "NATURAL"
    PortalKindExtraction PortalKind = "EXTRACTION"
)

type PortalStatus string

const (
    PortalStatusOpen      PortalStatus = "OPEN"
    PortalStatusClosed    PortalStatus = "CLOSED"
    PortalStatusCollapsed PortalStatus = "COLLAPSED"
)

type PortalStability string

const (
    PortalStable   PortalStability = "STABLE"
    PortalUnstable PortalStability = "UNSTABLE"
)

type PortalFlow string

const (
    PortalFlowNone     PortalFlow = "NONE"
    PortalFlowOutbound PortalFlow = "OUTBOUND"
    PortalFlowInbound  PortalFlow = "INBOUND"
)

type TerminationReason string

const (
    TerminationNone            TerminationReason = ""
    TerminationNaturalClose    TerminationReason = "NATURAL_CLOSE"
    TerminationManualClose     TerminationReason = "MANUAL_CLOSE"
    TerminationEnergyDepleted  TerminationReason = "ENERGY_DEPLETED"
    TerminationInstability     TerminationReason = "INSTABILITY"
)

type RiskLevel string

const (
    RiskLow      RiskLevel = "LOW"
    RiskMedium   RiskLevel = "MEDIUM"
    RiskHigh     RiskLevel = "HIGH"
    RiskCritical RiskLevel = "CRITICAL"
)
```

Exact names may be adjusted for Go style, but must remain semantically identical to Final Spec.

---

# 5. Proposed Portal structure

Stage 2 Portal should contain only fields required by current domain logic:

```go
type Portal struct {
    ID                 int64
    Name               string
    SlotIndex          int
    Kind               PortalKind
    DestinationPlaneID int64

    EnergyBase      float64
    EnergyBaseAt    time.Time
    EnergyDecayRate float64

    Stability              PortalStability
    InstabilityCollapseAt  *time.Time

    OpenedAt         time.Time
    ScheduledCloseAt time.Time

    CreaturesInitial int

    ObserverFlow PortalFlow

    Status            PortalStatus
    TerminationReason TerminationReason
    ClosedAt          *time.Time

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

No derived fields such as:

```text
CurrentEnergy
CreaturesInside
RiskScore
RiskLevel
RemainingSeconds
```

are stored in the struct.

They are computed.

---

# 6. Proposed Portal domain API

Prefer small explicit methods/functions.

Suggested shape:

```go
func (p Portal) IsTerminal() bool

func (p Portal) ScheduledRemaining(now time.Time) time.Duration

func (p Portal) CurrentEnergy(now time.Time) float64

func (p Portal) EnergyLifetime(now time.Time) time.Duration

func (p Portal) EffectiveLifetime(now time.Time) time.Duration

func (p Portal) CreaturesInside(now time.Time, cfg Config) int

func (p Portal) RiskScore(now time.Time, cfg Config) float64

func (p Portal) RiskLevel(now time.Time, cfg Config) RiskLevel
```

Mutations:

```go
func (p *Portal) ResolveLifecycle(now time.Time) (changed bool, err error)

func (p *Portal) Close(now time.Time, confirmCreatureInterrupt bool, cfg Config) error

func (p *Portal) Stabilize(now time.Time, cfg Config) error
```

Factories/helpers may be separated:

```go
func NewNaturalPortal(...)

func FirstFreeSlot(openPortals []Portal, maxSlots int) (int, bool)
```

Do not lock exact signatures until first RED tests reveal the simplest API.

Tests define the desired behaviour; implementation should remain minimal.

---

# 7. Lifecycle resolution semantics

A critical goal is to avoid tick-order-dependent behavior.

Instead of relying on:

```text
check Natural Close
then check Energy
then check Instability
```

the domain should reason from timestamps/current state.

## Rule A — Energy before Natural Close

If Energy reaches zero strictly **before** `scheduled_close_at`:

```text
COLLAPSED / ENERGY_DEPLETED
```

## Rule B — Energy exactly at Natural Close

Final Spec says Energy collapse occurs if Energy reaches zero **before** normal close.

Therefore if both occur at exactly the same instant:

```text
NATURAL_CLOSE wins
→ CLOSED
```

This removes ambiguous tick ordering.

## Rule C — Instability

If `instability_collapse_at` is strictly earlier than `scheduled_close_at` and is reached while Portal remains OPEN+UNSTABLE:

```text
COLLAPSED / INSTABILITY
```

The generator should normally create hidden collapse strictly before Natural Close.

If later implementation permits equal timestamps, define Natural Close as tie winner for deterministic behavior unless Final Spec is explicitly changed.

## Rule D — earliest termination wins

For OPEN Portal, effective termination event is the earliest applicable timestamp among:

```text
scheduled_close_at
energy_depletion_at
instability_collapse_at (only UNSTABLE)
```

with Natural Close winning exact ties.

This should be implemented/tested as deterministic domain semantics rather than accidental Tick ordering.

---

# 8. Substage 2.1 — Basic Portal state

## RED tests

### PORTAL-004 — allowed statuses represented

No invalid magic-string state should be needed in domain code.

### PORTAL-009 — terminal state immutable

Test:

```text
Portal CLOSED
attempt lifecycle resolution later
→ remains CLOSED

Portal COLLAPSED
attempt lifecycle resolution later
→ remains COLLAPSED
```

Potential tests:

```go
TestPortal_ClosedStateIsTerminal
TestPortal_CollapsedStateIsTerminal
```

## GREEN

Implement only enough enum/state helpers.

---

# 9. Substage 2.2 — Natural Close

Requirements:

```text
PORTAL-005
```

## RED tests

### Before TTL

```text
now < scheduled_close_at
→ OPEN
```

```go
TestPortal_RemainsOpenBeforeScheduledClose
```

### At TTL

```text
now == scheduled_close_at
→ CLOSED
→ NATURAL_CLOSE
→ ClosedAt = scheduled_close_at or resolved now according to chosen mutation semantics
```

```go
TestPortal_NaturalCloseAtScheduledTime
```

### After TTL

If resolving late:

```text
now > scheduled_close_at
→ CLOSED / NATURAL_CLOSE
```

The semantic termination timestamp should preferably be `scheduled_close_at`, not delayed Tick time.

```go
TestPortal_LateResolutionPreservesNaturalCloseTime
```

## Design rule

Persist the **actual semantic event time**, not merely when a delayed Tick noticed it.

---

# 10. Substage 2.3 — Portal Energy

Requirements:

```text
ENERGY-003
ENERGY-004
ENERGY-005
```

## RED table tests

Example:

```text
base 80
decay 0.5/sec
elapsed 10 sec
→ 75
```

```text
base 10
decay 1/sec
elapsed 11 sec
→ 0
not -1
```

Suggested:

```go
TestPortal_CurrentEnergy
```

table cases:

| base | decay | elapsed | expected |
|---:|---:|---:|---:|
| 80 | 0.5 | 0s | 80 |
| 80 | 0.5 | 10s | 75 |
| 10 | 1.0 | 10s | 0 |
| 10 | 1.0 | 15s | 0 |

Use tolerance for float comparisons.

## Energy depletion time

Add pure calculation:

```text
energy_depletion_at =
energy_base_at
+ energy_base / decay_rate
```

Test exact timestamp.

If `decay_rate <= 0`, production config should prevent it; domain function may return effectively infinite lifetime or validation error. Since Final Spec requires `0.1..1.0`, no zero decay should be created by valid factory.

---

# 11. Substage 2.4 — Energy Collapse

Requirements:

```text
PORTAL-007
ENERGY-005
```

## RED tests

### Energy depletion before TTL

```text
Energy reaches 0 at +10 sec
Natural close at +30 sec
resolve at +10 sec
→ COLLAPSED
→ ENERGY_DEPLETED
```

```go
TestPortal_CollapsesWhenEnergyDepletesBeforeNaturalClose
```

### Exact tie

```text
Energy reaches 0 at +10 sec
Natural close at +10 sec
→ CLOSED / NATURAL_CLOSE
```

```go
TestPortal_NaturalCloseWinsEnergyTie
```

### Natural close earlier

```text
Natural close +10
Energy depletion +100
→ CLOSED / NATURAL_CLOSE
```

---

# 12. Substage 2.5 — Stability and hidden collapse

Requirements:

```text
STABILITY-001..004
PORTAL-008
```

## RED tests

### STABLE

```text
STABLE
InstabilityCollapseAt == nil
```

### UNSTABLE

```text
UNSTABLE
InstabilityCollapseAt != nil
```

### Before hidden collapse

```text
now < hidden collapse
→ still OPEN
```

### At hidden collapse

```text
now >= hidden collapse
hidden < natural close
→ COLLAPSED / INSTABILITY
```

### Natural close earlier

```text
Natural close < hidden collapse
→ CLOSED / NATURAL_CLOSE
```

Potential test names:

```go
TestPortal_UnstableCollapsesAtHiddenTime
TestPortal_NaturalCloseWinsBeforeHiddenCollapse
```

---

# 13. Hidden instability timestamp generation

Decision fixed.

For every UNSTABLE Portal:

```text
instability_collapse_at =
random between:
opened_at + 5 sec
and
scheduled_close_at - 1 sec
```

Meaning:

- Portal cannot collapse from instability during the first 5 seconds after opening;
- hidden collapse, if it happens, is scheduled strictly before Natural Close;
- at least 1 second remains between hidden instability collapse and scheduled close.

Balance config:

```text
INSTABILITY_MIN_LIFETIME = 5 sec
INSTABILITY_CLOSE_MARGIN = 1 sec
```

Rationale:

- avoids instant 0–1 sec collapses;
- preserves unpredictability;
- guarantees hidden collapse before Natural Close;
- keeps short-lived unstable Portals dangerous but still interactable.

This is no longer an open decision.

---

# 14. Substage 2.6 — Stabilize primitive

Requirements:

```text
STABILITY-005..007
ENERGY-006..010
```

Again: Lab Energy cost is NOT part of this primitive yet.

## RED tests

### Happy path

```text
OPEN
UNSTABLE
current Energy 50
hidden collapse exists

Stabilize
→ STABLE
→ hidden collapse nil
→ Energy baseline 65
→ EnergyBaseAt = now
→ decay unchanged
```

```go
TestPortal_StabilizeConvertsUnstableToStable
```

### Boundary 85

```text
current Energy = 85
→ allowed
→ new Energy = 100
```

```go
TestPortal_StabilizeAllowsExactly85Percent
```

### Over 85

```text
85.1
→ ErrPortalOverchargeRisk
→ state unchanged
```

```go
TestPortal_StabilizeRejectsEnergyAbove85
```

### Stable

```text
STABLE
→ ErrPortalAlreadyStable
```

### Closed / collapsed

```text
→ ErrPortalNotOpen
```

### Current vs baseline

Important regression test:

```text
EnergyBase=80
Decay=.5
10 sec elapsed
Current=75

Stabilize now
→ 90
NOT 95
```

This catches the bug where implementation adds +15 to stale baseline instead of current derived Energy.

---

# 15. Substage 2.7 — Creatures

Requirements:

```text
CREATURE-002..008
```

## Max count tests

### Boundary TTL=10

```text
floor((10-2)/2) = 4
```

```go
TestCreatureLimit_TTL10AllowsMaxFour
```

### TTL=20

```text
floor((20-2)/2)=9
```

### TTL=30

```text
14 → capped at 10
```

### Defensive short TTL

Though valid Natural TTL >=10, helper should not produce negative max:

```text
TTL=1 → 0
```

## Current creatures table

For initial=4:

| elapsed | current |
|---:|---:|
| 0s | 4 |
| 1s | 4 |
| 2s | 3 |
| 3s | 3 |
| 4s | 2 |
| 6s | 1 |
| 8s | 0 |
| 20s | 0 |

```go
TestPortal_CreaturesInsideDecreasesEveryTwoSeconds
```

---

# 16. Substage 2.8 — Manual Close primitive

Portal-level close only.

Lab Energy cost later.

Observer transit confirmation later.

Requirements currently available:

```text
PORTAL-006
CREATURE-008
```

## RED tests

### Normal Close

```text
OPEN
creatures=0
Close(confirm=false)
→ CLOSED / MANUAL_CLOSE
```

### Creatures without confirmation

```text
creatures>0
confirm=false
→ ErrConfirmationRequired
→ Portal remains OPEN
```

### Creatures with confirmation

```text
creatures>0
confirm=true
→ CLOSED / MANUAL_CLOSE
```

### Terminal Portal

```text
CLOSED/COLLAPSED
→ ErrPortalNotOpen
```

Stage 3/4 later extends close orchestration to active Observer transit.

---

# 17. Substage 2.9 — Risk

Requirements:

```text
RISK-001..010
```

## 17.1 Energy lifetime

Test:

```text
Energy=50
Decay=.5/sec
→ 100 sec
```

```text
Energy=10
Decay=1/sec
→ 10 sec
```

## 17.2 Effective lifetime

```text
scheduled=80
energy lifetime=100
→ 80
```

```text
scheduled=120
energy lifetime=10
→ 10
```

## 17.3 Base formula boundary values

With stable Portal:

Formula:

```text
base = max(0, (45-effective)/45*100)
```

Key exact boundaries:

```text
effective >=45 sec
→ score 0

effective 33.75 sec
→ score 25

effective 22.5 sec
→ score 50

effective 11.25 sec
→ score 75
```

Risk levels:

```text
score ==25  → LOW
score >25   → MEDIUM

score ==50  → MEDIUM
score >50   → HIGH

score ==75  → HIGH
score >75   → CRITICAL
```

Use table-driven tests.

Suggested:

```go
TestPortal_RiskLevelBoundaries
```

## 17.4 Agreed examples

```text
effective=14 sec, STABLE
→ ~68.89
→ HIGH
```

```text
effective=10 sec, STABLE
→ ~77.78
→ CRITICAL
```

## 17.5 Instability penalty

Example:

```text
effective=30 sec
base≈33.33
STABLE → MEDIUM
UNSTABLE → ~53.33 → HIGH
```

Test that hidden collapse timestamp value itself does not change risk.

Two otherwise identical UNSTABLE portals with very different hidden collapse timestamps MUST have same Risk.

---

# 18. Risk for terminal Portals

Decision fixed.

Risk exists only for:

```text
OPEN Portal
```

For:

```text
CLOSED
COLLAPSED
```

current Risk is considered not applicable.

Portal Details for terminal Portals show:

- final Status;
- termination reason;
- final history;
- relevant recorded events;

but do not show a current LOW/MEDIUM/HIGH/CRITICAL value.

Preferred domain API:

```go
func (p Portal) RiskLevel(now time.Time, cfg Config) (RiskLevel, bool)
```

where:

```text
bool = false
```

for terminal Portals.

Do not fabricate LOW risk for CLOSED/COLLAPSED state.

---

# 19. Substage 2.10 — Portal Slots

Requirements:

```text
SLOT-001..007
```

Stage 2 should implement slot selection as pure/domain logic, not full LabManager.

Recommended API:

```go
func FirstFreeSlot(portals []Portal, maxSlots int) (slot int, ok bool)
```

Only `OPEN` Portals occupy Slots.

## RED tests

### Empty

```text
no OPEN portals
→ slot1
```

### Slots 1,2 occupied

```text
→ slot3
```

### Gap

```text
1,2,4 occupied
→ slot3
```

### Terminal records don't occupy

```text
Portal CLOSED with Slot1
Portal OPEN with Slot2
→ first free =1
```

### Full

```text
OPEN slots1..7
→ no free slot
```

### Portal retains assigned slot

No code should recalculate an existing OPEN Portal's `SlotIndex` merely because earlier slots became free.

This is primarily tested later in LabManager, but Portal/slot helper must not imply re-sorting.

---

# 20. Natural Portal factory

A factory can be introduced after core calculations are green.

Proposed responsibility:

```text
given:
- Plane ID
- Slot
- now
- Config
- Random

create valid NATURAL Portal
```

Generate:

```text
TTL
initial Energy
decay
stability
hidden collapse (if unstable)
creatures initial
```

It MUST NOT:
- choose Plane;
- choose Slot;
- mutate Lab state;
- write DB;
- emit WebSocket;
- run events.

Those belong to later orchestration.

## Factory tests

Using FakeRandom:

- Energy within config.
- Decay within config.
- stability generated deterministically.
- creature max respects TTL margin.
- unstable has hidden collapse.
- stable does not.
- Flow starts NONE.
- status OPEN.
- kind NATURAL.

Factory depends on open decision S2-D1 for hidden collapse range.

---

# 21. Domain errors required by end of Stage 2

At least:

```go
ErrPortalNotOpen
ErrPortalAlreadyStable
ErrPortalOverchargeRisk
ErrConfirmationRequired
ErrNoFreePortalSlot
```

Potential validation errors for invalid construction may be added if constructors validate config.

Errors should be stable domain values/types suitable for later HTTP 409 mapping.

Do not put HTTP status codes in domain package.

---

# 22. Mutation safety

All mutation methods should satisfy:

```text
On rejected action:
state remains unchanged.
```

Mandatory regression tests should snapshot Portal before action and verify equality after rejected mutation, excluding no fields.

Examples:

- Stabilize >85 rejected.
- Stabilize stable rejected.
- Close without required confirmation rejected.
- Close terminal rejected.

This is important before concurrency is added later.

---

# 23. Float handling

Portal Energy and internal Risk use floating point.

Rules:

- Domain tests use epsilon/tolerance.
- UI later rounds Energy to one decimal.
- Domain should not round Energy every tick.
- Risk classification uses full internal precision.
- Do not persist/display floating arithmetic artifacts.

Suggested tolerance:

```text
1e-9 or require.InDelta(..., 1e-6)
```

---

# 24. Time semantics

Use `time.Time` / `time.Duration`.

Tests always use fixed deterministic base time, e.g.:

```go
base := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
```

Avoid tests dependent on local timezone.

No `time.Sleep`.

---

# 25. TDD commit/checkpoint sequence

Recommended checkpoints:

## Checkpoint A — state + natural close

GREEN:
- terminal state;
- natural close;
- semantic close timestamp.

## Checkpoint B — energy

GREEN:
- current Energy;
- energy depletion timestamp;
- Energy collapse;
- natural/energy tie semantics.

## Checkpoint C — stability

GREEN:
- stable/unstable;
- hidden collapse;
- natural/hidden ordering;
- Stabilize.

## Checkpoint D — creatures + manual close

GREEN:
- clearance margin;
- derived creature count;
- close confirmation.

## Checkpoint E — Risk

GREEN:
- lifetime;
- formula;
- boundaries;
- instability penalty.

## Checkpoint F — Slots + factory

GREEN:
- first-free slot;
- max 7 helper;
- deterministic Natural Portal creation.

After every checkpoint:

```bash
go test ./...
go test -race ./...
```

Update traceability.

---

# 26. Traceability update

By Stage 2 completion, `docs/traceability.md` should map every requirement in scope to at least one test.

Example:

| ID | Test | Status |
|---|---|---|
| PORTAL-005 | `TestPortal_NaturalCloseAtScheduledTime` | GREEN |
| ENERGY-005 | `TestPortal_CollapsesWhenEnergyDepletesBeforeNaturalClose` | GREEN |
| STABILITY-005 | `TestPortal_StabilizeConvertsUnstableToStable` | GREEN |
| CREATURE-005 | `TestCreatureLimit_TTL10AllowsMaxFour` | GREEN |
| RISK-009 | `TestPortal_RiskLevelBoundaries/critical` | GREEN |
| SLOT-006 | `TestFirstFreeSlot_ReturnsNoneWhenAllSevenOpen` | GREEN |

---

# 27. AI Worklog requirements during Stage 2

Do not wait until the stage ends.

Record real examples of:

- prompts used to generate tests;
- tests suggested by AI that were useful;
- tests suggested by AI that contradicted Final Spec;
- domain APIs generated by AI and later simplified;
- edge cases found manually;
- failing tests exposing implementation mistakes;
- any disagreement over tie semantics;
- refactors after GREEN.

Especially note if AI:
- adds stale baseline instead of current Energy;
- stores Risk instead of deriving it;
- treats Natural Close as Collapse;
- lets terminal Portal mutate;
- ignores creature margin;
- leaks hidden collapse into Risk.

---

# 28. Stage 2 non-goals

Do NOT implement yet:

- Lab Energy deductions;
- Observer lifecycle;
- Observer direction actions;
- Event persistence;
- SQLite;
- REST;
- WebSocket;
- real 1-sec simulation goroutine;
- Tutorial;
- frontend;
- deployment.

It is acceptable to define domain errors/types that later stages will use.

---

# 29. Stage 2 Definition of Done

Stage complete only when:

## Portal lifecycle
- Natural Close deterministic.
- Energy Collapse deterministic.
- Instability Collapse deterministic.
- terminal state immutable.

## Energy
- derived from baseline/time.
- clamp >=0.
- Stabilize re-baselines correctly.
- stale-baseline bug covered.

## Stability
- STABLE/UNSTABLE semantics correct.
- hidden timer removed by Stabilize.
- overcharge boundary 85 tested.

## Creatures
- 2 sec transit.
- 2 sec clearance margin.
- TTL10→max4 tested.
- derived current count correct.
- close confirmation works.

## Risk
- effective lifetime formula tested.
- agreed 14 sec HIGH and 10 sec CRITICAL examples tested.
- exact 25/50/75 boundaries tested.
- hidden instability timestamp does not affect Risk.

## Slots
- first-free logic correct.
- 7/7 full state detected.
- terminal Portal releases slot logically.
- no implicit Portal reordering.

## Quality
- all tests GREEN.
- `go test ./...` passes.
- `go test -race ./...` passes.
- traceability updated.
- no `time.Sleep`.
- no real random in domain tests.
- no DB/HTTP dependencies.
- real AI Worklog notes appended.

---

# 30. Resolved Stage 2 decisions

No known blocking design decisions remain for Stage 2.

Resolved:

## S2-D1 — hidden instability timestamp

```text
random(
    opened_at + 5 sec,
    scheduled_close_at - 1 sec
)
```

## S2-D2 — Risk for terminal Portal

Risk is only defined for OPEN Portal.

CLOSED/COLLAPSED return no current Risk Level.

Stage 2 can proceed to implementation without additional domain decisions.

---

# 31. Next stage after completion

Stage 3:

```text
Observer lifecycle via TDD

AVAILABLE
→ OUTBOUND
→ EXPLORING
→ WAITING_RETURN
→ RETURNING
→ AVAILABLE / LOST
```

It will build on the stable Portal Core without changing Portal semantics.
