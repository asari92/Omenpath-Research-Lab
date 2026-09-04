# Stage 8 Simulation Design

**Date:** 2026-09-04  
**Status:** Approved for detailed TDD planning  
**Scope:** deterministic simulation aggregate, natural Portal generator, and
Needs Attention selection

## 1. Source and intent

`00_FINAL_SPEC_v5.md` remains the sole product/domain source of truth. This
design resolves technical omissions needed to implement roadmap Stage 8
without starting Events, persistence, transport, or concurrent management.

Stage 8 must provide deterministic domain processing for:

- the one-second simulation tick order from Final Spec §33;
- Natural Portal delay `random(0..20 sec)` and re-scheduling from §7;
- the hard maximum of seven OPEN Portals from §§5 and 7;
- random destination selection among the canonical 85 Planes;
- Needs Attention selection from §6.

The configured delay remains `0..20`, as required by Final Spec and
`config.Default()`. A zero delay never creates multiple Portals in one tick.

## 2. Chosen architecture

Introduce a pure domain-level `SimulationState` aggregate. It owns the active
Lab, Portal, Plane and Observer values needed by one deterministic tick, plus
technical scheduler state:

- next shared Portal sequence;
- Natural spawn scheduled-at and due-at timestamps;
- whether Natural generation is paused by 7/7 occupancy;
- the last processed tick timestamp.

`ResolveTick(now, rnd, cfg)` receives time and randomness explicitly. It does
not call a wall clock, start a ticker, create a goroutine, acquire a mutex,
persist, emit Events, build a transport snapshot, or broadcast.

Stage 11 `LabManager` will later own `SimulationState` behind a mutex and call
this pure tick from the real clock. This avoids implementing Stage 11 early
and gives it one reusable atomic transition boundary.

Rejected alternatives:

1. Implementing the tick directly in `LabManager` would prematurely introduce
   Stage 11 ownership and concurrency semantics.
2. Implementing only disconnected spawn/attention helpers would leave tick
   ordering and cross-aggregate atomicity unproved.

## 3. State boundaries

The aggregate stores canonical state and scheduling baselines only. It does
not persist derived realtime values such as current Portal Energy, creatures,
risk, effective lifetime, current Lab Energy, or Needs Attention.

Natural scheduler state distinguishes:

- **scheduled:** both scheduling origin and due timestamp exist;
- **paused:** no deadline exists because all seven Slots are occupied.

Invalid combinations are structural invariant errors. Initial Live bootstrap
will eventually create a scheduled state, but Tutorial/Live mode transitions
remain outside Stage 8.

The next Portal sequence is shared in intent across Natural and Extraction
openings. Stage 8 consumes it for Natural spawn. Stage 11 must use the same
counter when routing Extraction commands, so global uniqueness remains
PARTIAL until that command ownership exists.

## 4. Tick semantics

One call processes at most one logical tick and uses copy-then-commit across
the whole aggregate. Repeating the same timestamp is an unchanged no-op.
Moving backward from the last processed timestamp is an invariant error.
Large forward jumps perform one orchestration pass at `now`; existing domain
resolvers retain semantic transition timestamps and catch up their own phases.
The scheduler never manufactures multiple missed Natural spawns.

The Stage 8 subset of Final Spec §33 runs in this order:

1. validate the complete aggregate and config without randomness;
2. derive current Laboratory Energy;
3. resolve due Portal terminal transitions and Collapse/Leyline effects;
4. resolve Observer transit and research;
5. resolve Extraction synchronization;
6. derive current Portal Energy, creatures and Risk;
7. process the Natural spawn timer, creating at most one Portal;
8. derive Needs Attention from the resulting OPEN Portal set;
9. commit the working copy and last-tick timestamp.

Final Spec's Portal Energy, creatures and risk steps are derived reads, not
per-second writes. Natural Close, energy depletion and instability are already
one semantic lifecycle decision in `Portal.ResolveLifecycle`; Stage 8 must not
split or redefine the tie rules completed in Stage 2.

Due Portal transitions across different instances are applied by semantic
transition timestamp and then lower Portal ID. Observer and Extraction work is
visited by ascending entity/Portal ID without reordering stored slices.

Observer lifecycle runs before Extraction synchronization. Therefore an
Observer whose research completes at this tick is eligible for an Extraction
sync at the same timestamp, exactly matching the §33 order.

## 5. Natural generation

Scheduling a delay draws one whole-second value inclusively from
`cfg.SpawnDelayMin..cfg.SpawnDelayMax`. The due timestamp is the scheduling
origin plus that delay.

A scheduled spawn is eligible only on a tick later than its scheduling origin
and at or after its due timestamp. Consequently, delay zero means “the next
tick,” not another spawn in the current tick.

At the Natural spawn step:

1. Count only OPEN Portals.
2. If count is seven, enter paused state, clear any old deadline, and consume
   no random value.
3. If generation was paused and a Slot is now free, draw a fresh delay from
   `now`; do not spawn during this same tick, even when the delay is zero.
4. If a scheduled deadline is not due, do nothing.
5. If it is due, re-check the seven-OPEN limit, select the first free regular
   Slot, choose one destination uniformly by index from the validated 85 Plane
   roster, and create exactly one `NewNaturalPortal` at `now`.
6. Consume and increment the shared next Portal sequence once.
7. If the spawn filled the seventh Slot, enter paused state without drawing a
   throwaway delay. Otherwise draw the next delay from `now`.

Terminal Portal records remain in the slice but do not occupy Slots. Multiple
OPEN Portals may choose the same Plane. No destination de-duplication exists.

Deterministic random order within a tick is:

1. Extraction transit durations for synchronizing Extraction Portals in
   ascending Portal ID;
2. Natural destination index when a spawn is due;
3. the existing `NewNaturalPortal` draw sequence;
4. the next Natural delay, only when fewer than seven OPEN Portals remain.

Rejected/no-op paths consume none of these values.

## 6. Needs Attention

Needs Attention is a pure selector over OPEN Portals at `now`. It returns no
selection when none are OPEN and never mutates or sorts the stored Portal
slice.

Priority is exactly:

1. highest internal `risk_score`;
2. exact score tie: UNSTABLE first;
3. tie: lower `effective_lifetime`;
4. tie: older `opened_at`;
5. complete technical tie: lower Portal ID.

The final ID tie-breaker only makes otherwise indistinguishable state
deterministic. Numeric risk remains internal; Stage 8 returns only the selected
Portal identity/index needed by later snapshot construction.

Needs Attention is computed after Natural spawn, so a newly created Portal may
be selected immediately. Terminal Portals are never candidates.

## 7. Validation and atomicity

Before any random draw or mutation, validate:

- non-negative/ordered Stage 8 config durations and limits;
- exactly 85 unique canonical Plane IDs;
- unique Portal IDs and legal next sequence;
- unique OPEN Slot occupancy inside `1..MaxActivePortals`;
- Portal-to-Plane references;
- canonical Observer state and references;
- Natural scheduler field combinations;
- monotonic tick time.

All lifecycle, extraction, spawn and scheduler work occurs on copies. Any
error discards the complete working state. Preflight must make structural and
config failures observable before randomness is consumed.

Stage 8 adds a stable simulation invariant error. It does not introduce
ACTION_REJECTED, Event values, logging callbacks, retries, recovery, or partial
commits.

## 8. TDD decomposition

The detailed plan should use checkpoint-sized RED/GREEN cycles:

1. Simulation/scheduler state and structural validation.
2. Needs Attention priority and non-mutation.
3. Initial delay plus 7/7 pause/resume behavior.
4. Natural spawn, destination, first Slot and shared sequence.
5. Portal lifecycle and multi-Collapse/Leyline ordering.
6. Observer transit/research and Plane exploration within a tick.
7. Extraction synchronization ordering and random discipline.
8. Full-tick idempotence, atomicity, late-time and completed-stage regressions.

Every meaningful GREEN runs `gofmt -l .`, `go vet ./...`, `go build ./...`,
`go test -count=1 ./...`, and `go test -race -count=1 ./...`. RED and GREEN
commits remain separate.

## 9. Requirements and traceability

The requirement catalog currently represents Stage 8 only indirectly through
`SLOT-006`, `PORTAL-001/002`, `PLANE-003`, `UI-007`, and `WS-002`. The Stage 8
implementation pass should add catalog/traceability rows derived directly from
Final Spec, without changing product semantics:

- `SPAWN-*` for delay, repeat scheduling, 7/7 pause/resume, destination and
  one-spawn-per-tick behavior;
- `SIMULATION-*` for deterministic ordered/atomic tick processing;
- `ATTENTION-*` for the backend priority selector.

UI rendering and WebSocket cadence remain PLANNED. `SLOT-006` can become GREEN
for backend generator behavior; UI slot requirements retain their frontend
boundary. `PLANE-004` becomes PARTIAL for validated 85-Plane cardinality but
awaits canonical seed/bootstrap in Stage 10. Global Portal sequence remains
PARTIAL until Stage 11 serializes both Natural and Extraction openings.

## 10. Explicit exclusions

Stage 8 does not implement:

- domain Events, risk-change Events, Portal History or ACTION_REJECTED;
- SQLite, migrations, seed persistence or restart recovery;
- `LabManager` mutex ownership or REST/simulation concurrency;
- actual `time.Ticker`, goroutines, wall-clock reads or sleeps;
- state snapshot DTOs, WebSocket broadcasts or frontend Needs Attention;
- REST commands, Tutorial/Live state machine, or balance tuning.

No Stage 9 behavior begins in the Stage 8 implementation pass.

## 11. Resolved decisions

- **S8-D1:** Keep the Final Spec/configured spawn delay at inclusive `0..20`
  seconds.
- **S8-D2:** Process at most one Natural spawn per tick; delay zero becomes
  eligible only on a later tick.
- **S8-D3:** Re-check and enforce `MaxActivePortals == 7` before every spawn.
- **S8-D4:** At 7/7, discard the timer without a replacement draw; after a
  Slot frees, schedule a fresh delay and wait for a later tick.
- **S8-D5:** Use a pure domain aggregate now; real ticker/mutex ownership stays
  in Stage 11.
- **S8-D6:** Preserve Stage 2 semantic lifecycle ordering instead of deriving
  outcomes from slice or tick traversal order.
- **S8-D7:** Observer lifecycle precedes Extraction sync; Natural spawn and
  Needs Attention follow both.
- **S8-D8:** Lower Portal ID breaks a complete Needs Attention tie.
- **S8-D9:** Tick is atomic copy-then-commit and deterministic under supplied
  time/randomness.
- **S8-D10:** Stage 8 adds missing requirement IDs only as a catalog mapping of
  existing Final Spec rules.
