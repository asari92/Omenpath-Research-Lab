# Stage 8 Corrective Pass — Design

## Purpose

Correct the three verified Stage 8 integration defects before Stage 9 starts,
without broadening the pass into Events, persistence, LabManager, transport or
frontend work. The Final Spec remains the product source of truth.

## Scope

This pass owns:

- chronological `Plane.ExploredAt` when multiple Observer returns are resolved
  by one late tick;
- late-tick coordination of multiple due Extraction synchronizations;
- complete Stage 8 aggregate validation for Portal and Observer canonical
  fields;
- consistency updates to the Stage 8 plan, requirements, traceability and
  worklog;
- explicit tracking of the known Recommendation Engine contract and its
  product-definition gate.

This pass does not implement Stage 9, restart recovery, a real one-second
ticker, concurrency, persistence, APIs, WebSockets, UI, or the Recommendation
selection algorithm.

## Chosen approach

Use a targeted correction that preserves the public domain APIs and the Final
Spec tick-stage order. Do not replace the tick with a global chronological
event queue: that would overlap Stage 9 and change substantially more completed
behavior than the verified defects require.

### Earliest exploration timestamp

Observer traversal remains deterministic and does not reorder the stored
slice. For every Plane that is UNEXPLORED at the start of the Observer stage,
the stage records all successful RETURNING completions resolved by that tick.
After traversal, the Plane receives the earliest semantic return deadline.

An already EXPLORED Plane retains its original `ExploredAt`, including when a
later tick resolves another return whose stored deadline is older. This
preserves the established repeat-return idempotence contract while removing
Observer-ID dependence for simultaneous catch-up.

### Multiple late Extraction synchronizations

Strict stale-transit validation remains part of public SEND/RECALL command
admission. The Simulation tick already performs complete aggregate preflight
and therefore uses a prepared Extraction synchronization path that does not
re-run command-time freshness checks after each earlier synchronization has
mutated the working roster.

Extraction Portals remain visited by ascending Portal ID. Every due Portal
reselects the current longest-waiting Observer. An Observer selected by an
earlier Portal is RETURNING and cannot be selected again, so multiple due
Portals select distinct candidates while candidates exist.

The Observer stage remains before the Extraction stage. Consequently, a return
started during a late Extraction stage remains RETURNING until the next tick,
even when its semantic transit deadline is already at or before the current
`now`. The following tick catches it up. This follows the defined stage order
and avoids introducing a second Observer pass.

The public `ResolveExtractionSynchronization` contract remains strict and
atomic for direct callers. Shared internal preparation code may be extracted,
but the command path must retain all Stage 7 error identities and random-draw
behavior.

### Aggregate validation

`SimulationState.ResolveTick` rejects malformed state before mutation or random
consumption. Portal validation will cover:

- positive identity, destination reference, and Slot in `1..7` for historical
  as well as OPEN records;
- recognized kind, status, stability, flow and termination enums;
- ordered creation/opening/update, energy-baseline, lifecycle and terminal
  timestamps relative to the requested tick time;
- positive decay, bounded energy and creature baselines;
- STABLE implies no hidden collapse timestamp, including after stabilization;
  UNSTABLE requires a hidden timestamp inside its legal lifetime window;
- NATURAL has no Extraction marker;
- EXTRACTION is STABLE, INBOUND and creature-free;
- a completed Extraction marker equals `OpenedAt + ExtractionSync`;
- OPEN records have no terminal fields, while terminal status, reason and
  `ClosedAt` agree with the applicable lifecycle outcome.

Observer validation will retain due phases as legal input while checking:

- positive unique identity and resolvable Plane/Portal references;
- canonical status-dependent optional fields;
- ordered `CreatedAt`, `UpdatedAt`, `PhaseStartedAt` and `PhaseEndsAt` values;
- phase starts are not in the future, while phase ends may be due;
- OUTBOUND and RETURNING active portals have the compatible permanent flow;
- RETURNING Plane matches the active Portal destination.

Validation must continue to accept terminal Portals referenced by unresolved
transits, because the ordered tick needs to turn those Observers into LOST.

## Documentation consistency

The Stage 8 execution plan will be corrected so Observer-ID traversal is no
longer allowed to determine the exploration timestamp. Its invariant and test
sections will describe the new regression cases.

`docs/requirements.md` will distinguish global Extraction availability from
selected-Plane eligibility. Traceability statuses and notes affected by the
three defects will be made conservative until their new tests are GREEN.
`01_AI_WORKLOG_CURRENT.md` will record the audit, RED/GREEN commits, actual
implementation choices and final verification.

The known Recommendation contracts will be catalogued as PLANNED:

- deterministic and without a runtime LLM;
- limited to the Final Spec output enum;
- informational rather than an action restriction;
- exposed only in Portal Details.

The selection decision table is absent from the Final Spec. Before the Stage 12
or Stage 17 detailed plan may implement Recommendation, the product semantics
must be approved and added to the Final Spec. This corrective pass does not
invent that algorithm.

## TDD and commit structure

Use isolated RED/GREEN history for:

1. earliest exploration timestamp;
2. multiple late Extraction synchronization;
3. canonical aggregate validation.

Each RED commit contains only compiling tests and necessary documentation of
the expected rule. Each GREEN commit contains the minimal production change.
After all three cycles, run focused tests, the full tests, race detector,
formatting, vet and build. Then update traceability/worklog in a final docs
commit. Stage 9 remains unstarted.

## Acceptance criteria

- A late tick with two successful returns to one previously unexplored Plane
  records the earlier return deadline regardless of Observer IDs or slice
  order.
- A previously explored Plane never has its established exploration timestamp
  rewritten by a repeated return.
- Multiple due Extraction Portals cannot fail merely because an earlier Portal
  in the same Extraction stage created a transit whose semantic end is already
  due.
- Each due Extraction Portal remains one-shot and each automatic return selects
  a currently waiting Observer at most once.
- Every canonical invariant promised by the Stage 8 plan has a rejection test;
  malformed input is atomic and draw-free.
- Existing Stage 0–8 tests and public domain error contracts remain GREEN.
- Recommendation remains explicitly blocked on product definition rather than
  receiving invented behavior.
- No Stage 9 implementation is introduced.
