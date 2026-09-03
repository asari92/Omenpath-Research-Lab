# Omenpath Research Lab — Stage 0–1
## Executable Specification + TDD Foundation

## Goal

До Portal business implementation:
1. превратить Final Spec в requirement catalog;
2. обеспечить deterministic time/random tests;
3. создать Go skeleton;
4. подготовить traceability;
5. написать первые RED tests.

## Source of truth

```text
00_FINAL_SPEC_v5.md
→ Requirement Catalog
→ Tests
→ Implementation
```

Нельзя менять test, чтобы подогнать его под код, если requirement остался прежним.

## Requirement types

- **Invariant** — не может нарушаться.
- **Behavior** — transition/action/time behavior.
- **Balance Config** — тюнимый параметр.
- **UI Contract** — shape/visibility of authoritative state.

## Requirement ID families

```text
PLANE
PORTAL
SLOT
ENERGY
STABILITY
CREATURE
RISK
LAB
OBSERVER
FLOW
EXTRACTION
EMERGENCY
EVENT
TUTORIAL
API
WS
UI
PERSIST
```

## Initial requirement catalog

### PLANE
- PLANE-001 Plane independent from Portal.
- PLANE-002 Plane has 0..N Portals.
- PLANE-003 multiple OPEN Portals may target same Plane.
- PLANE-004 seed default UNEXPLORED.
- PLANE-005 SEND does not explore.
- PLANE-006 arrival does not explore.
- PLANE-007 research completion does not explore.
- PLANE-008 successful return explores.
- PLANE-009 already explored is idempotent.

### PORTAL
- PORTAL-001 unique instance.
- PORTAL-002 sequential `Omenpath #XXXX`.
- PORTAL-003 NATURAL/EXTRACTION kinds.
- PORTAL-004 OPEN/CLOSED/COLLAPSED statuses.
- PORTAL-005 TTL → CLOSED/NATURAL_CLOSE.
- PORTAL-006 manual Close → CLOSED/MANUAL_CLOSE.
- PORTAL-007 Energy0 → COLLAPSED/ENERGY_DEPLETED.
- PORTAL-008 hidden instability time → COLLAPSED/INSTABILITY.
- PORTAL-009 terminal state cannot become OPEN again.

### SLOT
- SLOT-001 exactly 7 Slots.
- SLOT-002 OPEN Portal occupies one.
- SLOT-003 new Portal takes first free Slot.
- SLOT-004 Portal keeps Slot throughout lifecycle.
- SLOT-005 terminal Portal releases Slot.
- SLOT-006 7 OPEN blocks natural spawn.
- SLOT-007 Extraction consumes regular Slot.

### ENERGY
- ENERGY-001 initial natural 10..100.
- ENERGY-002 decay 0.1..1.0/sec.
- ENERGY-003 current energy derived.
- ENERGY-004 clamp >=0.
- ENERGY-005 0 causes Collapse.
- ENERGY-006 Stabilize +15.
- ENERGY-007 Stabilize creates new baseline.
- ENERGY-008 decay unchanged after Stabilize.
- ENERGY-009 >85 rejects Stabilize.
- ENERGY-010 exactly85 allowed to100.

### STABILITY
- STABILITY-001 STABLE/UNSTABLE.
- STABILITY-002 stable no hidden timer.
- STABILITY-003 unstable has hidden timer.
- STABILITY-004 hidden timer not exposed.
- STABILITY-005 Stabilize unstable→stable.
- STABILITY-006 Stabilize clears hidden timer.
- STABILITY-007 stable cannot be stabilized again.

### CREATURE
- CREATURE-001 natural 0..10.
- CREATURE-002 passage2 sec.
- CREATURE-003 margin2 sec.
- CREATURE-004 max formula.
- CREATURE-005 TTL10→max4.
- CREATURE-006 current count derived.
- CREATURE-007 creatures block SEND/RECALL.
- CREATURE-008 Close requires confirmation.

### RISK
- RISK-001 energy lifetime formula.
- RISK-002 effective lifetime=min(time,energy lifetime).
- RISK-003 base risk formula using 45 sec horizon.
- RISK-004 unstable +20.
- RISK-005 max100.
- RISK-006 0..25 LOW.
- RISK-007 >25..50 MEDIUM.
- RISK-008 >50..75 HIGH.
- RISK-009 >75..100 CRITICAL.
- RISK-010 hidden instability time not used.
- RISK-011 UI receives level, not score.
- RISK-012 critical blocks SEND.
- RISK-013 critical blocks RECALL.

### LAB
- LAB-001 range0..100 integer.
- LAB-002 initial100.
- LAB-003 regen+1/sec.
- LAB-004 max100.
- LAB-005 SEND0.
- LAB-006 RECALL0.
- LAB-007 CLOSE5.
- LAB-008 STABILIZE20.
- LAB-009 EXTRACTION30.
- LAB-010 insufficient energy rejects paid action.

### OBSERVER
- OBSERVER-001 exactly10.
- OBSERVER-002 initial AVAILABLE.
- OBSERVER-003 AVAILABLE means Lab.
- OBSERVER-004 main lifecycle.
- OBSERVER-005 LOST terminal.
- OBSERVER-006 transit5..15.
- OBSERVER-007 duration fixed once started.
- OBSERVER-008 outbound completion→EXPLORING.
- OBSERVER-009 research20 sec.
- OBSERVER-010 research completion→WAITING_RETURN.
- OBSERVER-011 return completion→AVAILABLE.
- OBSERVER-012 Portal CLOSED during transit→LOST.
- OBSERVER-013 Portal COLLAPSED during transit→LOST.
- OBSERVER-014 multiple Observers allowed per Plane.
- OBSERVER-015 sending to explored allowed.
- OBSERVER-016 RECALL picks longest waiting.

### FLOW
- FLOW-001 natural starts NONE.
- FLOW-002 first SEND→OUTBOUND.
- FLOW-003 first RECALL→INBOUND.
- FLOW-004 OUTBOUND rejects RECALL.
- FLOW-005 INBOUND rejects SEND.
- FLOW-006 one simultaneous transit per Portal.
- FLOW-007 next transit allowed after previous ends, same direction.

### EXTRACTION
- EXTRACTION-001 cost30.
- EXTRACTION-002 requires waiting observer.
- EXTRACTION-003 requires free Slot.
- EXTRACTION-004 stable/inbound/creatures0.
- EXTRACTION-005 Energy60..100.
- EXTRACTION-006 TTL30..60.
- EXTRACTION-007 sync5.
- EXTRACTION-008 first observer auto-returns.
- EXTRACTION-009 only one automatic.
- EXTRACTION-010 further returns manual.

### EMERGENCY
- EMERGENCY-001 Collapse→Lab Energy0.
- EMERGENCY-002 Collapse→Override20 sec.
- EMERGENCY-003 Override Close/Stabilize free.
- EMERGENCY-004 Extraction unchanged.
- EMERGENCY-005 regen continues.
- EMERGENCY-006 new Collapse resets energy/timer.

### EVENT
- EVENT-001 meaningful transition creates Event.
- EVENT-002 Risk Event only level change.
- EVENT-003 rejected action creates ACTION_REJECTED.
- EVENT-004 Portal History filters portal_id.
- EVENT-005 Global Log same event source.

### TUTORIAL
- TUTORIAL-001 state machine.
- TUTORIAL-002 advance only after expected condition.
- TUTORIAL-003 wrong reversible action stays step.
- TUTORIAL-004 wrong irreversible action recreates scenario.
- TUTORIAL-005 Stabilize tutorial guarantees HIGH→MEDIUM.
- TUTORIAL-006 Critical step expects rejected SEND.
- TUTORIAL-007 exploration only after successful return.
- TUTORIAL-008 Energy not reset entering Live.

## Traceability

Create:
```text
docs/requirements.md
docs/traceability.md
```

Traceability columns:
```text
ID | Rule | Type | Test | Implementation | Status | Notes
```

Status:
```text
PLANNED
RED
GREEN
REFACTORED
```

## Go skeleton

```text
omenpath-lab/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── plane.go
│   │   ├── portal.go
│   │   ├── observer.go
│   │   ├── lab.go
│   │   ├── event.go
│   │   ├── risk.go
│   │   └── errors.go
│   ├── engine/manager.go
│   ├── clock/clock.go
│   ├── random/random.go
│   └── config/config.go
├── testutil/
│   ├── clock.go
│   ├── random.go
│   └── builders.go
├── docs/
│   ├── requirements.md
│   └── traceability.md
├── go.mod
└── README.md
```

No API/WS/SQLite/frontend implementation in this stage.

## Clock abstraction

Production domain code does not call `time.Now()` directly.

```go
type Clock interface {
    Now() time.Time
}
```

Production `RealClock`.

Tests `FakeClock`:
```go
clock.Advance(10 * time.Second)
```

No `time.Sleep` in domain tests.

## Random abstraction

```go
type Random interface {
    IntInclusive(min, max int) int
    FloatRange(min, max float64) float64
}
```

Production real PRNG.

FakeRandom provides deterministic preloaded values.

No real random in domain tests.

## Test builders

Create readable fixture helpers/builders.

Example:
```go
portal := NewPortalBuilder().
    Stable().
    Energy(80).
    Decay(0.5).
    TTL(60*time.Second).
    Build()
```

Builders are test helpers, not alternative domain constructors used in production.

## Testing dependencies

Use:
```text
testing
github.com/stretchr/testify/require
```

No heavy framework.

## Naming

Behavior-oriented:
```go
TestPortal_NaturalCloseWhenTTLExpires
TestObserver_BecomesLostWhenPortalClosesDuringTransit
TestRecall_IsRejectedForOutboundPortal
```

## Table-driven tests

Especially:
- risk bands/boundaries;
- energy thresholds;
- TTL/creature boundary;
- stabilization 85/85+;
- transit random boundaries.

## First RED cycle

Write before initial Portal behavior implementation:

1. `PORTAL-005`
```text
expired TTL → CLOSED + NATURAL_CLOSE
```

2. `ENERGY-005`
```text
Energy 0 → COLLAPSED + ENERGY_DEPLETED
```

3. `PORTAL-009`
```text
terminal Portal cannot return OPEN
```

Only then minimal Portal implementation.

## Quality gates

Must pass for foundation:
```bash
go test ./...
go test -race ./...
```

## Do NOT implement in Stage 0–1

- HTTP handlers;
- chi routes;
- WebSocket;
- SQLite repository/migrations;
- real simulation goroutine;
- frontend;
- Tutorial UI;
- deployment.

## Stage 0–1 DoD

- Final Spec recognized as canonical.
- Requirement Catalog exists.
- Traceability exists.
- Domain rules separated from balance config.
- Go module/skeleton exists.
- Clock/RealClock/FakeClock works.
- Random/real/fake works.
- Test builder foundation works.
- Tests pass.
- Race detector passes.
- First RED Portal tests can be written with no sleep, DB, HTTP or real random.

## Next detailed stage

Stage 2 — Portal Core via TDD:
```text
Portal State
→ Natural Close
→ Portal Energy
→ Energy Collapse
→ Stability
→ Hidden Collapse
→ Stabilize
→ Creatures
→ Risk
→ Portal Slots
```
