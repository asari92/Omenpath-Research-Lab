# Omenpath Research Lab — Stage 3
## Observer Lifecycle via TDD

> Depends on:
> - `00_FINAL_SPEC_v5.md`
> - `03_STAGE_00_01_TDD_FOUNDATION.md`
> - `04_STAGE_02_PORTAL_CORE_TDD.md`
>
> This stage implements the Observer state machine, deterministic transit/research timing,
> transit failure, and Plane exploration timing.
>
> It does **not** implement Portal observer direction/admission rules (`SEND` / `RECALL`),
> Laboratory Energy, Extraction, persistence, HTTP/WebSocket transport, event persistence,
> or `LabManager` concurrency orchestration.

---

# 1. Goal

Implement the complete **Observer lifecycle core** as deterministic Go domain logic using TDD:

```text
AVAILABLE
→ OUTBOUND
→ EXPLORING
→ WAITING_RETURN
→ RETURNING
→ AVAILABLE

OUTBOUND / RETURNING
→ LOST    # only when the active Portal becomes terminal before transit ends
```

Also implement the exploration rule:

```text
SEND / arrival / research completion
DO NOT explore the Plane.

Only a successful RETURNING → AVAILABLE transition
for an Observer that completed research explores the Plane.
```

At the end of Stage 3 all Observer timing must be testable entirely in memory with:

- `FakeClock`;
- `FakeRandom`;
- explicit `Portal.ClosedAt` fixtures;
- deterministic `Plane` fixtures;
- no `time.Sleep`;
- no database;
- no HTTP;
- no goroutine-driven simulation loop.

---

# 2. Requirements covered

Primary Stage 3 requirements:

```text
OBSERVER-001  Exactly 10 permanent Observers                       # expected PARTIAL: roster/count=10 proven; permanence/persistence bootstrap — later stages
OBSERVER-002  Initial status AVAILABLE
OBSERVER-003  AVAILABLE = Laboratory
OBSERVER-004  Main lifecycle
OBSERVER-005  LOST terminal
OBSERVER-006  Transit random 5..15 sec
OBSERVER-007  Transit duration generated once
OBSERVER-008  Successful OUTBOUND → EXPLORING
OBSERVER-009  Research duration 20 sec
OBSERVER-010  Research completion → WAITING_RETURN
OBSERVER-011  Successful RETURNING → AVAILABLE
OBSERVER-012  Portal CLOSED during transit → LOST
OBSERVER-013  Portal COLLAPSED during transit → LOST
OBSERVER-014  Multiple Observers may occupy same Plane

PLANE-005     SEND does not explore                                # expected PARTIAL: StartOutbound primitive only; real SEND command — Stage 4
PLANE-006     OUTBOUND arrival does not explore
PLANE-007     Research completion does not explore
PLANE-008     Successful return after research explores
PLANE-009     Re-return to already explored Plane is idempotent    # expected PARTIAL: Plane state idempotence proven; progress aggregation — later
```

Requirements intentionally deferred to Stage 4 or later:

```text
OBSERVER-015  SEND to already EXPLORED Plane allowed without warning  # Stage 4 command rules
OBSERVER-016  RECALL chooses longest-waiting Observer                 # Stage 4

FLOW-001..007 Portal observer flow and one-transit-at-a-time rules     # Stage 4
CREATURE-007  Creatures block SEND/RECALL                              # Stage 4
RISK-*        CRITICAL blocks SEND/RECALL                              # Stage 4
LAB-*         command energy accounting                                # Stage 5
EXTRACTION-*  extraction return flow                                   # Stage 7
EVENT-*       durable domain Event emission                            # Stage 9
PERSIST-*     Observer/Plane persistence                               # Stage 10
API-*         HTTP commands                                            # later transport stage
```

Stage 3 may prepare APIs that Stage 4 calls, but MUST NOT implement Stage 4 admission policy.

---

# 3. Hard stage boundary

Stage 3 answers:

```text
"Given that a transit has already been authorized and started,
 what happens to this Observer as time advances?"
```

Stage 4 answers:

```text
"May SEND or RECALL start through this Portal right now,
 and which Observer is selected?"
```

Therefore Stage 3 MUST NOT add command restrictions such as:

```text
Portal OPEN
Risk != CRITICAL
observer_flow != opposite direction
creatures_inside == 0
Portal not busy
AVAILABLE Observer exists
WAITING_RETURN Observer exists
longest-waiting selection
UNSTABLE warning
```

Those are Stage 4 orchestration/admission concerns.

Likewise, Stage 3 MUST NOT mutate:

```go
Portal.ObserverFlow
```

That field is owned by Stage 4 behavior.

---

# 4. Existing domain types to preserve

The repository already defines the Stage 3 state vocabulary in `internal/domain/observer.go`:

```go
type ObserverStatus string

const (
    ObserverAvailable     ObserverStatus = "AVAILABLE"
    ObserverOutbound      ObserverStatus = "OUTBOUND"
    ObserverExploring     ObserverStatus = "EXPLORING"
    ObserverWaitingReturn ObserverStatus = "WAITING_RETURN"
    ObserverReturning     ObserverStatus = "RETURNING"
    ObserverLost          ObserverStatus = "LOST"
)

type Observer struct {
    ID             int64
    Status         ObserverStatus
    CurrentPlaneID *int64
    ActivePortalID *int64
    PhaseStartedAt *time.Time
    PhaseEndsAt    *time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

And `internal/domain/plane.go` already contains:

```go
type Plane struct {
    ID          int64
    Name        string
    Aliases     []string
    CatalogTier string
    Explored    bool
    ExploredAt  *time.Time
}
```

Stage 3 should extend behavior around these types rather than inventing duplicate lifecycle models.

---

# 5. Canonical state invariants

The following representation is fixed for Stage 3.

## 5.1 AVAILABLE

```text
status            = AVAILABLE
current_plane_id  = null
active_portal_id  = null
phase_started_at  = null
phase_ends_at     = null
```

AVAILABLE means physically in the Laboratory.

## 5.2 OUTBOUND

```text
status            = OUTBOUND
current_plane_id  = null
active_portal_id  = portal ID
phase_started_at  = transit start
phase_ends_at     = transit end
```

The destination Plane is known by the Portal used for transit.

## 5.3 EXPLORING

```text
status            = EXPLORING
current_plane_id  = destination Plane ID
active_portal_id  = null
phase_started_at  = successful outbound arrival time
phase_ends_at     = phase_started_at + ResearchDuration
```

## 5.4 WAITING_RETURN

```text
status            = WAITING_RETURN
current_plane_id  = Plane ID
active_portal_id  = null
phase_started_at  = research completion time
phase_ends_at     = null
```

`phase_started_at` is deliberately retained as the beginning of the waiting period.
Stage 4 can therefore implement "longest-waiting" without inventing another timestamp.

## 5.5 RETURNING

```text
status            = RETURNING
current_plane_id  = source Plane ID
active_portal_id  = portal ID
phase_started_at  = return transit start
phase_ends_at     = transit end
```

The Observer remains associated with the Plane until the return transit succeeds.

## 5.6 LOST

```text
status            = LOST
current_plane_id  = null
active_portal_id  = null
phase_started_at  = null
phase_ends_at     = null
```

LOST is terminal.

Canonicalization note: the Final Spec does not explicitly define the
current-state field values of a LOST Observer. Clearing `CurrentPlaneID`,
`ActivePortalID`, `PhaseStartedAt` and `PhaseEndsAt` is a **Stage 3
canonicalization decision** for these fields — a lost Observer is neither
in the Lab nor attached to a live transit — and must not be presented as a
direct Final Spec requirement.

A LOST Observer is neither "in Lab" nor "in a World" for current-state counting.
Its historical Plane/Portal association belongs in the Event Log later.

---

# 6. Timestamp semantics

All state transitions use the **effective transition time**, not merely the time at which a resolver happened to run.

Example:

```text
OUTBOUND started        12:00:00
transit end             12:00:10
resolver called at      12:00:14
```

The transition is recorded as:

```text
OUTBOUND → EXPLORING at 12:00:10
phase_started_at        12:00:10
phase_ends_at           12:00:30
updated_at              12:00:10
```

not `12:00:14`.

This makes behavior independent from simulation tick frequency and is required for restart/catch-up correctness later.

---

# 7. Transit duration rule

Each OUTBOUND or RETURNING transit draws its duration exactly once when transit starts:

```text
random integer 5..15 seconds, inclusive
```

Using current config:

```go
cfg.ObserverTransitMin // 5s
cfg.ObserverTransitMax // 15s
```

The generated duration is materialized by setting:

```text
phase_ends_at = phase_started_at + generated duration
```

After that:

- lifecycle resolution MUST NOT draw random again;
- repeated reads MUST NOT draw random;
- retrying `Resolve...` MUST NOT change the deadline;
- `FakeRandom` consumption must prove one draw per actual transit start.

The random value is not stored separately because `phase_started_at` + `phase_ends_at` already persists the generated result.

---

# 8. Research timing

Successful OUTBOUND arrival starts research immediately at the arrival timestamp:

```text
OUTBOUND end at T
→ EXPLORING at T
→ phase_ends_at = T + 20 sec
```

Research duration comes from:

```go
cfg.ResearchDuration
```

At research completion:

```text
EXPLORING → WAITING_RETURN
```

The Plane remains UNEXPLORED.

No random draw is involved in research duration.

---

# 9. Plane exploration rule

Plane exploration belongs to the Plane, never to the Portal.

The following MUST NOT set `Plane.Explored = true`:

```text
starting OUTBOUND
successful OUTBOUND arrival
starting EXPLORING
research completion
entering WAITING_RETURN
starting RETURNING
Observer loss
```

Only successful return does:

```text
RETURNING → AVAILABLE
Plane.Explored = true
Plane.ExploredAt = successful return timestamp
```

If the Plane was already explored:

```text
Plane.Explored remains true
Plane.ExploredAt remains the original exploration timestamp
```

A later successful return must not overwrite the first exploration timestamp.

This is Stage 3's idempotence rule for the Plane-state half of `PLANE-009`
(`Explored`/`ExploredAt`). The "does not increase progress again" half is
proven only when progress aggregation exists (later stages) — hence the
expected `PLANE-009` status after Stage 3 is `PARTIAL`.

---

# 10. Portal failure during transit

Final Spec §16:

```text
If Portal becomes CLOSED or COLLAPSED before transit ends:
Observer → LOST
```

This applies equally to:

```text
OUTBOUND
RETURNING
```

Use `Portal.ClosedAt` from Stage 2 as the canonical terminal timestamp.

## Rule A — terminal before transit end

```text
portal.ClosedAt < observer.PhaseEndsAt
→ Observer LOST at portal.ClosedAt
```

The loss timestamp is the Portal terminal timestamp, not resolver `now`.

## Rule B — Portal survives through transit end

```text
portal.ClosedAt == nil
OR
portal.ClosedAt >= observer.PhaseEndsAt
```

The transit succeeds at `observer.PhaseEndsAt`.

## Rule C — exact tie

The spec says **"before transit ends"**.
Therefore exact equality is not a transit failure:

```text
portal.ClosedAt == observer.PhaseEndsAt
→ transit succeeds
```

This tie rule MUST have explicit regression tests for both OUTBOUND and RETURNING.

## Rule D — terminal after successful transit

If the Portal closes after transit already ended, it cannot retroactively lose the Observer.

Example:

```text
transit end     T=10
Portal ClosedAt T=12
resolve at      T=20
```

Correct catch-up result:

```text
OUTBOUND succeeded at T=10
research started at T=10
Portal close at T=12 does not affect Observer
```

Do not implement failure as merely:

```go
if portal.Status != PortalStatusOpen { observer = LOST }
```

That is incorrect because current Portal status loses the historical ordering.

---

# 11. Catch-up resolution

Lifecycle resolution must tolerate time jumps without `time.Sleep`.

Example:

```text
OUTBOUND starts at T=0
transit ends at T=10
research ends at T=30
resolver called at T=40
Portal survived transit
```

One resolver call should produce the state that is true at T=40:

```text
OUTBOUND → EXPLORING at T=10
EXPLORING → WAITING_RETURN at T=30
final state at T=40 = WAITING_RETURN
```

It is acceptable for implementation to perform multiple deterministic internal transitions in one call.

The resolver MUST terminate and MUST NOT repeatedly process an already completed phase.

RETURNING can produce at most one automatic transition:

```text
RETURNING → AVAILABLE
```

WAITING_RETURN has no automatic deadline.

AVAILABLE and LOST have no automatic deadline.

---

# 12. Proposed Stage 3 domain API

Exact Go naming may be adjusted, but responsibilities must remain separated.

Suggested API:

```go
func NewObserver(id int64, now time.Time) Observer

func NewObserverRoster(count int, now time.Time) []Observer

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

Alternative pure-function shapes are acceptable if tests and invariants are clearer.

The important separation is:

```text
StartOutbound / StartReturning
= lifecycle primitive after command admission has already succeeded.

ResolveObserverLifecycle
= deterministic time progression / transit outcome.
```

Stage 3 functions MUST NOT make SEND/RECALL policy decisions from Risk, creatures, Portal Flow, Lab Energy, or global Observer collections.

---

# 13. StartOutbound primitive

Precondition:

```text
Observer.Status == AVAILABLE
```

On success:

```text
Status          = OUTBOUND
CurrentPlaneID  = nil
ActivePortalID  = portalID
PhaseStartedAt  = now
PhaseEndsAt     = now + one random transit duration
UpdatedAt       = now
```

`CreatedAt` is unchanged.

Starting OUTBOUND MUST NOT mutate Plane exploration state.

Calling it from any non-AVAILABLE state returns a stable domain error and performs no partial mutation.

Stage 3 does not check whether the Portal is eligible for SEND.
Stage 4 guarantees that before calling this primitive.

---

# 14. Successful OUTBOUND resolution

At the transit deadline, if the Portal did not become terminal before the deadline:

```text
Status          = EXPLORING
CurrentPlaneID  = portal.DestinationPlaneID
ActivePortalID  = nil
PhaseStartedAt  = outbound transit end
PhaseEndsAt     = outbound transit end + cfg.ResearchDuration
UpdatedAt       = outbound transit end
```

Plane remains unchanged:

```text
Explored   unchanged
ExploredAt unchanged
```

The destination Plane passed to the resolver must match the Portal destination.
Mismatch is a programming/invariant error and must not silently mutate the wrong Plane.

---

# 15. Research completion resolution

When:

```text
Observer.Status == EXPLORING
now >= PhaseEndsAt
```

transition at the research deadline:

```text
Status          = WAITING_RETURN
CurrentPlaneID  = existing Plane ID
ActivePortalID  = nil
PhaseStartedAt  = research completion time
PhaseEndsAt     = nil
UpdatedAt       = research completion time
```

Plane remains UNEXPLORED if it was UNEXPLORED.

This explicit waiting timestamp is later used by Stage 4 to choose the longest-waiting Observer.

---

# 16. StartReturning primitive

Precondition:

```text
Observer.Status == WAITING_RETURN
Observer.CurrentPlaneID != nil
```

On success:

```text
Status          = RETURNING
CurrentPlaneID  = existing Plane ID
ActivePortalID  = portalID
PhaseStartedAt  = now
PhaseEndsAt     = now + one newly generated transit duration
UpdatedAt       = now
```

A return transit gets its own random duration draw.
It does not reuse the prior outbound duration.

Calling from AVAILABLE / OUTBOUND / EXPLORING / RETURNING / LOST returns a stable domain error with no mutation.

Stage 3 does not choose which WAITING_RETURN Observer to recall.
That is Stage 4.

---

# 17. Successful RETURNING resolution

At the transit deadline, if the Portal did not become terminal before the deadline:

```text
Observer.Status          = AVAILABLE
Observer.CurrentPlaneID  = nil
Observer.ActivePortalID  = nil
Observer.PhaseStartedAt  = nil
Observer.PhaseEndsAt     = nil
Observer.UpdatedAt       = return transit end
```

At exactly the same effective timestamp:

```text
if Plane.Explored == false:
    Plane.Explored = true
    Plane.ExploredAt = return transit end
```

If already explored, preserve the original `ExploredAt`.

This successful return is the only Stage 3 transition that may explore a Plane.

---

# 18. LOST transition

A transit failure occurs only while:

```text
Observer.Status == OUTBOUND || Observer.Status == RETURNING
```

and only when:

```text
Portal.ClosedAt < Observer.PhaseEndsAt
```

Transition at `Portal.ClosedAt`:

```text
Status          = LOST
CurrentPlaneID  = nil
ActivePortalID  = nil
PhaseStartedAt  = nil
PhaseEndsAt     = nil
UpdatedAt       = Portal.ClosedAt
```

Clearing the four current-state fields follows the Stage 3
canonicalization decision documented in §5.6 — the Final Spec does not
explicitly define them for LOST.

No Plane exploration mutation occurs.

LOST is terminal:

- cannot start OUTBOUND;
- cannot start RETURNING;
- lifecycle resolution cannot revive it;
- repeated resolution is idempotent.

Portal termination reason does not matter for loss semantics:

```text
MANUAL_CLOSE
NATURAL_CLOSE
ENERGY_DEPLETED
INSTABILITY
```

Any CLOSED/COLLAPSED terminal transition before transit end is sufficient.

---

# 19. Stable domain errors

Add stable domain errors only for lifecycle preconditions actually owned by Stage 3.

Suggested minimum:

```go
var (
    ErrObserverNotAvailable     = errors.New("observer is not available")
    ErrObserverNotWaitingReturn = errors.New("observer is not waiting for return")
    ErrObserverLost             = errors.New("observer is lost")
    ErrObserverInvariant        = errors.New("observer lifecycle invariant violated")
)
```

Exact decomposition may be smaller if callers can distinguish states without ambiguity.

Do NOT add Stage 4 command-policy errors yet, such as:

```text
portal direction conflict
portal busy
critical risk
creatures present
no available observer
no waiting observer
```

---

# 20. Test fixtures

Add `ObserverBuilder` / helpers only if they reduce duplication without hiding important timestamps.

Recommended deterministic fixtures:

```text
BaseTime             = existing testutil.BaseTime
Observer ID          = 1
Plane ID             = 7
Portal ID            = 11
Portal destination   = Plane 7
Transit duration     = FakeRandom → 10 sec unless boundary test
Research duration    = 20 sec
```

Useful helpers may include:

```go
NewObserverBuilder()
Available()
Outbound(portalID, start, end)
Exploring(planeID, start, end)
WaitingReturn(planeID, since)
Returning(planeID, portalID, start, end)
Lost()
Build()
```

But test setup must keep phase timestamps visually obvious.

No fixture should call production lifecycle methods merely to manufacture the state being tested.

---

# 21. TDD checkpoint A — Observer creation and invariants

Write tests first.

Required tests:

```text
TestNewObserver_StartsAvailableInLaboratory
TestNewObserverRoster_CreatesConfiguredCount
TestNewObserverRoster_DefaultConfigCreatesTen
TestObserver_AvailableCanonicalFields
TestObserver_LostIsTerminal
```

RED must demonstrate missing construction/behavior.

Minimal GREEN:

- constructor/roster helper;
- canonical AVAILABLE fields;
- no unrelated orchestration.

Traceability target:

```text
OBSERVER-001    # expected PARTIAL: roster/count=10 only; permanence/persistence bootstrap — later
OBSERVER-002
OBSERVER-003
OBSERVER-004
OBSERVER-005
```

---

# 22. TDD checkpoint B — Start OUTBOUND and random duration

Required tests:

```text
TestObserver_StartOutboundTransitionsFromAvailable
TestObserver_StartOutboundTransitMinimumFiveSeconds
TestObserver_StartOutboundTransitMaximumFifteenSeconds
TestObserver_StartOutboundDrawsTransitDurationExactlyOnce
TestObserver_StartOutboundDoesNotExplorePlane
TestObserver_StartOutboundRejectsNonAvailableState
TestObserver_StartOutboundRejectsLostObserver
TestObserver_StartOutboundFailureIsAtomic
```

Use `FakeRandom` and assert consumption count/index.

Minimum and maximum boundaries must use current config values rather than magic behavior hidden in tests.

Traceability target:

```text
OBSERVER-006
OBSERVER-007
PLANE-005       # expected PARTIAL: primitive does not explore; SEND command — Stage 4
```

---

# 23. TDD checkpoint C — Successful OUTBOUND

Required tests:

```text
TestObserver_OutboundBeforeDeadlineRemainsOutbound
TestObserver_OutboundAtDeadlineBecomesExploring
TestObserver_OutboundArrivalUsesDeadlineAsTransitionTime
TestObserver_OutboundArrivalSetsCurrentPlane
TestObserver_OutboundArrivalClearsActivePortal
TestObserver_OutboundArrivalStartsTwentySecondResearch
TestObserver_OutboundArrivalDoesNotExplorePlane
TestObserver_OutboundResolveDoesNotConsumeRandom
```

Boundary:

```text
now = PhaseEndsAt - 1ns → still OUTBOUND
now = PhaseEndsAt       → EXPLORING
```

Traceability target:

```text
OBSERVER-008
OBSERVER-009
PLANE-006
```

---

# 24. TDD checkpoint D — Research completion

Required tests:

```text
TestObserver_ExploringBeforeDeadlineRemainsExploring
TestObserver_ResearchCompletesAtDeadline
TestObserver_ResearchCompletionUsesDeadlineAsTransitionTime
TestObserver_ResearchCompletionBecomesWaitingReturn
TestObserver_WaitingReturnStartsAtResearchCompletion
TestObserver_ResearchCompletionDoesNotExplorePlane
TestObserver_WaitingReturnHasNoAutomaticEnd
```

Boundary:

```text
now = PhaseEndsAt - 1ns → EXPLORING
now = PhaseEndsAt       → WAITING_RETURN
```

Traceability target:

```text
OBSERVER-009
OBSERVER-010
PLANE-007
```

---

# 25. TDD checkpoint E — Start RETURNING and successful return

Required tests:

```text
TestObserver_StartReturningFromWaitingReturn
TestObserver_StartReturningDrawsFreshTransitDuration
TestObserver_StartReturningDrawsTransitDurationExactlyOnce
TestObserver_StartReturningPreservesCurrentPlaneDuringTransit
TestObserver_StartReturningRejectsInvalidStates
TestObserver_StartReturningFailureIsAtomic

TestObserver_ReturningBeforeDeadlineRemainsReturning
TestObserver_ReturningAtDeadlineBecomesAvailable
TestObserver_ReturnUsesDeadlineAsTransitionTime
TestObserver_ReturnClearsCurrentPlaneAndActivePortal
TestObserver_SuccessfulReturnExploresPlane
TestObserver_SuccessfulReturnSetsExploredAtToArrival
TestObserver_ReturnToAlreadyExploredPlanePreservesOriginalExploredAt
TestObserver_ReturnResolveDoesNotConsumeRandom
```

Traceability target:

```text
OBSERVER-011
PLANE-008
PLANE-009       # expected PARTIAL: Plane-state idempotence only (see §9)
```

---

# 26. TDD checkpoint F — Transit failure / LOST

Test both outbound and return direction.

Required tests:

```text
TestObserver_OutboundLostWhenPortalClosesBeforeTransitEnd
TestObserver_OutboundLostWhenPortalCollapsesBeforeTransitEnd
TestObserver_ReturningLostWhenPortalClosesBeforeTransitEnd
TestObserver_ReturningLostWhenPortalCollapsesBeforeTransitEnd
TestObserver_LossUsesPortalClosedAtAsTransitionTime
TestObserver_LossClearsActiveState
TestObserver_LossDoesNotExplorePlane
TestObserver_LostCannotBeRevivedByResolve
TestObserver_LostCannotStartOutbound
TestObserver_LostCannotStartReturning
```

Traceability target:

```text
OBSERVER-005
OBSERVER-012
OBSERVER-013
```

---

# 27. TDD checkpoint G — ordering, exact ties, catch-up

These are regression tests for subtle time semantics.

Required tests:

```text
TestObserver_OutboundSucceedsWhenPortalClosesExactlyAtTransitEnd
TestObserver_ReturningSucceedsWhenPortalClosesExactlyAtTransitEnd
TestObserver_PortalClosingAfterOutboundEndDoesNotRetroactivelyLoseObserver
TestObserver_PortalClosingAfterReturnEndDoesNotRetroactivelyLoseObserver
TestObserver_CatchUpOutboundThroughResearchEndsWaitingReturn
TestObserver_CatchUpUsesEffectiveTransitionTimestamps
TestObserver_RepeatedResolveIsIdempotent
```

Also test a large jump:

```text
start OUTBOUND at T=0
transit end at T=10
research end at T=30
resolve at T=1h
→ WAITING_RETURN since T=30
```

No real sleeping and no repeated random draws.

---

# 28. TDD checkpoint H — multiple Observers in one Plane

Required test:

```text
TestObservers_MultipleObserversMayExploreSamePlaneConcurrently
```

This Stage 3 test should prove that lifecycle code itself does not impose uniqueness by Plane.

At minimum:

- two Observers can be EXPLORING the same `Plane.ID`;
- two Observers can be WAITING_RETURN in the same Plane;
- first successful return explores Plane;
- later successful return preserves original `ExploredAt`.

Do not add a global "one observer per plane" guard.

Traceability target:

```text
OBSERVER-014
PLANE-009       # expected PARTIAL: Plane-state idempotence only (see §9)
```

---

# 29. Resolver invariant checks

Lifecycle code should reject impossible/corrupted state rather than silently guessing.

Examples:

```text
OUTBOUND without ActivePortalID
OUTBOUND without PhaseEndsAt
EXPLORING without CurrentPlaneID
EXPLORING without PhaseEndsAt
WAITING_RETURN without CurrentPlaneID
RETURNING without CurrentPlaneID
RETURNING without ActivePortalID
RETURNING without PhaseEndsAt
```

These invariant checks are defensive domain correctness for later persistence/restart.

Do not over-engineer a generic validator if small explicit checks in transition code are clearer.

---

# 30. Interaction with Stage 2 Portal lifecycle

Stage 3 may read from Portal:

```text
ID
DestinationPlaneID
Status
ClosedAt
```

Stage 3 MUST NOT change Stage 2 Portal terminal semantics.

Especially do not:

- move Observer loss logic into `Portal.Close`;
- make `Portal.ResolveLifecycle` mutate Observers;
- make Portal own an Observer pointer;
- make `Portal.Close` search a global Observer collection.

The final cross-aggregate orchestration belongs in `LabManager` later.

For Stage 3 tests, an already resolved terminal Portal with a correct `ClosedAt` is sufficient input.

---

# 31. Interaction with future Stage 4

Stage 4 will orchestrate the command-level behavior around these primitives.

Expected Stage 4 usage conceptually:

```text
SEND command
→ validate Portal/Risk/creatures/flow/busy/available observer
→ choose Observer
→ set/validate Portal flow
→ observer.StartOutbound(...)

RECALL command
→ validate Portal/Risk/creatures/flow/busy/waiting observers
→ choose longest-waiting Observer
→ set/validate Portal flow
→ observer.StartReturning(...)
```

Therefore Stage 3 APIs should be usable by Stage 4 without duplicating lifecycle mutation.

---

# 32. Interaction with future LabManager invariant

The Stage 0–2 audit fixed this future invariant:

```text
Before any time-sensitive command/read, LabManager resolves Portal lifecycle
under the same lock before executing the requested action.
```

Stage 3 does not implement the manager lock yet.

However its time model must be compatible with later orchestration:

1. Portal terminal transition gets a precise `ClosedAt`.
2. Observer resolver compares `ClosedAt` with transit `PhaseEndsAt`.
3. Exact ordering determines success vs LOST.
4. No transition depends on scheduler/tick timing.

This prevents Stage 11 from needing to rewrite Observer semantics.

---

# 33. No event-system implementation yet

The enum values already exist:

```text
OBSERVER_DISPATCHED
OBSERVER_ARRIVED
RESEARCH_STARTED
RESEARCH_COMPLETED
OBSERVER_RETURN_STARTED
OBSERVER_RETURNED
OBSERVER_LOST
PLANE_EXPLORED
```

Stage 3 must NOT build durable event persistence/emission architecture.
That is Stage 9.

Tests in Stage 3 verify state transitions directly.

If implementation needs an internal transition result to avoid later duplicate logic,
a small non-persistent transition enum/value is acceptable, but do not prematurely construct Event Log infrastructure.

---

# 34. No persistence assumptions

Stage 3 must keep all state needed for future restart recovery in the existing current-state fields:

```text
status
current_plane_id
active_portal_id
phase_started_at
phase_ends_at
updated_at
```

This is why transit duration must be materialized into `phase_ends_at` when started.

Do not store:

```text
remaining_seconds
research_remaining
transit_remaining
```

Those are derived from timestamps.

---

# 35. No wall-clock access in domain code

Stage 3 domain logic MUST NOT call:

```go
time.Now()
time.Sleep(...)
time.After(...)
time.NewTicker(...)
```

Time enters through explicit `now time.Time` arguments or the existing Clock abstraction at orchestration boundaries.

Observer lifecycle tests must remain instantaneous and deterministic.

---

# 36. Race-safety boundary

Stage 3 domain structs are not required to contain their own mutexes.

Do NOT add mutexes inside:

```text
Observer
Plane
Portal
```

Atomic cross-entity mutation is a `LabManager` responsibility in Stage 11.

Nevertheless every checkpoint must still run:

```bash
go test -race -count=1 ./...
```

This catches accidental shared mutable globals and regressions in existing concurrency-safe test utilities.

---

# 37. Suggested production files

Prefer small domain-focused files, for example:

```text
internal/domain/observer.go
internal/domain/observer_lifecycle.go
internal/domain/observer_errors.go   # optional; existing errors.go is also fine
```

Tests may be split by behavior:

```text
internal/domain/observer_test.go
internal/domain/observer_transit_test.go
internal/domain/observer_research_test.go
internal/domain/observer_return_test.go
internal/domain/observer_loss_test.go
```

Exact filenames are flexible.

Do not create Stage 4 command service files during Stage 3.

---

# 38. TDD execution protocol

For each checkpoint:

```text
1. Map requirement IDs.
2. Write focused tests.
3. Run targeted tests and capture RED.
4. Commit RED tests.
5. Implement minimum behavior.
6. Run targeted tests → GREEN.
7. Run full suite.
8. Run race suite.
9. Refactor only while GREEN.
10. Update traceability.
11. Commit GREEN implementation.
```

Do not fake RED by intentionally breaking existing production code.
RED should arise because the new requirement is not implemented yet.

A regression test that already passes is acceptable only when explicitly documented as a characterization/regression test, exactly as with the Stage 2 tie regression.

---

# 39. Required verification commands

At each meaningful GREEN checkpoint:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
go test -race -count=1 ./...
```

`gofmt -l .` must produce no project Go files.

No `time.Sleep` may be introduced in domain tests.

Useful grep checks:

```bash
grep -Rni 'time.Sleep\|time.Now()\|time.NewTicker\|time.After' internal/domain
```

Any match must be reviewed and justified; expected Stage 3 domain result is none.

Also check that Stage 4 policy did not leak in:

```bash
grep -Rni 'ObserverFlow\|RiskCritical\|CreaturesInside' internal/domain/observer*.go
```

Stage 3 lifecycle implementation should not depend on these command-admission concepts.

---

# 40. Traceability updates expected after Stage 3

After implementation, `docs/traceability.md` should approximately become:

```text
OBSERVER-001       PARTIAL  # roster/count=10 proven; permanent/persistence bootstrap — later stages
OBSERVER-002..014  GREEN

PLANE-005          PARTIAL  # StartOutbound primitive does not explore; real SEND command — Stage 4
PLANE-006..008     GREEN
PLANE-009          PARTIAL  # Plane state idempotence (Explored/ExploredAt) proven; "progress not increased again" — with progress aggregation later

OBSERVER-015       PLANNED  # Stage 4 SEND policy
OBSERVER-016       PLANNED  # Stage 4 RECALL selection
FLOW-001..007      PLANNED  # Stage 4
```

`PARTIAL` here follows the semantics fixed in `docs/traceability.md`:
the covered helper/primitive/subset is proven by concrete tests, the
remaining orchestration/behavior belongs to a later stage. Do not mark
these rows fully `GREEN` after Stage 3.

Every GREEN row must name:

- concrete test(s);
- concrete implementation symbol(s);
- any remaining orchestration boundary in Notes.

`docs/requirements.md` should remain semantically unchanged unless a genuine spec interpretation needs documentation.

---

# 41. Worklog update expected

`01_AI_WORKLOG_CURRENT.md` must record:

- Stage 3 execution agent/model;
- checkpoint RED/GREEN commits;
- tests added;
- exact tie rule (`ClosedAt == PhaseEndsAt` → transit succeeds);
- catch-up semantics;
- whether LOST canonicalization clears current Plane/phase fields;
- verification command results;
- explicit statement that Stage 4 was not started.

Do not claim commands passed unless actually executed.

---

# 42. Stage 3 acceptance checklist

Stage 3 is complete only if all are true:

- [ ] 10-observer roster behavior covered.
- [ ] New Observer starts AVAILABLE in Lab.
- [ ] AVAILABLE canonical fields enforced.
- [ ] OUTBOUND starts only from AVAILABLE lifecycle primitive.
- [ ] Transit duration generated once in inclusive 5..15 sec range.
- [ ] Successful OUTBOUND enters EXPLORING at exact transit end.
- [ ] Research lasts exactly configured 20 sec.
- [ ] Research completion enters WAITING_RETURN.
- [ ] Waiting timestamp supports future longest-waiting selection.
- [ ] RETURNING starts only from WAITING_RETURN lifecycle primitive.
- [ ] RETURNING draws a fresh duration exactly once.
- [ ] Successful return enters AVAILABLE and clears location/transit fields.
- [ ] SEND lifecycle primitive (`StartOutbound`) does not explore Plane (PLANE-005 stays PARTIAL until the Stage 4 SEND command).
- [ ] OUTBOUND arrival does not explore Plane.
- [ ] Research completion does not explore Plane.
- [ ] Successful return explores Plane.
- [ ] Re-return to explored Plane preserves first `ExploredAt` (Plane-state half of PLANE-009; row stays PARTIAL until progress aggregation exists).
- [ ] CLOSED-before-end loses OUTBOUND Observer.
- [ ] COLLAPSED-before-end loses OUTBOUND Observer.
- [ ] CLOSED-before-end loses RETURNING Observer.
- [ ] COLLAPSED-before-end loses RETURNING Observer.
- [ ] Loss uses Portal `ClosedAt`, not resolver time.
- [ ] Exact `ClosedAt == PhaseEndsAt` tie succeeds.
- [ ] Portal closing after transit end cannot retroactively lose Observer.
- [ ] LOST is terminal and cannot be revived.
- [ ] Multiple Observers may occupy the same Plane.
- [ ] Large time jumps/catch-up resolve correctly.
- [ ] Repeated lifecycle resolution is idempotent.
- [ ] Resolver consumes no randomness.
- [ ] No `time.Sleep` in domain tests.
- [ ] No Stage 4 `observer_flow` or SEND/RECALL restrictions implemented.
- [ ] No Lab Energy implementation.
- [ ] No Extraction implementation.
- [ ] No persistence/API/event-system implementation.
- [ ] `gofmt -l .` clean.
- [ ] `go vet ./...` passes.
- [ ] `go build ./...` passes.
- [ ] `go test -count=1 ./...` passes.
- [ ] `go test -race -count=1 ./...` passes.
- [ ] Traceability updated honestly.
- [ ] Worklog updated.
- [ ] Working tree clean after final Stage 3 commit(s).
- [ ] Stage 4 not started.

---

# 43. Explicit non-goals

Do NOT implement in Stage 3:

```text
SEND command admission
RECALL command admission
Portal observer_flow mutation
one-transit-per-Portal global enforcement
longest-waiting selection
UNSTABLE confirmation warning
Risk-based command blocking
creature-based command blocking
Lab Energy costs
manual-close observer confirmation orchestration
Extraction Portal
automatic Extraction recall
domain Event persistence/emission
SQLite repositories
REST endpoints
WebSocket snapshots
simulation goroutine
LabManager mutex orchestration
```

If a test requires one of these to pass, the test belongs to a later stage.

---

# 44. Stop condition

Once all Stage 3 acceptance checks are GREEN:

```text
STOP.
```

Do not begin Stage 4 in the same pass.

Produce a final Stage 3 report containing:

```text
- files changed;
- tests added;
- RED evidence;
- GREEN evidence;
- race result;
- traceability changes;
- commit hashes;
- any characterization tests that were already GREEN;
- confirmation that Stage 4 was not started.
```

---

# 45. Next stage after completion

Stage 4:

```text
Portal Observer Flow via TDD

NONE / OUTBOUND / INBOUND
one transit per Portal
SEND admission
RECALL admission
longest-waiting selection
UNSTABLE warning boundary
Risk / creatures restrictions
manual-close active-transit interaction
```

Stage 4 must consume the Stage 3 lifecycle primitives rather than reimplementing Observer state transitions.
