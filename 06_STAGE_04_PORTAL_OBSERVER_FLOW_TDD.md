# Omenpath Research Lab — Stage 4
## Portal Observer Flow and Command Admission via TDD

> Depends on:
> - `00_FINAL_SPEC_v5.md`
> - `02_IMPLEMENTATION_ROADMAP_TDD.md`
> - `04_STAGE_02_PORTAL_CORE_TDD.md`
> - `05_STAGE_03_OBSERVER_LIFECYCLE_TDD.md`
>
> This stage implements SEND/RECALL command admission, permanent Portal
> direction, one-transit-per-Portal enforcement, deterministic Observer
> selection, UNSTABLE confirmation, and manual-close interaction with an
> active transit.
>
> It does **not** implement Laboratory Energy, Leyline Override, Extraction,
> events, persistence, HTTP/WebSocket transport, simulation goroutines, or
> `LabManager` locking.

---

# 1. Goal

Build the pure domain orchestration that decides whether an Observer transit
may start through a Portal and, after every restriction succeeds, delegates the
actual lifecycle mutation to the Stage 3 primitives:

```text
SEND
→ validate current aggregate state
→ choose one AVAILABLE Observer
→ require UNSTABLE confirmation when applicable
→ set permanent OUTBOUND flow on first successful use
→ Observer.StartOutbound

RECALL
→ validate current aggregate state
→ choose the longest-waiting Observer in the destination Plane
→ require UNSTABLE confirmation when applicable
→ set permanent INBOUND flow on first successful use
→ Observer.StartReturning
```

Also complete manual-close interaction with an active transit:

```text
active OUTBOUND/RETURNING through Portal
+ manual Close without confirmation
→ rejected, no mutation

same state + confirmation
→ Portal CLOSED / MANUAL_CLOSE
→ transiting Observer LOST at close time
```

All Stage 4 behavior must remain deterministic and testable in memory with:

- fixed `now time.Time` values;
- `testutil.FakeRandom`;
- Stage 2 `Portal` calculations;
- Stage 3 Observer lifecycle primitives;
- no `time.Sleep`;
- no database, HTTP, goroutines, or real randomness.

---

# 2. Source-of-truth order

Use the repository hierarchy fixed in `AGENTS.md`:

```text
00_FINAL_SPEC_v5.md
→ 01_AI_WORKLOG_CURRENT.md (implementation history and recorded decisions)
→ 02_IMPLEMENTATION_ROADMAP_TDD.md
→ this Stage 4 detailed plan
→ docs/requirements.md
→ tests
→ implementation
```

The worklog records history and decisions but never overrides Final Spec. If
this plan conflicts with Final Spec, Final Spec wins. Do not change Stage 2
Portal lifecycle or Stage 3 Observer lifecycle semantics merely to simplify a
Stage 4 test.

---

# 3. Verified starting point

Stage 3 ended at commit `69c34a2` with:

```go
func (o *Observer) StartOutbound(
    now time.Time,
    portalID int64,
    rnd random.Random,
    cfg config.Config,
) error

func (o *Observer) StartReturning(
    now time.Time,
    portalID int64,
    rnd random.Random,
    cfg config.Config,
) error

func ResolveObserverLifecycle(
    o *Observer,
    plane *Plane,
    portal *Portal,
    now time.Time,
    cfg config.Config,
) error
```

Existing semantics that Stage 4 consumes rather than reimplements:

- transit duration is drawn once at start (`5..15 sec`);
- OUTBOUND starts only from AVAILABLE;
- RETURNING starts only from WAITING_RETURN;
- successful return explores Plane;
- Portal terminal strictly before transit end makes Observer LOST;
- `ClosedAt == PhaseEndsAt` is a successful transit;
- lifecycle resolver supports catch-up and is idempotent;
- LOST is terminal.

Stage 2 already supplies:

```go
Portal.ObserverFlow
Portal.Status / IsTerminal()
Portal.CreaturesInside(now, cfg)
Portal.RiskLevel(now, cfg)
Portal.Close(now, confirmCreatureInterrupt, cfg)
```

Stage 4 should add orchestration around these APIs, not duplicate their
formulas or transitions.

---

# 4. Requirements covered

Primary requirements:

```text
FLOW-001  NATURAL Portal starts NONE
FLOW-002  first SEND fixes OUTBOUND direction
FLOW-003  first RECALL fixes INBOUND direction
FLOW-004  OUTBOUND rejects RECALL
FLOW-005  INBOUND rejects SEND
FLOW-006  only one Observer in transit per Portal
FLOW-007  next transit allowed after previous ends; direction persists

CREATURE-007  creatures_inside > 0 blocks SEND and RECALL
RISK-012      CRITICAL blocks SEND
RISK-013      CRITICAL blocks RECALL

OBSERVER-015  SEND to an already EXPLORED Plane is allowed without an
              exploration-specific warning
OBSERVER-016  RECALL chooses the longest-waiting Observer

PLANE-005     real SEND command does not explore the Plane
PORTAL-006    manual Close remains CLOSED/MANUAL_CLOSE; Stage 4 adds the
              active-transit confirmation/loss orchestration from §21
OBSERVER-012  CLOSED during OUTBOUND/RETURNING transit → LOST
```

Direct Final Spec acceptance behavior without a separate current catalog ID:

```text
§18  SEND requires an AVAILABLE Observer
§18  SEND warning only for UNSTABLE
§18  no SEND warning for low energy, low remaining time, or explored Plane
§19  RECALL requires a WAITING_RETURN Observer in Portal destination
§19  RECALL warning only for UNSTABLE
§21  active transit requires Close confirmation
§29  confirmation retries the same command with confirm=true
```

These behaviors must still have named tests and traceability notes attached to
the closest existing requirement rows. Do not invent product behavior merely
to create another ID.

---

# 5. Hard stage boundary

Stage 4 answers:

```text
"May SEND / RECALL / manual Close start against this already-resolved
 current domain state, and which Observer is affected?"
```

Stage 4 does not answer:

```text
"How much Laboratory Energy does the action cost?"       # Stage 5
"How does Collapse activate Leyline Override?"           # Stage 6
"How is an Extraction Portal created/synchronized?"      # Stage 7
"When does the simulation tick run?"                     # Stage 8
"Which durable Events are emitted?"                      # Stage 9
"How are aggregates persisted/recovered?"                # Stage 10
"How are concurrent commands serialized?"               # Stage 11
"Which HTTP status/body is returned?"                    # Stage 12
```

Therefore Stage 4 MUST NOT:

- deduct or inspect Lab Energy;
- implement SEND/RECALL costs (both are already balance value 0);
- activate Leyline Override;
- create Extraction Portals or automatic extraction returns;
- emit/persist Events;
- add repositories or migrations;
- add chi routes, request DTOs, WebSocket code, or frontend code;
- add a simulation goroutine or mutex to domain entities;
- implement `LabManager` ahead of Stage 11.

---

# 6. Architecture choice

Use small domain command functions in a new focused file, for example:

```text
internal/domain/observer_commands.go
```

They receive the current Portal, destination Plane, Observer slice, `now`,
config, confirmation flag and Random dependency explicitly. They validate all
restrictions before mutation, select the affected Observer, and then reuse
Stage 3 lifecycle methods.

Suggested shape:

```go
func SendObserver(
    portal *Portal,
    plane *Plane,
    observers []Observer,
    now time.Time,
    confirmUnstable bool,
    rnd random.Random,
    cfg config.Config,
) (observerID int64, err error)

func RecallObserver(
    portal *Portal,
    plane *Plane,
    observers []Observer,
    now time.Time,
    confirmUnstable bool,
    rnd random.Random,
    cfg config.Config,
) (observerID int64, err error)

func ClosePortalWithObservers(
    portal *Portal,
    plane *Plane,
    observers []Observer,
    now time.Time,
    confirm bool,
    cfg config.Config,
) error
```

Exact Go names may change if first RED tests reveal a clearer API. The
responsibility split may not change:

```text
Stage 4 command = admission + selection + cross-entity atomic mutation
Stage 3 method  = lifecycle state transition after admission
Stage 2 method  = Portal primitive/formulas
```

---

# 7. Alternatives considered

## A. Pure domain command functions — selected

Advantages:

- directly testable without manager, database, or locks;
- reuses Stage 2/3 behavior;
- gives future LabManager one clear command boundary;
- keeps Stage 4 within its roadmap scope.

## B. Implement commands directly on `LabManager` — rejected for Stage 4

This would prematurely pull in shared active state, resolve-first locking,
concurrency and possibly persistence/event concerns assigned to Stage 11.

## C. Put Observer collections inside `Portal` — rejected

This conflicts with the established aggregate model. Portal must not own
Observer pointers or search global Observer state; Stage 3 explicitly preserved
this boundary.

---

# 8. Aggregate precondition

Stage 4 functions operate on a state already resolved to `now` by the caller:

```text
Portal.ResolveLifecycle(now) has already run.
Relevant Observer lifecycle phases due at/before now have already run.
```

Today tests construct such state directly. Stage 11 LabManager will enforce the
precondition under its mutex before invoking these commands.

Stage 4 MUST NOT move the roadmap's resolve-first policy into Portal or
Observer primitives. It also must not emit future lifecycle Events while
resolving. A clearly stale transit (`PhaseEndsAt <= now` while still marked
OUTBOUND/RETURNING) is an invariant error at this command boundary, not a reason
to guess scheduler order.

---

# 9. Structural validation

Before business admission, command functions validate the supplied aggregate:

```text
portal != nil
plane != nil
plane.ID == portal.DestinationPlaneID
Observer slice contains no duplicate Observer IDs
transit states have canonical Stage 3 fields
WAITING_RETURN candidates have CurrentPlaneID and PhaseStartedAt
```

Impossible/corrupted state returns `ErrObserverInvariant` (or a narrowly named
Stage 4 invariant error) with no mutations and no random draw.

Do not build a reflection-based generic validator. Small explicit helpers are
preferred.

---

# 10. Deterministic selection decisions

Final Spec fixes which status is eligible, but does not specify every tie.
Stage 4 records the following **technical canonicalization decisions**, not new
gameplay rules:

## 10.1 SEND selection

When multiple AVAILABLE Observers exist:

```text
choose the AVAILABLE Observer with the lowest ID
```

Observers currently have no differing capabilities, so this only makes state,
tests and future persistence deterministic.

## 10.2 RECALL selection

Longest waiting means the earliest non-nil:

```text
PhaseStartedAt
```

among WAITING_RETURN Observers whose `CurrentPlaneID` equals
`portal.DestinationPlaneID`.

Exact waiting-time tie:

```text
choose the lowest Observer ID
```

The tie-breaker is deterministic and does not alter the Final Spec priority.
If product semantics later choose another tie-breaker, update Final Spec first.

---

# 11. Suggested selection/busy helpers

Small helpers may be exported if future LabManager will reuse them:

```go
func AvailableObserverIndex(observers []Observer) (index int, ok bool)

func LongestWaitingObserverIndex(
    observers []Observer,
    planeID int64,
) (index int, ok bool, err error)

func ActiveTransitObserverIndex(
    observers []Observer,
    portalID int64,
    now time.Time,
) (index int, ok bool, err error)
```

Returning an index avoids pointers to range-loop copies and permits mutation of
the actual slice element.

`ActiveTransitObserverIndex` considers only canonical OUTBOUND/RETURNING states
whose `ActivePortalID` matches the Portal. More than one match means the
FLOW-006 invariant has already been violated and returns an invariant error.

---

# 12. Stable domain errors

Add only errors owned by Stage 4 admission:

```go
var (
    ErrPortalCriticalRisk      = errors.New("portal risk is critical")
    ErrPortalDirectionConflict = errors.New("portal observer direction conflict")
    ErrPortalBusy              = errors.New("portal already has an observer in transit")
    ErrPortalCreaturesPresent  = errors.New("creatures are still inside portal")
    ErrNoAvailableObserver     = errors.New("no available observer")
    ErrNoWaitingObserver       = errors.New("no observer waiting in destination plane")
)
```

Reuse existing errors when semantically exact:

```text
ErrPortalNotOpen
ErrConfirmationRequired
ErrObserverInvariant
ErrObserverNotAvailable
ErrObserverNotWaitingReturn
```

Do not put HTTP status codes or response bodies in `internal/domain`.

---

# 13. Business-validation order

After structural validation, use this deterministic order for both SEND and
RECALL:

```text
1. Portal OPEN
2. current Risk is not CRITICAL
3. Portal direction permits action
4. creatures_inside == 0
5. Portal has no active transit
6. eligible Observer exists
7. UNSTABLE confirmation is present
8. perform mutation
```

Rationale:

- hard rejection is returned before an optional warning;
- no confirmation modal is requested for an action that cannot succeed;
- random duration is drawn only after every admission check succeeds;
- tests can lock a stable error precedence instead of depending on incidental
  `if` ordering.

This ordering is an internal API contract. It does not weaken any Final Spec
restriction.

---

# 14. SEND admission

SEND is allowed only when all are true:

```text
Portal.Status == OPEN
Portal.RiskLevel(now, cfg) != CRITICAL
Portal.ObserverFlow != INBOUND
Portal.CreaturesInside(now, cfg) == 0
no OUTBOUND/RETURNING Observer has ActivePortalID == Portal.ID
at least one Observer is AVAILABLE
Portal is STABLE OR confirmUnstable == true
```

SEND intentionally does **not** reject because:

```text
Plane.Explored == true
another Observer is EXPLORING in the destination Plane
another Observer is WAITING_RETURN in the destination Plane
Risk is LOW / MEDIUM / HIGH
Portal Energy is low but Risk is not CRITICAL
scheduled remaining time is low but Risk is not CRITICAL
```

No warning is generated for those conditions.

---

# 15. Successful SEND mutation

After all checks:

```text
selected Observer.StartOutbound(now, portal.ID, rnd, cfg)

if Portal.ObserverFlow == NONE:
    Portal.ObserverFlow = OUTBOUND
    Portal.UpdatedAt = now

if Portal.ObserverFlow == OUTBOUND:
    keep OUTBOUND
```

Return the selected Observer ID for later Event/API layers.

Important ordering:

1. validate everything;
2. start Observer transit;
3. fix flow if it was NONE.

After prevalidation, `StartOutbound` should not return an ordinary domain
error. If it does, flow must remain unchanged. Do not consume a second random
value and do not retry silently.

Starting SEND must leave Plane exploration unchanged.

---

# 16. SEND UNSTABLE warning

For an otherwise valid UNSTABLE Portal:

```text
confirmUnstable == false
→ ErrConfirmationRequired
→ Portal unchanged
→ all Observers unchanged
→ Plane unchanged
→ FakeRandom queue unchanged
```

Retrying the same command with:

```text
confirmUnstable == true
```

performs the normal successful SEND.

The eventual transport/UI message is the Final Spec text:

```text
This Omenpath is unstable.
It may collapse unexpectedly during transit.
```

Stage 4 owns the confirmation boundary, not the HTTP/modal representation.

---

# 17. RECALL admission

RECALL is allowed only when all are true:

```text
Portal.Status == OPEN
Portal.RiskLevel(now, cfg) != CRITICAL
Portal.ObserverFlow != OUTBOUND
Portal.CreaturesInside(now, cfg) == 0
no OUTBOUND/RETURNING Observer has ActivePortalID == Portal.ID
at least one WAITING_RETURN Observer has
    CurrentPlaneID == Portal.DestinationPlaneID
Portal is STABLE OR confirmUnstable == true
```

Observers waiting in other Planes do not make this Portal recall-eligible.

---

# 18. Successful RECALL mutation

Choose the longest-waiting eligible Observer using §10.2, then:

```text
selected Observer.StartReturning(now, portal.ID, rnd, cfg)

if Portal.ObserverFlow == NONE:
    Portal.ObserverFlow = INBOUND
    Portal.UpdatedAt = now

if Portal.ObserverFlow == INBOUND:
    keep INBOUND
```

Return the selected Observer ID.

Only the selected Observer changes. Other WAITING_RETURN Observers stay in the
Plane. This is also the behavior later reused after Extraction's first
automatic return.

---

# 19. RECALL UNSTABLE warning

The semantics mirror SEND:

```text
otherwise valid + UNSTABLE + confirmUnstable=false
→ ErrConfirmationRequired
→ no mutations and no random draw

same command + confirmUnstable=true
→ successful RETURNING start
```

Do not warn merely because Risk is HIGH, Energy is low, or remaining time is
short. CRITICAL is a hard rejection, not a confirmable warning.

---

# 20. Permanent Portal flow

Flow changes only on the first successful transit start:

```text
NONE + successful SEND   → OUTBOUND forever
NONE + successful RECALL → INBOUND forever
```

It does not reset when:

- the transit succeeds;
- the transit fails and Observer becomes LOST;
- the Portal becomes idle;
- research completes;
- another Portal to the same Plane opens.

A rejected or confirmation-required action never changes flow.

Stage 4 must not add any method that resets `ObserverFlow` to NONE.

---

# 21. One transit per Portal

Portal is busy when one Observer is semantically in an active OUTBOUND or
RETURNING transit through that exact `Portal.ID`.

Rules:

```text
busy Portal rejects SEND
busy Portal rejects RECALL
another Portal to the same Plane remains independent
after transit resolves, same-direction next transit is allowed
```

Global restrictions are forbidden:

```text
only one transit in the whole laboratory        # wrong
only one Observer per Plane                     # wrong
one busy Portal blocks another Portal           # wrong
```

If two active transits already reference the same Portal, return invariant
error. Do not choose one arbitrarily.

---

# 22. Manual Close with active transit

Stage 2 `Portal.Close` already owns Portal state mutation and creature
confirmation. Stage 4 adds Observer-aware orchestration without moving loss
logic into Portal.

For an active transit whose `PhaseEndsAt > now`:

```text
confirm == false
→ ErrConfirmationRequired
→ Portal and all Observers unchanged

confirm == true
→ call Portal.Close(now, true, cfg)
→ call ResolveObserverLifecycle for the active Observer using the now-closed
  Portal
→ Observer becomes LOST at now
```

The same single confirmation acknowledges both present creatures and active
transit when both conditions exist. Transport may later render combined copy;
Stage 4 only guarantees one retry with `confirm=true` is sufficient.

If no Observer is active, delegate to the existing Portal primitive so its
creature confirmation behavior remains canonical.

---

# 23. Manual Close atomicity

Before calling `Portal.Close`, prevalidate everything required to resolve the
affected Observer loss:

```text
at most one active transit
active Observer has canonical Stage 3 transit fields
plane matches Portal destination
PhaseEndsAt > now
```

This prevents the Portal from closing and then discovering that the Observer
cannot be transitioned.

After a successful close with active transit, assert both outcomes together:

```text
Portal.Status == CLOSED
Portal.TerminationReason == MANUAL_CLOSE
Portal.ClosedAt == now
Observer.Status == LOST
Observer.UpdatedAt == now
Observer current Plane/Portal/phase fields == nil
```

Lab Energy cost `5` is still deferred to Stage 5.

---

# 24. Exact transit-end boundary for manual Close

Final Spec and Stage 3 define:

```text
Portal closes before transit end → LOST
Portal closes exactly at transit end → transit succeeds
```

Stage 4 command state must already be resolved to `now`. Therefore an Observer
still marked OUTBOUND/RETURNING with `PhaseEndsAt <= now` is stale input and
must produce `ErrObserverInvariant` before close mutation.

Future LabManager behavior:

```text
resolve Portal/Observer state under lock
→ if transit completed exactly now, Observer is no longer active
→ Close proceeds without active-transit warning/loss
```

Do not override the Stage 3 exact-tie rule inside the close command.

---

# 25. Mutation safety

Every rejected command must leave byte-for-byte-equivalent snapshots of:

```text
Portal
Plane
all Observers
```

and must not consume FakeRandom.

Mandatory rejection snapshots:

- terminal Portal;
- CRITICAL Risk;
- wrong flow;
- creatures present;
- busy Portal;
- no AVAILABLE Observer;
- no WAITING_RETURN Observer at destination;
- UNSTABLE without confirmation;
- malformed aggregate;
- manual Close without required confirmation.

Do not "reserve" flow or mutate Observer before the last admission check.

---

# 26. Error precedence tests

Lock the §13 order with compound-invalid fixtures.

Examples:

```text
terminal + critical-like derived state
→ ErrPortalNotOpen

CRITICAL + wrong flow
→ ErrPortalCriticalRisk

wrong flow + creatures
→ ErrPortalDirectionConflict

creatures + busy
→ ErrPortalCreaturesPresent

busy + no eligible Observer
→ ErrPortalBusy

no eligible Observer + UNSTABLE
→ ErrNoAvailableObserver / ErrNoWaitingObserver
  (not confirmation)
```

This is deterministic error reporting; it does not make later restrictions
less important.

---

# 27. Interaction with current Risk

Use only:

```go
level, ok := portal.RiskLevel(now, cfg)
```

Rules:

- `RiskCritical` rejects;
- LOW/MEDIUM/HIGH do not reject and do not warn;
- do not inspect numeric `RiskScore` for a separate threshold;
- do not include hidden instability timestamp in admission;
- terminal Portal rejection happens before relying on Risk's `ok=false`.

Do not duplicate the Risk formula in observer command code.

---

# 28. Interaction with creatures

Use only:

```go
portal.CreaturesInside(now, cfg)
```

Rules:

- any value `>0` rejects SEND/RECALL;
- exactly `0` permits continued validation;
- do not read `CreaturesInitial` as current state;
- do not change Stage 2's two-second passage or terminal freeze semantics.

Manual Close remains confirmable with creatures through `Portal.Close`.

---

# 29. Interaction with Plane exploration

SEND receives the destination Plane for aggregate validation but must not mutate:

```text
Plane.Explored
Plane.ExploredAt
```

An already explored Plane is valid. Another Observer already in that Plane is
also valid.

RECALL selection uses only Plane identity and Observer WAITING_RETURN state.
It does not mark exploration; successful return later does that through the
Stage 3 resolver.

---

# 30. No Event implementation

Stage 4 actions will eventually correspond to:

```text
OBSERVER_DISPATCHED
OBSERVER_RETURN_STARTED
OBSERVER_LOST
PORTAL_CLOSED
ACTION_REJECTED
```

Do not create or persist those Events here. Stage 9/11 orchestration will use
the returned Observer ID and state diffs to emit each event exactly once.

---

# 31. Race/concurrency boundary

Stage 4 functions are deterministic mutations of caller-owned in-memory
objects. They do not add mutexes to Portal, Plane or Observer.

Stage 11 will call them under `LabManager`'s lock so concurrent REST and
simulation transitions have one winner.

Still run the race suite at every checkpoint to detect accidental shared
globals and regressions in existing utilities.

---

# 32. Suggested production files

Prefer:

```text
internal/domain/observer_commands.go   # selection, admission, SEND/RECALL
internal/domain/portal_observer_close.go
internal/domain/errors.go              # stable Stage 4 errors
```

Suggested tests:

```text
internal/domain/observer_selection_test.go
internal/domain/observer_send_test.go
internal/domain/observer_recall_test.go
internal/domain/observer_flow_test.go
internal/domain/portal_observer_close_test.go
```

Modify existing Stage 2/3 production files only if a demonstrated RED test
requires a small reusable primitive. Do not refactor their semantics.

---

# 33. Test fixtures

Reuse existing deterministic fixtures where clear:

```text
BaseTime             = testutil.BaseTime
Portal ID            = 11
Destination Plane ID = 7
AVAILABLE IDs         = 1, 2, ...
Transit duration     = FakeRandom 10 sec
```

Add focused test helpers for:

- stable non-critical Portal with zero creatures;
- CRITICAL Portal using real Stage 2 Risk calculation;
- OUTBOUND/INBOUND flow variants;
- AVAILABLE roster in deliberately unsorted ID order;
- WAITING_RETURN Observers with explicit waiting timestamps;
- active OUTBOUND/RETURNING fixtures with explicit `PhaseEndsAt`.

No fixture should invoke the production command under test merely to construct
its initial state.

---

# 34. TDD checkpoint A — selection and busy helpers

Write tests first.

Required tests:

```text
TestAvailableObserverIndex_SelectsLowestAvailableID
TestAvailableObserverIndex_IgnoresNonAvailableObservers
TestAvailableObserverIndex_ReturnsNoneWhenUnavailable

TestLongestWaitingObserverIndex_SelectsEarliestWaitingTimestamp
TestLongestWaitingObserverIndex_FiltersByDestinationPlane
TestLongestWaitingObserverIndex_BreaksExactTieByLowestID
TestLongestWaitingObserverIndex_RejectsMissingWaitingTimestamp

TestActiveTransitObserverIndex_FindsOutbound
TestActiveTransitObserverIndex_FindsReturning
TestActiveTransitObserverIndex_IgnoresOtherPortals
TestActiveTransitObserverIndex_ReturnsNoneWhenIdle
TestActiveTransitObserverIndex_RejectsMultipleTransitsForSamePortal
TestActiveTransitObserverIndex_RejectsStaleTransitAtDeadline
```

RED should show missing helpers.

Minimal GREEN:

- index-based deterministic selectors;
- canonical field validation required by selection;
- no command mutation yet.

Traceability preparation:

```text
OBSERVER-016  PARTIAL until RECALL command consumes selector
FLOW-006      PARTIAL until SEND/RECALL enforce busy result
```

---

# 35. TDD checkpoint B — successful SEND and first flow

Required tests:

```text
TestSendObserver_SelectsLowestAvailableObserver
TestSendObserver_StartsOutboundTransit
TestSendObserver_ReturnsSelectedObserverID
TestSendObserver_FirstUseSetsOutboundFlow
TestSendObserver_ExistingOutboundFlowRemainsOutbound
TestSendObserver_UpdatesPortalOnlyWhenFlowFirstChanges
TestSendObserver_DrawsTransitDurationExactlyOnce
TestSendObserver_DoesNotExplorePlane
TestSendObserver_AllowsAlreadyExploredPlane
TestSendObserver_AllowsAnotherObserverInDestinationPlane
```

RED must demonstrate missing command behavior.

Minimal GREEN:

- structural validation;
- selection of lowest-ID AVAILABLE Observer;
- delegation to `StartOutbound`;
- NONE→OUTBOUND mutation only after success;
- returned Observer ID.

Traceability target:

```text
FLOW-002
PLANE-005     GREEN after real command proof
OBSERVER-015
```

---

# 36. TDD checkpoint C — SEND restrictions and warning

Required tests:

```text
TestSendObserver_RejectsClosedPortal
TestSendObserver_RejectsCollapsedPortal
TestSendObserver_RejectsCriticalRisk
TestSendObserver_RejectsInboundFlow
TestSendObserver_RejectsCreaturesInside
TestSendObserver_RejectsBusyPortal
TestSendObserver_RejectsWhenNoObserverAvailable

TestSendObserver_UnstableRequiresConfirmation
TestSendObserver_UnstableConfirmedStartsTransit
TestSendObserver_HighRiskStablePortalNeedsNoConfirmation
TestSendObserver_LowEnergyDoesNotCreateSeparateWarning
TestSendObserver_LowRemainingTimeDoesNotCreateSeparateWarning
TestSendObserver_ExploredPlaneDoesNotCreateSeparateWarning

TestSendObserver_RejectionIsAtomic
TestSendObserver_RejectionDoesNotConsumeRandom
TestSendObserver_UsesDocumentedErrorPrecedence
```

Traceability target:

```text
RISK-012
CREATURE-007  PARTIAL until RECALL is also covered
FLOW-005
FLOW-006      PARTIAL until both commands covered
```

---

# 37. TDD checkpoint D — successful RECALL and longest waiting

Required tests:

```text
TestRecallObserver_SelectsLongestWaitingInDestination
TestRecallObserver_BreaksWaitingTieByLowestID
TestRecallObserver_IgnoresWaitingObserversInOtherPlanes
TestRecallObserver_StartsReturningTransit
TestRecallObserver_PreservesCurrentPlaneDuringTransit
TestRecallObserver_ReturnsSelectedObserverID
TestRecallObserver_FirstUseSetsInboundFlow
TestRecallObserver_ExistingInboundFlowRemainsInbound
TestRecallObserver_DrawsFreshTransitDurationExactlyOnce
TestRecallObserver_LeavesOtherWaitingObserversUnchanged
```

Minimal GREEN:

- reuse checkpoint A selector;
- delegate to `StartReturning`;
- NONE→INBOUND only after success;
- only selected Observer mutates.

Traceability target:

```text
FLOW-003
OBSERVER-016  GREEN
```

---

# 38. TDD checkpoint E — RECALL restrictions and warning

Required tests:

```text
TestRecallObserver_RejectsClosedPortal
TestRecallObserver_RejectsCollapsedPortal
TestRecallObserver_RejectsCriticalRisk
TestRecallObserver_RejectsOutboundFlow
TestRecallObserver_RejectsCreaturesInside
TestRecallObserver_RejectsBusyPortal
TestRecallObserver_RejectsWhenDestinationHasNoWaitingObserver
TestRecallObserver_DoesNotUseWaitingObserverFromOtherPlane

TestRecallObserver_UnstableRequiresConfirmation
TestRecallObserver_UnstableConfirmedStartsTransit
TestRecallObserver_HighRiskStablePortalNeedsNoConfirmation
TestRecallObserver_LowEnergyDoesNotCreateSeparateWarning
TestRecallObserver_LowRemainingTimeDoesNotCreateSeparateWarning

TestRecallObserver_RejectionIsAtomic
TestRecallObserver_RejectionDoesNotConsumeRandom
TestRecallObserver_UsesDocumentedErrorPrecedence
```

Traceability target:

```text
RISK-013
CREATURE-007  GREEN
FLOW-004
FLOW-006      GREEN for command admission
```

---

# 39. TDD checkpoint F — permanent flow and next transit

Required tests:

```text
TestNaturalPortal_ObserverFlowStartsNone
TestPortalFlow_DoesNotChangeOnRejectedFirstSend
TestPortalFlow_DoesNotChangeOnRejectedFirstRecall
TestPortalFlow_DoesNotResetAfterSuccessfulOutbound
TestPortalFlow_DoesNotResetAfterSuccessfulReturn
TestPortalFlow_DoesNotResetAfterObserverLost

TestPortalFlow_AllowsNextOutboundAfterPreviousTransitEnds
TestPortalFlow_AllowsNextInboundAfterPreviousTransitEnds
TestPortalFlow_RejectsRecallAfterOutboundTransitEnds
TestPortalFlow_RejectsSendAfterInboundTransitEnds
TestPortalBusy_IsScopedToPortalID
TestPortalBusy_DoesNotBlockAnotherPortalToSamePlane
```

`TestNaturalPortal_ObserverFlowStartsNone` may already be GREEN because the
Stage 2 factory sets NONE. Record it explicitly as a characterization; do not
fake RED.

Traceability target:

```text
FLOW-001..007 GREEN
```

---

# 40. TDD checkpoint G — manual Close with active transit

Required tests:

```text
TestClosePortalWithObservers_ActiveOutboundRequiresConfirmation
TestClosePortalWithObservers_ActiveReturningRequiresConfirmation
TestClosePortalWithObservers_ConfirmedOutboundBecomesLost
TestClosePortalWithObservers_ConfirmedReturningBecomesLost
TestClosePortalWithObservers_ConfirmedCloseUsesNowForBothTransitions
TestClosePortalWithObservers_LossClearsObserverCurrentState
TestClosePortalWithObservers_LossDoesNotExplorePlane
TestClosePortalWithObservers_CreaturesAndTransitUseSingleConfirmation
TestClosePortalWithObservers_NoTransitDelegatesToPortalClose
TestClosePortalWithObservers_RejectsTerminalPortal
TestClosePortalWithObservers_RejectsMultipleActiveTransits
TestClosePortalWithObservers_RejectsStaleTransitBeforeMutation
TestClosePortalWithObservers_RejectionIsAtomic
```

Minimal GREEN:

- preflight active transit and Plane invariants;
- reuse `Portal.Close`;
- reuse `ResolveObserverLifecycle` for LOST;
- no Lab Energy accounting.

Traceability target:

```text
PORTAL-006    add Observer-confirmation tests/implementation note
OBSERVER-012  add confirmed manual-close coverage
FLOW-006      invariant defense for corrupted multiple-active state
```

---

# 41. TDD checkpoint H — atomicity and integration regressions

Required tests:

```text
TestObserverCommands_DoNotMutateOnStructuralInvariantError
TestObserverCommands_DoNotChangeUnselectedObservers
TestObserverCommands_DoNotChangePlaneExplorationOnTransitStart
TestObserverCommands_DoNotConsumeRandomBeforeAllChecksPass
TestObserverCommands_DifferentPortalsOperateIndependently
TestObserverCommands_MultipleObserversMayRemainInSamePlane
TestObserverCommands_Stage3TransitDeadlineRemainsFixed
TestObserverCommands_DoNotResetFlowDuringLifecycleCatchUp
```

These may be characterization/regression tests over already-GREEN behavior.
Document any test that passes immediately; do not manufacture a RED failure.

No new behavior beyond Final Spec §§15–19/21 may be introduced here.

---

# 42. TDD execution protocol

For each checkpoint:

```text
1. Map requirement IDs and Final Spec sections.
2. Write focused tests.
3. Run targeted tests and record genuine RED.
4. Commit RED tests.
5. Implement the minimum behavior.
6. Run targeted tests → GREEN.
7. Run full test suite.
8. Run race suite.
9. Refactor only while GREEN.
10. Update docs/traceability.md.
11. Commit GREEN implementation.
```

Do not fake RED by breaking Stage 2/3 production code. An already-passing
characterization test is allowed only when called out in commit/worklog.

Recommended history:

```text
test(stage4): RED checkpoint A selection and busy helpers
feat(stage4): GREEN checkpoint A selection and busy helpers
...
test(stage4): RED checkpoint G manual close transit confirmation
feat(stage4): GREEN checkpoint G manual close transit confirmation
test(stage4): GREEN checkpoint H integration characterizations
docs(stage4): worklog and verification
```

---

# 43. Required verification commands

At every meaningful GREEN checkpoint and at final Stage 4 completion:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
go test -race -count=1 ./...
```

`gofmt -l .` must print no project Go files.

Additional scope checks:

```bash
grep -Rni 'time.Sleep\|time.Now()\|time.NewTicker\|time.After' internal/domain
grep -Rni 'LabEnergy\|Leyline\|Extraction\|database/sql\|net/http' internal/domain/observer_commands*.go internal/domain/portal_observer_close*.go
```

Expected Stage 4 command implementation:

- no wall-clock calls;
- no Stage 5+ dependencies;
- no mutexes in domain entities;
- no real random in tests.

---

# 44. Traceability update expected after Stage 4

Expected final statuses:

```text
FLOW-001..007   GREEN
CREATURE-007   GREEN
RISK-012..013  GREEN
OBSERVER-015   GREEN
OBSERVER-016   GREEN
PLANE-005      GREEN

OBSERVER-001   remains PARTIAL  # persistence/bootstrap later
PLANE-009      remains PARTIAL  # progress aggregation later
LAB-*          remains PLANNED  # Stage 5
EXTRACTION-*   remains PLANNED  # Stage 7
EVENT-*        remains PLANNED  # Stage 9
```

Enhance existing `PORTAL-006` and `OBSERVER-012` rows with manual-close active
transit tests without changing already-correct Stage 2/3 semantics.

Every GREEN row must list concrete test names and implementation symbols.

`docs/requirements.md` remains semantically unchanged unless implementation
reveals a genuine missing or contradictory requirement. A catalog-only wording
clarification must still cite the exact Final Spec section.

---

# 45. Worklog update expected

Append to `01_AI_WORKLOG_CURRENT.md`:

- Stage 4 execution agent/model;
- RED/GREEN commit pairs;
- actual selection/tie behavior used;
- warning confirmation behavior;
- validation/error precedence;
- any atomicity bug found by tests;
- any accidental Stage 5+ scope leak caught and removed;
- final verification results;
- explicit statement that Stage 5 was not started.

Do not invent AI mistakes or claim commands passed before running them.

---

# 46. Stage 4 acceptance checklist

Stage 4 is complete only if all are true:

- [ ] Natural Portal flow NONE is covered.
- [ ] First successful SEND sets OUTBOUND.
- [ ] First successful RECALL sets INBOUND.
- [ ] Flow never resets after transit completion/loss.
- [ ] INBOUND rejects SEND.
- [ ] OUTBOUND rejects RECALL.
- [ ] One active transit per Portal is enforced.
- [ ] Busy state is scoped by Portal ID, not Plane/global state.
- [ ] Next same-direction transit is allowed after the previous transit ends.
- [ ] SEND selects the lowest-ID AVAILABLE Observer deterministically.
- [ ] SEND rejects when no Observer is AVAILABLE.
- [ ] SEND to explored Plane succeeds without an exploration warning.
- [ ] SEND allows other Observers in the same Plane.
- [ ] RECALL filters WAITING_RETURN Observers by destination Plane.
- [ ] RECALL selects earliest waiting timestamp.
- [ ] Equal waiting timestamps use the documented lowest-ID canonicalization.
- [ ] CRITICAL rejects SEND and RECALL.
- [ ] LOW/MEDIUM/HIGH do not independently reject or warn.
- [ ] Creatures block SEND and RECALL.
- [ ] UNSTABLE otherwise-valid SEND requires confirmation.
- [ ] UNSTABLE otherwise-valid RECALL requires confirmation.
- [ ] Confirmed UNSTABLE action starts exactly one transit.
- [ ] Rejected/confirmation-required action consumes no randomness.
- [ ] Rejected action leaves Portal, Plane and all Observers unchanged.
- [ ] Successful first action changes flow only after Observer start succeeds.
- [ ] Manual Close with active transit requires confirmation.
- [ ] Confirmed active-transit Close produces CLOSED/MANUAL_CLOSE and LOST.
- [ ] One confirmation covers simultaneous creatures + active transit.
- [ ] Manual Close active-transit loss reuses Stage 3 strict-before semantics.
- [ ] Stale transit-at-deadline input is rejected as an invariant violation.
- [ ] No Lab Energy behavior was implemented.
- [ ] No Leyline Override behavior was implemented.
- [ ] No Extraction behavior was implemented.
- [ ] No Events/persistence/API/WebSocket/simulation/LabManager was implemented.
- [ ] No Stage 2 Portal or Stage 3 Observer semantics were changed without a RED regression.
- [ ] `gofmt -l .` clean.
- [ ] `go vet ./...` passes.
- [ ] `go build ./...` passes.
- [ ] `go test -count=1 ./...` passes.
- [ ] `go test -race -count=1 ./...` passes.
- [ ] Traceability updated honestly.
- [ ] Worklog updated honestly.
- [ ] Stage 5 not started.

---

# 47. Explicit non-goals

Do NOT implement in Stage 4:

```text
Laboratory Energy state or deductions
Close/Stabilize cost orchestration
Leyline Override
Extraction Portal creation or sync
automatic Extraction return
Natural Portal generator
simulation tick/goroutine
Needs Attention
Event emission/persistence
SQLite
REST/HTTP confirmations
WebSocket snapshots
Tutorial behavior
frontend warnings/modals
LabManager mutex/concurrency orchestration
```

Tests requiring those behaviors belong to later stages.

---

# 48. Resolved Stage 4 planning decisions

No blocking Final Spec contradiction is known.

Resolved for deterministic implementation:

## S4-D1 — command layer

Pure domain command functions wrap Stage 2/3 primitives. LabManager remains
deferred to Stage 11.

## S4-D2 — SEND candidate tie

Choose lowest-ID AVAILABLE Observer. This is explicitly technical
canonicalization because Final Spec does not distinguish Observer abilities.

## S4-D3 — RECALL exact waiting tie

Earliest `PhaseStartedAt` wins; exact timestamp tie uses lowest Observer ID.

## S4-D4 — warning order

All hard restrictions and candidate availability are checked before UNSTABLE
confirmation. Impossible actions do not ask for confirmation.

## S4-D5 — confirmation mutation

Confirmation-required results are fully atomic and consume no random draw.

## S4-D6 — manual Close boundary

Stage 4 accepts only Observer state already resolved to `now`; stale transit at
or past its deadline is an invariant error. Stage 3 exact-tie semantics remain
unchanged.

Stage 4 can proceed through TDD without starting Stage 5.

---

# 49. Stop condition

After every Stage 4 acceptance item is verified:

```text
STOP.
```

Do not begin Stage 5 in the same implementation pass.

Final Stage 4 report must include:

- files changed;
- tests added;
- RED/GREEN commit evidence;
- characterization tests that were already GREEN;
- selection/tie decisions used;
- verification command results;
- traceability changes;
- commit hashes;
- confirmation that Stage 5 was not started.

---

# 50. Next stage after completion

Stage 5:

```text
Laboratory Energy via TDD

integer 0..100
derived +1/sec regeneration
Close 5
Stabilize 20
SEND / RECALL 0
Extraction 30
insufficient-energy rejection
```

Stage 5 must wrap the Stage 4 commands without changing their Portal/Observer
admission and flow semantics.
