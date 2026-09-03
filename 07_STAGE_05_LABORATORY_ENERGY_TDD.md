# Omenpath Research Lab — Stage 5
## Laboratory Energy and Action Cost Orchestration via TDD

> **For agentic workers:** REQUIRED SUB-SKILL: use
> `superpowers:subagent-driven-development` when explicitly permitted, or
> `superpowers:executing-plans` to execute this plan checkpoint-by-checkpoint.
>
> Depends on:
> - `00_FINAL_SPEC_v5.md`
> - `01_AI_WORKLOG_CURRENT.md`
> - `02_IMPLEMENTATION_ROADMAP_TDD.md`
> - `04_STAGE_02_PORTAL_CORE_TDD.md`
> - `05_STAGE_03_OBSERVER_LIFECYCLE_TDD.md`
> - `06_STAGE_04_PORTAL_OBSERVER_FLOW_TDD.md`

**Goal:** implement integer Laboratory Energy, derived regeneration, ordinary
action costs, insufficient-energy rejection, and atomic cost orchestration
around the completed Stage 2/4 domain commands.

**Architecture:** keep `LabState` as a timestamped integer baseline. Add pure
derived-energy methods and thin energy-aware command wrappers; do not move
cost logic into `Portal`, `Observer`, or completed Stage 4 commands. Paid
wrappers preflight affordability, delegate the existing domain command, and
commit one energy debit only after command success.

**Tech stack:** Go, `time.Time`/`time.Duration`, existing `config.Config`,
`testify/require`, `testutil.FakeRandom`, and repository Git/TDD protocol.

---

# 1. Goal

Stage 5 answers:

```text
What is current Laboratory Energy at time now?
Can the Laboratory pay the configured ordinary action cost?
If the action succeeds, what is the new energy baseline?
If it fails, can every aggregate remain unchanged?
```

The required behavior is:

```text
integer energy in 0..100
derived regeneration at +1 per completed second
cap at 100
no per-second state writes

SEND        costs 0
RECALL      costs 0
CLOSE       costs 5
STABILIZE   costs 20
EXTRACTION  costs 30

insufficient energy rejects a paid action without mutation
```

Stage 5 must reuse:

```text
Stage 2  Portal.Stabilize / Portal.Close primitives
Stage 3  Observer lifecycle primitives
Stage 4  SendObserver / RecallObserver / ClosePortalWithObservers
```

It must not duplicate or weaken any of their restrictions.

---

# 2. Source-of-truth order

Use the repository hierarchy fixed in `AGENTS.md`:

```text
00_FINAL_SPEC_v5.md
→ 01_AI_WORKLOG_CURRENT.md (history only; never overrides Final Spec)
→ 02_IMPLEMENTATION_ROADMAP_TDD.md
→ this Stage 5 detailed plan
→ docs/requirements.md
→ tests
→ implementation
```

If any instruction below conflicts with Final Spec, Final Spec wins. Stop on a
real gameplay contradiction instead of inventing semantics.

---

# 3. Verified starting point

Stage 4 ended at commit `6b1728a` with:

```go
func SendObserver(
    portal *Portal,
    plane *Plane,
    observers []Observer,
    now time.Time,
    confirmUnstable bool,
    rnd random.Random,
    cfg config.Config,
) (int64, error)

func RecallObserver(
    portal *Portal,
    plane *Plane,
    observers []Observer,
    now time.Time,
    confirmUnstable bool,
    rnd random.Random,
    cfg config.Config,
) (int64, error)

func ClosePortalWithObservers(
    portal *Portal,
    plane *Plane,
    observers []Observer,
    now time.Time,
    confirm bool,
    cfg config.Config,
) error
```

Stage 2 already provides:

```go
func (p *Portal) Stabilize(now time.Time, cfg config.Config) error
func (p *Portal) Close(now time.Time, confirm bool, cfg config.Config) error
```

The existing `LabState` skeleton is:

```go
type LabState struct {
    EnergyBase   int
    EnergyBaseAt time.Time

    LeylineOverrideUntil *time.Time
}
```

The override field exists for schema/domain continuity but Stage 6 behavior is
not implemented. Stage 5 ordinary costs assume no active override.

`config.Default()` already pins:

```text
LAB_ENERGY_MAX      100
LAB_REGEN_PER_SEC   1
CLOSE_COST          5
STABILIZE_COST      20
EXTRACTION_COST     30
```

Stage 5 consumes these values instead of adding literals to domain code.

---

# 4. Requirements covered

Primary requirement family:

```text
LAB-001  energy is an integer in 0..100
LAB-002  Tutorial starts with 100
LAB-003  +1/sec derived regeneration from baseline timestamp
LAB-004  regeneration caps at 100
LAB-005  SEND costs 0
LAB-006  RECALL costs 0
LAB-007  CLOSE costs 5
LAB-008  STABILIZE costs 20
LAB-009  EXTRACTION costs 30
LAB-010  insufficient energy rejects paid actions
```

Integration rows enhanced but not semantically changed:

```text
PORTAL-006    manual Close remains CLOSED/MANUAL_CLOSE
STABILITY-005 Stabilize remains UNSTABLE → STABLE
ENERGY-006    Stabilize still adds 15 Portal Energy
ENERGY-007    Stabilize still re-baselines Portal Energy
ENERGY-008    Stabilize still preserves Portal decay
FLOW-002..007 Stage 4 SEND/RECALL semantics remain unchanged
```

Stage 5 does not complete:

```text
EMERGENCY-001..006   # Stage 6
EXTRACTION-001..010  # Stage 7
```

`LAB-009` can prove the configured 30-point debit contract, but the actual
Extraction command remains Stage 7. `LAB-010` remains PARTIAL until Stage 6
adds override exceptions and Stage 7 exercises insufficient energy through
the real Extraction command.

---

# 5. Hard stage boundary

Stage 5 MUST NOT implement:

```text
Collapse → Lab Energy 0
Leyline Override activation or expiry
free Close/Stabilize during Override
Extraction Portal creation
Extraction synchronization or automatic RECALL
natural Portal spawning
simulation tick/goroutine
Events or ACTION_REJECTED
SQLite persistence/recovery
LabManager, locks, or concurrent command arbitration
REST/HTTP/WebSocket/frontend behavior
Tutorial step advancement or Tutorial → Live transition
```

The presence of `LeylineOverrideUntil` in `LabState` does not authorize Stage 6
logic. Tests in this stage use the ordinary, no-override path.

---

# 6. Architecture choice

Use two focused production files:

```text
internal/domain/lab.go                  # LabState and derived-energy methods
internal/domain/lab_energy_commands.go  # energy-aware action wrappers
```

Responsibilities:

```text
LabState
  → construction and invariants
  → derived CurrentEnergy(now)
  → affordability
  → atomic re-baselined debit

energy-aware wrappers
  → choose configured action cost
  → reject insufficient energy before action mutation
  → delegate completed Stage 2/4 command
  → debit exactly once after success
```

Portal and Observer must remain unaware of Laboratory Energy.

---

# 7. Alternatives considered

## A. LabState primitives plus thin wrappers — selected

Advantages:

- preserves completed Stage 2–4 APIs and semantics;
- keeps energy derivation independently testable;
- gives Stage 11 `LabManager` one future orchestration boundary;
- supports atomic failure without persistence, locks, or events;
- lets Stage 6 change effective Close/Stabilize cost without moving energy
  fields into Portal.

## B. Add `LabState` directly to Stage 4 commands — rejected

This would change completed Stage 4 signatures and mix Portal/Observer
admission with a later resource system. It also makes zero-cost commands
unnecessarily dependent on mutable energy internals.

## C. Implement costs inside `Portal.Close` and `Portal.Stabilize` — rejected

Portal must not own Laboratory state. This would violate aggregate boundaries
and make Stage 6 override logic leak into Portal primitives.

## D. Implement on `LabManager` now — rejected

Locks, resolve-first orchestration, persistence coordination and event
deduplication belong to Stage 11.

---

# 8. Proposed LabState API

Suggested public surface:

```go
func NewLabState(
    initialEnergy int,
    now time.Time,
    cfg config.Config,
) (LabState, error)

func NewTutorialLabState(
    now time.Time,
    cfg config.Config,
) LabState

func (l LabState) CurrentEnergy(
    now time.Time,
    cfg config.Config,
) int

func (l LabState) CanAfford(
    now time.Time,
    cost int,
    cfg config.Config,
) bool

func (l *LabState) SpendEnergy(
    now time.Time,
    cost int,
    cfg config.Config,
) error
```

Exact names may improve during the first RED cycle. The responsibility split
and behavior may not change without updating this plan.

`CanAfford` is conservative: it returns `false` for a negative cost, malformed
baseline, or backward mutation time. `SpendEnergy` returns the corresponding
typed invariant error, so command wrappers do not need to infer error identity
from the boolean result.

---

# 9. Proposed action-wrapper API

Suggested functions:

```go
func SendObserverWithLabEnergy(
    lab *LabState,
    portal *Portal,
    plane *Plane,
    observers []Observer,
    now time.Time,
    confirmUnstable bool,
    rnd random.Random,
    cfg config.Config,
) (observerID int64, err error)

func RecallObserverWithLabEnergy(
    lab *LabState,
    portal *Portal,
    plane *Plane,
    observers []Observer,
    now time.Time,
    confirmUnstable bool,
    rnd random.Random,
    cfg config.Config,
) (observerID int64, err error)

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

These wrappers are pure in-memory domain orchestration. They do not resolve
time-sensitive Portal/Observer lifecycle first; that remains the future
Stage 11 caller invariant.

---

# 10. Integer regeneration semantics

Final Spec requires integer energy and `+1/sec`. Therefore regeneration uses
completed whole seconds:

```text
elapsed = max(0, now - EnergyBaseAt)
whole_seconds = floor(elapsed / 1 second)
current = min(LAB_ENERGY_MAX,
              EnergyBase + whole_seconds × LAB_REGEN_PER_SEC)
```

Examples with defaults:

```text
base 40 at T
T + 999ms   → 40
T + 1s      → 41
T + 10.9s   → 50
T + 60s     → 100, not 101
```

This is a technical consequence of “integer only” plus “+1/sec”, not a new
gameplay rate.

---

# 11. Derived state, not ticking state

`CurrentEnergy` must not mutate `LabState`:

```text
read at T+10s → calculated value
read again    → same stored EnergyBase/EnergyBaseAt
```

No ticker, goroutine, sleep, database write or per-second assignment belongs
in Stage 5. Meaningful action debits are the only Stage 5 re-baseline events.

---

# 12. Cap and lower bound

Valid persisted/baseline state is:

```text
0 <= EnergyBase <= cfg.LabEnergyMax
```

Constructors and mutations must preserve it. `CurrentEnergy` is total and
clamps its result into the same range, but mutation methods reject malformed
state instead of silently repairing it.

No action may produce negative energy.

---

# 13. Construction

`NewLabState` accepts inclusive boundaries:

```text
0
cfg.LabEnergyMax
```

and rejects values outside them with a stable domain invariant error.

`NewTutorialLabState(now, cfg)` creates:

```text
EnergyBase = cfg.LabEnergyMax  # default 100
EnergyBaseAt = now
LeylineOverrideUntil = nil
```

Full Tutorial bootstrap and mode transition remain Stage 14. Therefore
`LAB-002` is PARTIAL after this stage even though the domain constructor is
covered.

---

# 14. Time ordering

Pure reads before `EnergyBaseAt` treat elapsed time as zero. Mutating energy at
a time before the current baseline is rejected as an invariant violation:

```text
CurrentEnergy(EnergyBaseAt - 1s) → EnergyBase
SpendEnergy(EnergyBaseAt - 1s)   → invariant error, no mutation
```

This prevents a command from moving the baseline backwards while keeping
derived reads deterministic for hand-built fixtures.

---

# 15. Spending and re-baselining

For a positive affordable cost:

```text
current = CurrentEnergy(now, cfg)
new base = current - cost
new base timestamp = now
```

Example:

```text
base 10 at T
spend 5 at T+3s
current before spend = 13
stored after spend = 8 at T+3s
```

Re-baselining is required so already-earned regeneration is not counted twice.

---

# 16. Zero-cost semantics

For `cost == 0`:

```text
affordability always succeeds for a valid LabState
SpendEnergy returns nil
EnergyBase unchanged
EnergyBaseAt unchanged
LeylineOverrideUntil unchanged
```

SEND/RECALL wrappers therefore do not create meaningless energy transitions.

---

# 17. Insufficient energy

Add stable domain errors owned by Stage 5:

```go
var (
    ErrLabEnergyInvariant   = errors.New("laboratory energy invariant violated")
    ErrInsufficientLabEnergy = errors.New("insufficient laboratory energy")
)
```

For `current < cost`:

```text
return ErrInsufficientLabEnergy
LabState unchanged
Portal unchanged
Plane unchanged
Observers unchanged
random queue unchanged
```

Exact affordability is allowed.

---

# 18. Invalid cost

A negative cost is an internal invariant violation:

```text
SpendEnergy(now, -1, cfg)
→ ErrLabEnergyInvariant
→ no mutation
```

Do not interpret negative cost as an energy grant. Stage 5 has no generic
credit operation; regeneration is derived from time.

---

# 19. Wrapper transaction order

For ordinary paid actions, use this deterministic order:

```text
1. validate LabState pointer/baseline/time
2. derive current Lab Energy
3. reject if current < configured cost
4. delegate the completed domain command
5. if command succeeds, debit exactly once at the same now
```

Consequences:

- insufficient energy is detected before confirmation or action mutation;
- an underlying Portal/Observer error never consumes energy;
- energy debit cannot fail after a successful command because the same state,
  timestamp and already-validated cost are committed synchronously;
- Stage 11 later puts the whole sequence under one lock.

When energy is sufficient, the delegated command retains its established
Stage 2/4 validation and error precedence unchanged.

---

# 20. SEND cost orchestration

`SendObserverWithLabEnergy` uses:

```text
cost = 0
delegate = SendObserver
```

Required behavior:

```text
energy 0 still permits an otherwise-valid SEND
successful SEND does not change either Lab energy baseline field
rejected SEND does not change LabState
UNSTABLE confirmation behavior remains Stage 4 behavior
random draw remains exactly one only on successful SEND
```

Do not add a new SEND restriction based on low Laboratory Energy.

---

# 21. RECALL cost orchestration

`RecallObserverWithLabEnergy` uses:

```text
cost = 0
delegate = RecallObserver
```

Required behavior mirrors SEND:

```text
energy 0 permits otherwise-valid RECALL
no LabState mutation or re-baseline
Stage 4 destination/longest-waiting/direction/busy rules preserved
rejection consumes no energy and no random value
```

---

# 22. CLOSE cost orchestration

Ordinary Close uses:

```text
cost = cfg.CloseCost  # default 5
delegate = ClosePortalWithObservers
```

Required behavior:

```text
energy 5 + successful Close → energy 0
energy 4 → ErrInsufficientLabEnergy; Portal/Observer unchanged
confirmation-required Close → no debit
terminal/malformed Close → no debit
confirmed active-transit Close → Portal CLOSED + Observer LOST + one debit
```

Do not implement the Stage 6 free-during-override exception.

---

# 23. STABILIZE cost orchestration

Ordinary Stabilize uses:

```text
cost = cfg.StabilizeCost  # default 20
delegate = Portal.Stabilize
```

Required behavior:

```text
energy 20 + valid Stabilize → Lab energy 0
energy 19 → ErrInsufficientLabEnergy; Portal unchanged
successful action retains Stage 2 Portal +15/re-baseline/decay semantics
stable Portal rejection → no debit
Portal energy >85 rejection → no debit
terminal Portal rejection → no debit
```

Laboratory Energy and Portal Energy are distinct resources. Spending 20 Lab
Energy does not subtract from Portal Energy; `Portal.Stabilize` separately adds
15 to current Portal Energy.

---

# 24. EXTRACTION cost boundary

Stage 5 pins and exercises the generic debit with:

```text
cost = cfg.ExtractionCost  # default 30
```

It must prove:

```text
30 is affordable at 30
29 is insufficient
successful generic spend re-baselines to 0
```

It MUST NOT create an Extraction Portal or select a Plane. Consequently:

```text
LAB-009        PARTIAL after Stage 5
EXTRACTION-001 PLANNED until Stage 7
```

Stage 7 must call the same tested energy primitive rather than duplicating the
30-point arithmetic.

---

# 25. Leyline Override boundary

Stage 5 always applies ordinary Close/Stabilize costs. It neither checks nor
mutates `LeylineOverrideUntil`.

Stage 6 will add:

```text
active override → effective Close cost 0
active override → effective Stabilize cost 0
Extraction remains 30
regeneration continues
Collapse forces Lab Energy to 0
```

Stage 5 tests must not assert what a non-nil override deadline does. That would
start Stage 6 early.

---

# 26. Resolve-first boundary

As fixed in the roadmap, the future `LabManager` must resolve time-sensitive
Portal/Observer state before executing commands under the same lock.

Stage 5 wrappers assume:

```text
Portal and Observer state already resolved to now
Lab Energy is derived directly at now
```

They must not call `Portal.ResolveLifecycle` or start a simulation loop.

---

# 27. Mutation atomicity

On every error, compare snapshots of all supplied aggregates:

```text
LabState
Portal
Plane
Observers
```

For SEND/RECALL also prove the FakeRandom queue is untouched unless the
delegated Stage 4 command succeeds.

No rollback-by-reconstruction should be required. Affordability is checked
before delegation; debit occurs only after a successful delegate call.

---

# 28. No event semantics

Stage 5 does not emit:

```text
LAB_ENERGY_CHANGED
ACTION_REJECTED
PORTAL_CLOSED
PORTAL_STABILIZED
OBSERVER_SENT
OBSERVER_RECALLED
```

Those durable records belong to Stage 9. Do not add temporary event slices or
callbacks to the wrappers.

---

# 29. Concurrency boundary

No mutex belongs in `LabState` or a domain entity.

Stage 5 proves deterministic sequential transactions. Stage 11 will own:

```text
sync.RWMutex
resolve-first under lock
exactly one concurrent valid transition
no duplicate events
```

The repository-wide race suite remains mandatory even though Stage 5 adds no
goroutines.

---

# 30. Suggested production files

Create:

```text
internal/domain/lab_energy_commands.go
```

Modify:

```text
internal/domain/lab.go
internal/domain/errors.go
```

Do not modify Stage 2/4 implementation files unless a new RED regression
demonstrates a genuine reusable-boundary defect:

```text
internal/domain/portal.go
internal/domain/observer_commands.go
internal/domain/portal_observer_close.go
```

---

# 31. Suggested test files

Create focused tests:

```text
internal/domain/lab_energy_test.go
internal/domain/lab_energy_spend_test.go
internal/domain/lab_energy_send_test.go
internal/domain/lab_energy_recall_test.go
internal/domain/lab_energy_close_test.go
internal/domain/lab_energy_stabilize_test.go
internal/domain/lab_energy_extraction_cost_test.go
internal/domain/lab_energy_integration_test.go
```

Reuse existing Stage 4 test helpers only if package boundaries make that
clear. Do not construct initial state by calling the production wrapper under
test.

---

# 32. Test fixtures

Use fixed values:

```text
Base time            testutil.BaseTime
Lab max              config.Default().LabEnergyMax
Regen                 config.Default().LabRegenPerSec
Close cost            config.Default().CloseCost
Stabilize cost        config.Default().StabilizeCost
Extraction cost       config.Default().ExtractionCost
Portal ID             11
Destination Plane ID  7
Transit duration      FakeRandom 10 sec
```

Provide a small fixture helper such as:

```go
func labAt(energy int, at time.Time) domain.LabState {
    return domain.LabState{EnergyBase: energy, EnergyBaseAt: at}
}
```

No fixture may use `time.Now`, `time.Sleep`, real random, database, HTTP or a
simulation tick.

---

# 33. TDD checkpoint A — LabState construction and derived energy

Write tests first.

Required tests:

```text
TestNewLabState_AcceptsZero
TestNewLabState_AcceptsMaximum
TestNewLabState_RejectsBelowZero
TestNewLabState_RejectsAboveMaximum
TestNewTutorialLabState_StartsAtMaximum
TestNewTutorialLabState_UsesProvidedTimestamp
TestNewTutorialLabState_HasNoOverride

TestLabState_CurrentEnergyAtBaseline
TestLabState_CurrentEnergyUsesCompletedWholeSeconds
TestLabState_CurrentEnergyAtExactSecond
TestLabState_CurrentEnergyUsesConfiguredRate
TestLabState_CurrentEnergyCapsAtMaximum
TestLabState_CurrentEnergyBeforeBaselineDoesNotRegenerate
TestLabState_CurrentEnergyDoesNotMutateBaseline
TestLabState_CurrentEnergyIsInteger
```

Expected RED:

```text
undefined NewLabState / NewTutorialLabState / CurrentEnergy
```

Minimal GREEN:

- constructor bounds;
- Tutorial maximum baseline;
- completed-whole-second derivation;
- cap at configured maximum;
- no mutation.

Traceability target:

```text
LAB-001 GREEN
LAB-002 PARTIAL  # domain constructor; full Tutorial bootstrap Stage 14
LAB-003 GREEN
LAB-004 GREEN
```

Recommended commits:

```text
test(stage5): RED checkpoint A lab energy derivation
feat(stage5): GREEN checkpoint A lab energy derivation
```

---

# 34. TDD checkpoint B — spending, insufficiency and re-baseline

Required tests:

```text
TestLabState_SpendEnergySubtractsCost
TestLabState_SpendEnergyUsesRegeneratedCurrentValue
TestLabState_SpendEnergyRebasesAtNow
TestLabState_SpendEnergyPreventsDoubleCountingRegeneration
TestLabState_SpendEnergyAllowsExactBalance
TestLabState_SpendEnergyRejectsInsufficientBalance
TestLabState_SpendEnergyInsufficientIsAtomic
TestLabState_SpendEnergyZeroDoesNotRebase
TestLabState_SpendEnergyRejectsNegativeCost
TestLabState_SpendEnergyRejectsMalformedBaseline
TestLabState_SpendEnergyRejectsBackwardTime
TestLabState_CanAffordUsesDerivedEnergy
```

Expected RED demonstrates missing debit/affordability behavior.

Minimal GREEN:

- `ErrLabEnergyInvariant`;
- `ErrInsufficientLabEnergy`;
- derived affordability;
- positive-cost re-baseline;
- zero-cost no-op;
- atomic rejection.

Traceability target:

```text
LAB-001 remains GREEN
LAB-003/004 remain GREEN
LAB-010 PARTIAL  # primitive proven; action integrations follow
```

Recommended commits:

```text
test(stage5): RED checkpoint B spending and insufficiency
feat(stage5): GREEN checkpoint B spending and insufficiency
```

---

# 35. TDD checkpoint C — zero-cost SEND wrapper

Required tests:

```text
TestSendObserverWithLabEnergy_AllowsZeroEnergy
TestSendObserverWithLabEnergy_CostsZero
TestSendObserverWithLabEnergy_DoesNotRebaseEnergy
TestSendObserverWithLabEnergy_DelegatesStage4Selection
TestSendObserverWithLabEnergy_DelegatesStage4Flow
TestSendObserverWithLabEnergy_PreservesUnstableConfirmation
TestSendObserverWithLabEnergy_RejectionDoesNotChangeLab
TestSendObserverWithLabEnergy_RejectionDoesNotConsumeRandom
TestSendObserverWithLabEnergy_SuccessDrawsTransitOnce
TestSendObserverWithLabEnergy_RejectsInvalidLabStateBeforeMutation
```

Expected RED:

```text
undefined SendObserverWithLabEnergy
```

Minimal GREEN:

- validate LabState;
- fixed zero cost from the Final Spec balance contract;
- delegate `SendObserver` exactly once;
- never call positive debit/re-baseline path.

Traceability target:

```text
LAB-005 GREEN
FLOW-002/005/006 remain GREEN with wrapper coverage added
```

Recommended commits:

```text
test(stage5): RED checkpoint C zero-cost SEND
feat(stage5): GREEN checkpoint C zero-cost SEND
```

---

# 36. TDD checkpoint D — zero-cost RECALL wrapper

Required tests:

```text
TestRecallObserverWithLabEnergy_AllowsZeroEnergy
TestRecallObserverWithLabEnergy_CostsZero
TestRecallObserverWithLabEnergy_DoesNotRebaseEnergy
TestRecallObserverWithLabEnergy_DelegatesLongestWaitingSelection
TestRecallObserverWithLabEnergy_DelegatesStage4Flow
TestRecallObserverWithLabEnergy_PreservesUnstableConfirmation
TestRecallObserverWithLabEnergy_RejectionDoesNotChangeLab
TestRecallObserverWithLabEnergy_RejectionDoesNotConsumeRandom
TestRecallObserverWithLabEnergy_SuccessDrawsTransitOnce
TestRecallObserverWithLabEnergy_RejectsInvalidLabStateBeforeMutation
```

Minimal GREEN mirrors checkpoint C but calls `RecallObserver`.

Traceability target:

```text
LAB-006 GREEN
OBSERVER-016 and FLOW-003/004/006 remain GREEN with wrapper coverage added
```

Recommended commits:

```text
test(stage5): RED checkpoint D zero-cost RECALL
feat(stage5): GREEN checkpoint D zero-cost RECALL
```

---

# 37. TDD checkpoint E — paid manual Close

Required tests:

```text
TestClosePortalWithLabEnergy_ChargesFive
TestClosePortalWithLabEnergy_AllowsExactFive
TestClosePortalWithLabEnergy_UsesRegeneratedEnergy
TestClosePortalWithLabEnergy_RejectsFour
TestClosePortalWithLabEnergy_InsufficientLeavesPortalUnchanged
TestClosePortalWithLabEnergy_InsufficientLeavesObserverUnchanged
TestClosePortalWithLabEnergy_ConfirmationRequiredDoesNotDebit
TestClosePortalWithLabEnergy_ConfirmedTransitClosesAndLosesObserver
TestClosePortalWithLabEnergy_ConfirmedTransitDebitsOnce
TestClosePortalWithLabEnergy_TerminalPortalDoesNotDebit
TestClosePortalWithLabEnergy_StructuralErrorDoesNotDebit
TestClosePortalWithLabEnergy_RejectsInvalidLabStateBeforeClose
```

Expected RED:

```text
undefined ClosePortalWithLabEnergy
```

Minimal GREEN:

- precheck `cfg.CloseCost` affordability;
- delegate `ClosePortalWithObservers`;
- debit once after success;
- no Stage 6 override branch.

Traceability target:

```text
LAB-007 GREEN
LAB-010 remains PARTIAL
PORTAL-006 stays GREEN with cost integration evidence
OBSERVER-012 stays GREEN
```

Recommended commits:

```text
test(stage5): RED checkpoint E paid manual close
feat(stage5): GREEN checkpoint E paid manual close
```

---

# 38. TDD checkpoint F — paid Stabilize

Required tests:

```text
TestStabilizePortalWithLabEnergy_ChargesTwenty
TestStabilizePortalWithLabEnergy_AllowsExactTwenty
TestStabilizePortalWithLabEnergy_UsesRegeneratedEnergy
TestStabilizePortalWithLabEnergy_RejectsNineteen
TestStabilizePortalWithLabEnergy_InsufficientLeavesPortalUnchanged
TestStabilizePortalWithLabEnergy_SuccessPreservesPortalSemantics
TestStabilizePortalWithLabEnergy_StablePortalDoesNotDebit
TestStabilizePortalWithLabEnergy_OverchargeRejectionDoesNotDebit
TestStabilizePortalWithLabEnergy_TerminalPortalDoesNotDebit
TestStabilizePortalWithLabEnergy_DebitsExactlyOnce
TestStabilizePortalWithLabEnergy_RejectsInvalidLabStateBeforeMutation
```

`SuccessPreservesPortalSemantics` must assert:

```text
UNSTABLE → STABLE
hidden collapse timestamp cleared
Portal current Energy +15 and re-baselined at now
Portal decay unchanged
Lab current Energy -20 and re-baselined at now
```

Minimal GREEN delegates `Portal.Stabilize` and commits a single Lab debit.

Traceability target:

```text
LAB-008 GREEN
LAB-010 remains PARTIAL
STABILITY-005/006 stay GREEN with cost integration evidence
ENERGY-006..008 stay GREEN
```

Recommended commits:

```text
test(stage5): RED checkpoint F paid stabilize
feat(stage5): GREEN checkpoint F paid stabilize
```

---

# 39. TDD checkpoint G — Extraction cost contract only

Required tests:

```text
TestLabState_ExtractionCostIsThirty
TestLabState_SpendExtractionCostAllowsExactThirty
TestLabState_SpendExtractionCostRejectsTwentyNine
TestLabState_SpendExtractionCostRebasesToZero
TestLabState_SpendExtractionCostUsesRegeneratedEnergy
TestLabState_SpendExtractionCostDoesNotCreatePortal
```

The config balance assertion may already be GREEN from Stage 0. Record it as a
characterization rather than manufacturing RED. Generic spend behavior may
also already be GREEN from checkpoint B.

No `OpenExtractionPortal` symbol may be added.

Traceability target:

```text
LAB-009 PARTIAL
EXTRACTION-001 remains PLANNED
```

Recommended commit:

```text
test(stage5): GREEN checkpoint G characterize extraction cost contract
```

---

# 40. TDD checkpoint H — transaction integration and regressions

Required tests:

```text
TestLabEnergyCommands_InsufficientPaidActionDoesNotCallDomainMutation
TestLabEnergyCommands_DomainFailureDoesNotDebit
TestLabEnergyCommands_ConfirmationFailureDoesNotDebit
TestLabEnergyCommands_ZeroCostActionsPreserveBaselineAtCap
TestLabEnergyCommands_PaidActionSpendsRegeneratedEnergyOnce
TestLabEnergyCommands_LabAndPortalEnergyRemainIndependent
TestLabEnergyCommands_DoNotMutateOverrideDeadline
TestLabEnergyCommands_DoNotChangeStage4ErrorIdentity
TestLabEnergyCommands_DoNotChangeStage4RandomConsumption
TestLabEnergyCommands_RepeatedReadDoesNotWriteBaseline
```

These may be already-GREEN characterizations over checkpoints A–F. Document
that honestly. If a real atomicity gap appears, create a RED commit before its
minimal fix.

Traceability target:

```text
LAB-001..008 current Stage 5 boundary reviewed
LAB-009/010 remain honestly PARTIAL
Stage 2/4 rows retain GREEN
```

Recommended commit when no production correction is required:

```text
test(stage5): GREEN checkpoint H transaction characterizations
```

---

# 41. TDD execution protocol

For every checkpoint:

```text
1. Map requirement IDs and exact Final Spec sections.
2. Write focused tests before production code.
3. Run targeted tests and observe genuine RED.
4. Commit RED tests.
5. Implement the smallest behavior that satisfies the checkpoint.
6. Run targeted tests and observe GREEN.
7. Run the complete test suite.
8. Run the race suite.
9. Refactor only while GREEN.
10. Update docs/traceability.md honestly.
11. Commit GREEN implementation.
```

Do not fake RED for checkpoint G or H characterizations that already pass.

---

# 42. Required verification commands

At every meaningful GREEN checkpoint and at final Stage 5 completion:

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
grep -Rni 'time.Sleep\|time.Now()\|time.NewTicker\|time.After' internal/domain/lab*.go
grep -Rni 'database/sql\|net/http\|sync\.\|OpenExtraction\|ACTION_REJECTED' internal/domain/lab*.go
grep -Rni 'ResolveLifecycle' internal/domain/lab_energy_commands.go
```

Expected output for all three scope searches: no matches.

---

# 43. Traceability update expected after Stage 5

Expected statuses:

```text
LAB-001  GREEN
LAB-002  PARTIAL  # constructor proven; application Tutorial bootstrap later
LAB-003  GREEN
LAB-004  GREEN
LAB-005  GREEN
LAB-006  GREEN
LAB-007  GREEN
LAB-008  GREEN
LAB-009  PARTIAL  # cost/debit proven; real Extraction command Stage 7
LAB-010  PARTIAL  # ordinary actions proven; override/Extraction paths later

PORTAL-006      remains GREEN, add Close cost wrapper tests
STABILITY-005   remains GREEN, add Stabilize cost wrapper tests
ENERGY-006..008 remain GREEN, add cross-resource tests

EMERGENCY-*     remain PLANNED
EXTRACTION-*    remain PLANNED
```

Every GREEN/PARTIAL row must list concrete test names, implementation symbols,
and the exact remaining boundary.

`docs/requirements.md` remains semantically unchanged unless a genuine Final
Spec conflict is discovered.

---

# 44. Worklog update expected

Append to `01_AI_WORKLOG_CURRENT.md`:

- Stage 5 execution agent/model;
- baseline verification result;
- every RED/GREEN commit pair;
- integer whole-second derivation decision;
- zero-cost no-rebase behavior;
- ordinary paid-action transaction order;
- any atomicity bug found by tests;
- extraction/override boundaries and honest PARTIAL statuses;
- final verification results;
- explicit statement that Stage 6 was not started.

Do not claim a command passed until its output was observed.

---

# 45. Commit strategy

Recommended history:

```text
test(stage5): RED checkpoint A lab energy derivation
feat(stage5): GREEN checkpoint A lab energy derivation
test(stage5): RED checkpoint B spending and insufficiency
feat(stage5): GREEN checkpoint B spending and insufficiency
test(stage5): RED checkpoint C zero-cost SEND
feat(stage5): GREEN checkpoint C zero-cost SEND
test(stage5): RED checkpoint D zero-cost RECALL
feat(stage5): GREEN checkpoint D zero-cost RECALL
test(stage5): RED checkpoint E paid manual close
feat(stage5): GREEN checkpoint E paid manual close
test(stage5): RED checkpoint F paid stabilize
feat(stage5): GREEN checkpoint F paid stabilize
test(stage5): GREEN checkpoint G characterize extraction cost contract
test(stage5): GREEN checkpoint H transaction characterizations
docs(stage5): record TDD evidence and verification
```

If a characterization reveals missing behavior, split it into a real RED/GREEN
pair and record the observed failure in the worklog.

---

# 46. Stage 5 acceptance checklist

Stage 5 is complete only if all are true:

- [ ] Lab Energy is represented only as integers.
- [ ] Valid stored baseline is bounded to `0..cfg.LabEnergyMax`.
- [ ] Tutorial LabState constructor starts at maximum/default 100.
- [ ] Derived energy uses completed whole seconds.
- [ ] Derived energy uses configured regeneration rate.
- [ ] Derived energy caps at maximum.
- [ ] Reads do not mutate baseline state.
- [ ] Reads before baseline time do not regenerate.
- [ ] Positive spend uses current derived energy.
- [ ] Positive spend re-baselines at `now`.
- [ ] Re-baseline prevents double-counted regeneration.
- [ ] Exact affordability succeeds.
- [ ] Insufficient affordability rejects atomically.
- [ ] Negative costs reject as invariant errors.
- [ ] Backward-time mutations reject atomically.
- [ ] Zero-cost spend does not re-baseline.
- [ ] SEND succeeds at zero Lab Energy when Stage 4 permits it.
- [ ] RECALL succeeds at zero Lab Energy when Stage 4 permits it.
- [ ] SEND/RECALL success leaves both Lab baseline fields unchanged.
- [ ] SEND/RECALL preserve Stage 4 restrictions, confirmation and random use.
- [ ] Ordinary Close costs exactly `cfg.CloseCost`/5.
- [ ] Close exact balance succeeds and reaches zero.
- [ ] Insufficient Close leaves Lab/Portal/Plane/Observers unchanged.
- [ ] Confirmation-required Close does not debit.
- [ ] Confirmed active-transit Close debits once and preserves LOST behavior.
- [ ] Ordinary Stabilize costs exactly `cfg.StabilizeCost`/20.
- [ ] Stabilize exact balance succeeds and reaches zero.
- [ ] Insufficient Stabilize leaves Lab/Portal unchanged.
- [ ] Failed Stabilize does not debit.
- [ ] Successful Stabilize preserves Stage 2 Portal-energy semantics.
- [ ] Lab Energy and Portal Energy remain independent resources.
- [ ] Extraction cost 30 is covered only through generic cost/debit contract.
- [ ] No Extraction Portal or synchronization behavior was implemented.
- [ ] `LeylineOverrideUntil` is neither read nor mutated by Stage 5 commands.
- [ ] No Collapse→Lab-zero behavior was implemented.
- [ ] No Events, persistence, HTTP, WebSocket, simulation or LabManager added.
- [ ] Completed Stage 2–4 semantics were not changed without demonstrated RED.
- [ ] `gofmt -l .` prints nothing.
- [ ] `go vet ./...` passes.
- [ ] `go build ./...` passes.
- [ ] `go test -count=1 ./...` passes.
- [ ] `go test -race -count=1 ./...` passes.
- [ ] Traceability is updated honestly.
- [ ] Worklog is updated honestly.
- [ ] Stage 6 was not started.

---

# 47. Explicit non-goals

Do not implement in Stage 5:

```text
Leyline Override activation, expiry or free costs
Collapse interaction with Lab Energy
Extraction Portal creation, Plane selection or synchronization
automatic Extraction return
natural spawn scheduling
simulation tick or goroutine
event emission or persistence
SQLite schemas/repositories
REST endpoints or confirmation payloads
WebSocket snapshots
frontend energy display
Tutorial controller/step progression
LabManager or mutexes
```

Tests requiring these behaviors belong to later detailed plans.

---

# 48. Resolved Stage 5 planning decisions

No blocking Final Spec contradiction is known.

## S5-D1 — integer regeneration

Only completed whole seconds regenerate energy. Fractional seconds do not
produce fractional or rounded-up energy.

## S5-D2 — zero-cost actions

SEND/RECALL validate LabState but do not re-baseline it. Cost 0 must be a true
state no-op.

## S5-D3 — paid action order

Lab invariant and affordability checks happen before delegated mutation.
Delegated domain errors happen before the one successful debit.

## S5-D4 — ordinary costs only

Close and Stabilize always use configured ordinary cost in Stage 5. Override
cost selection is exclusively Stage 6.

## S5-D5 — Extraction boundary

Stage 5 proves the reusable 30-point spend contract without creating any
Extraction aggregate or command.

## S5-D6 — backward time

Derived reads clamp negative elapsed to zero; mutations before the baseline
timestamp reject as invariant violations.

## S5-D7 — manager boundary

Wrappers are sequential pure-domain orchestration. Stage 11 later owns locking
and resolve-first arbitration without moving energy into Portal/Observer.

---

# 49. Stop condition

After every Stage 5 acceptance item is verified:

```text
STOP.
```

Do not begin Stage 6 in the same implementation pass.

Final Stage 5 report must include:

- files changed;
- tests added;
- RED/GREEN commit evidence;
- characterization tests that were already GREEN;
- integer derivation and transaction-order decisions used;
- verification command results;
- traceability changes and remaining PARTIAL rows;
- commit hashes;
- explicit confirmation that Stage 6 was not started.

---

# 50. Next stage after completion

Stage 6:

```text
Emergency / Leyline Override via TDD

every COLLAPSED → Lab Energy 0
override active for 20 sec
Close/Stabilize cost 0 during override
all other restrictions remain
Extraction stays 30
Lab Energy regeneration continues
another Collapse resets energy and deadline
```

Stage 6 must extend the Stage 5 cost boundary without changing ordinary
no-override energy semantics.
