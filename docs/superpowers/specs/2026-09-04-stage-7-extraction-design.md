# Stage 7 Extraction Portal Design

## Status and authority

This design records the Stage 7 decisions approved on 2026-09-04. It is
subordinate to `00_FINAL_SPEC_v5.md`; the detailed implementation plan must
stop if it discovers a real conflict with that document.

Stage 7 implements only the domain behavior in Final Spec §20 and the related
balance and lifecycle rules in §§5, 14, 19, 21, 22 and 37. Simulation, Events,
persistence, LabManager/concurrency, REST/WebSocket and UI remain later stages.

## Goal

Allow the user to select a Plane containing at least one `WAITING_RETURN`
Observer, spend 30 Laboratory Energy, and open a regular-slot Extraction
Portal that synchronizes for five seconds and then automatically begins at
most one return transit.

## Domain model

`Portal` gains one nullable current-state field:

```go
ExtractionSynchronizedAt *time.Time
```

It is `nil` while an Extraction Portal is synchronizing and is set to the
semantic synchronization deadline, `OpenedAt + cfg.ExtractionSync`, when the
one-shot synchronization attempt completes. It remains `nil` for Natural
Portals. This marker prevents a later resolver call from starting a second
automatic return.

No `WAITING_EXTRACTION` Observer status is introduced. Observers remain in the
existing `WAITING_RETURN` state until a return transit actually begins.

## Components

Stage 7 uses three focused domain operations:

1. `NewExtractionPortal` is a pure factory. It creates an OPEN, STABLE,
   INBOUND, creature-free Extraction Portal in a caller-supplied regular Slot
   and selected Plane. It draws TTL 30..60 seconds, Energy 60..100 and decay
   0.1..1.0 from the injected random source.
2. `OpenExtractionPortal` is the transaction boundary. It validates the Lab,
   selected Plane, Observer roster, current waiting eligibility, free Slot and
   configured cost before consuming randomness. It creates the Portal,
   appends it and commits the 30-point debit exactly once.
3. `ResolveExtractionSynchronization` is a one-shot lifecycle operation. At or
   after the five-second deadline it records synchronization and, when a
   current waiting Observer exists in the destination Plane, starts that
   Observer's RETURNING transit.

The opening operation receives the sequential Portal number from its caller,
matching the existing Natural Portal factory boundary. Stage 7 does not add a
global sequence generator or shared-state manager.

## Opening data flow

The opening command performs these steps in order:

1. Validate the Laboratory Energy baseline and calculate the derived balance
   at `now`.
2. Validate the selected Plane and Observer roster.
3. Require at least one current `WAITING_RETURN` Observer whose
   `CurrentPlaneID` equals the selected Plane ID.
4. Find the first regular free Slot with `FirstFreeSlot`.
5. Require the full configured Extraction cost, including during an active
   Leyline Override.
6. Draw deterministic Portal parameters and construct the Portal.
7. Commit the Laboratory Energy debit once and append the Portal once.

Any failure before step 7 leaves Lab, Portals and Observers unchanged and does
not consume random values. Opening does not reserve or mutate an Observer.

## Synchronization and automatic return

The synchronization interval is half-open:

```text
[OpenedAt, OpenedAt + ExtractionSync)
```

Before the deadline, resolution is a pure no-op. At the exact deadline or on a
late resolver call, the operation uses the deadline as the semantic transition
timestamp.

At synchronization time the operation selects again from the current roster;
it does not remember the Observer who was longest-waiting when the Portal was
opened. Selection uses the existing rule:

1. status is `WAITING_RETURN`;
2. `CurrentPlaneID` is the Extraction Portal destination;
3. earliest `PhaseStartedAt` wins;
4. an exact waiting-time tie uses the lowest Observer ID.

If the original longest-waiting Observer has already left through another
Portal but another waiting Observer remains, the current longest-waiting
Observer starts RETURNING automatically.

If no matching Observer remains, synchronization still completes, the marker
is set and no transit-duration random value is consumed. The cost is not
refunded, the Portal remains OPEN/INBOUND, and later returns through it are
manual only.

When an Observer is selected, setting `ExtractionSynchronizedAt` and starting
`WAITING_RETURN -> RETURNING` are one atomic state commit. The return transit
itself is not part of synchronization: it takes the existing random 5..15
seconds. Total time from Portal opening to a successful arrival is therefore
10..20 seconds.

Repeated synchronization resolution after the marker is set is idempotent: it
does not mutate state, draw randomness or start another automatic return.

## Manual recall and closure

Manual RECALL through an OPEN Extraction Portal is rejected with a stable
domain error while `ExtractionSynchronizedAt` is nil. Portal-not-open and
structural invariant errors retain their existing precedence. After
synchronization, manual RECALL delegates to the completed Stage 4/5 rules.

Closing an Extraction Portal during synchronization closes only the Portal.
No Observer is in transit through it, so Observers remain `WAITING_RETURN` in
their Planes and synchronization never completes.

Once synchronization has atomically started RETURNING, the existing active
transit close rule applies:

- without confirmation, Close is rejected without mutation;
- with confirmation, the Portal becomes CLOSED and the RETURNING Observer
  becomes LOST.

The existing Observer lifecycle also makes that Observer LOST if the Portal
later closes or collapses before the return transit deadline. Other
`WAITING_RETURN` Observers in the Plane are unaffected.

## Error and atomicity rules

Stage 7 reuses `ErrInsufficientLabEnergy`, `ErrNoFreePortalSlot`,
`ErrNoWaitingObserver`, Portal/Observer invariant errors and all existing
RECALL/Close errors. It adds a stable synchronization error for an otherwise
valid manual RECALL attempted before sync completion.

The opening and synchronization operations use preflight plus copy-then-commit
where multiple aggregates can change. A failed factory precondition, roster
invariant, energy check, synchronization transition or Observer transition
cannot leave a partial Lab debit, appended Portal, marker update or Observer
mutation.

## TDD structure

The implementation plan will use eight checkpoints:

1. Portal field and pure Extraction factory.
2. Plane selection and waiting-observer availability.
3. regular Slot allocation and transactional 30-point debit, including
   Leyline Override.
4. synchronization timing and semantic timestamps.
5. current longest-waiting automatic return and deterministic random use.
6. no-wait completion, one-shot idempotence and manual RECALL gating.
7. Close/loss integration before and after synchronization.
8. aggregate atomicity, regressions, traceability and scope audit.

Every missing behavior follows tests -> observed RED -> RED commit -> minimal
implementation -> GREEN -> full verification -> traceability -> GREEN commit.
Already-correct characterizations remain honest GREEN-only checkpoints.

## Expected traceability outcome

After Stage 7 domain completion:

- `EXTRACTION-001..010` are GREEN;
- `SLOT-007`, `LAB-009`, `LAB-010` and `EMERGENCY-004` are GREEN;
- related Portal, Flow and Observer rows gain concrete Stage 7 evidence;
- Stage 8+ requirements remain at their previous status.

`docs/requirements.md` changes only if comparison with Final Spec exposes an
actual requirements-index error. Stage 7 completion updates
`01_AI_WORKLOG_CURRENT.md` and `docs/traceability.md`.

## Non-goals

Stage 7 does not implement:

- Natural Portal spawn timing or the simulation tick;
- event creation or Extraction event types;
- persistence or restart recovery;
- mutexes, a LabManager or concurrent command serialization;
- API endpoints, WebSocket broadcasts or Plane-selection UI;
- any Stage 8 behavior.
