# Omenpath Research Lab — Traceability Matrix

> `requirement → test → implementation`. Source: [`docs/requirements.md`](requirements.md) ← `00_FINAL_SPEC_v5.md`.
>
> Columns: **ID | Rule | Type | Test | Implementation | Status | Notes**
>
> Status: `PLANNED → RED → GREEN → REFACTORED`; `PARTIAL` — правило покрыто частично (чистый helper / подмножество поведения), полная оркестрация или остаток поведения — на будущей стадии (введён corrective-пасом после аудита Stage 0–2).
>
> Infrastructure (clock / random / builders / config) не имеет ID; покрытие см. внизу.

## PLANE

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| PLANE-001 | Plane permanent, independent of Portal | INV | — | — | PLANNED | |
| PLANE-002 | Plane 0..N Portals | INV | — | — | PLANNED | |
| PLANE-003 | Multiple OPEN Portals to same Plane | BEH | — | — | PLANNED | |
| PLANE-004 | Seed 85, default UNEXPLORED | BEH | — | — | PLANNED | |
| PLANE-005 | SEND does not explore | INV | `TestObserver_StartOutboundDoesNotExplorePlane`, `TestSendObserver_DoesNotExplorePlane`, `TestSendObserver_AllowsAlreadyExploredPlane` | `Observer.StartOutbound`, `SendObserver` | GREEN | real SEND command leaves both unexplored and already-explored Plane state unchanged |
| PLANE-006 | Arrival does not explore | INV | `TestObserver_OutboundArrivalDoesNotExplorePlane` | `ResolveObserverLifecycle` | GREEN | successful OUTBOUND leaves Plane exploration unchanged |
| PLANE-007 | Research completion does not explore | INV | `TestObserver_ResearchCompletionDoesNotExplorePlane` | `ResolveObserverLifecycle` | GREEN | WAITING_RETURN retains Plane location but not exploration |
| PLANE-008 | Successful return explores | BEH | `TestObserver_SuccessfulReturnExploresPlane`, `TestObserver_SuccessfulReturnSetsExploredAtToArrival` | `ResolveObserverLifecycle` | GREEN | explored at semantic return deadline |
| PLANE-009 | Re-return idempotent | INV | `TestObserver_ReturnToAlreadyExploredPlanePreservesOriginalExploredAt`, `TestObservers_MultipleObserversMayExploreSamePlaneConcurrently` | `ResolveObserverLifecycle` | PARTIAL | Plane-state idempotence (`Explored`/`ExploredAt`) доказана; progress aggregation — later stage |

## PORTAL

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| PORTAL-001 | Unique instance per opening | INV | — | — | PLANNED | |
| PORTAL-002 | Sequential `Omenpath #XXXX` | BEH | `TestNewNaturalPortal_GeneratesUnstableWithinSpec` | `NewNaturalPortal` | PARTIAL | формат имени из seq покрыт; сквозная нумерация экземпляров — оркестрация Stage 8/11 |
| PORTAL-003 | NATURAL/EXTRACTION kinds | BEH | `TestNewNaturalPortal_GeneratesUnstableWithinSpec` | `PortalKind` enum + factory | GREEN | создание EXTRACTION-портала — Stage 7 |
| PORTAL-004 | OPEN/CLOSED/COLLAPSED statuses | INV | `TestPortal_ScheduledRemainingIsZeroWhenTerminal`, `TestPortal_EnergyFreezesAtManualClose`, `TestPortal_EnergyFreezesAtCollapse`, `TestPortal_CreaturesFreezeAtCollapse` | `PortalStatus` enum + terminal derived state freeze | GREEN | все три статуса и разные outcomes (CLOSED vs COLLAPSED, разные termination reasons) покрыты прямо; повышено с PLANNED в corrective-пасе |
| PORTAL-005 | TTL expiry → CLOSED/NATURAL_CLOSE | BEH | `TestPortal_NaturalCloseWhenTTLExpires`, `TestPortal_LateResolutionPreservesNaturalCloseTime` | `Portal.ResolveLifecycle` | GREEN | semantic ClosedAt; late resolution сохраняет время события |
| PORTAL-006 | Manual Close → CLOSED/MANUAL_CLOSE | BEH | `TestPortal_ManualCloseWithoutCreatures`, `TestPortal_ManualCloseRejectsTerminalPortal`, `TestPortal_ManuallyClosedPortalIsTerminalForLifecycle` | `Portal.Close` | GREEN | cost 5 — оркестрация (LAB-007); observer-transit подтверждение — Stage 3/4 |
| PORTAL-007 | Energy 0 → COLLAPSED/ENERGY_DEPLETED | BEH | `TestPortal_CollapsesWhenEnergyReachesZeroBeforeNaturalClose`, `TestPortal_NaturalCloseWinsEnergyTie`, `TestPortal_NaturalCloseBeforeEnergyDepletion` | `Portal.ResolveLifecycle` | GREEN | tie → NATURAL_CLOSE (Stage 2 plan Rule B) |
| PORTAL-008 | Hidden instability → COLLAPSED/INSTABILITY | BEH | `TestPortal_UnstableCollapsesAtHiddenTime`, `TestPortal_NaturalCloseWinsBeforeHiddenCollapse`, `TestPortal_NaturalCloseTiesHiddenCollapse`, `TestPortal_EnergyDepletionWinsInstabilityTie` | `Portal.ResolveLifecycle` | GREEN | Rule C: tie с NATURAL_CLOSE → NATURAL_CLOSE; exact tie с ENERGY_DEPLETED (оба раньше close) → ENERGY_DEPLETED (Rule D, regression) |
| PORTAL-009 | Terminal cannot return OPEN | INV | `TestPortal_TerminalStateCannotReopen`, `TestPortal_ScheduledRemainingIsZeroWhenTerminal`, `TestPortal_EnergyFreezesAtManualClose`, `TestPortal_EnergyFreezesAtCollapse`, `TestPortal_CreaturesFreezeAtManualClose`, `TestPortal_CreaturesFreezeAtCollapse` | `Portal.IsTerminal` + `ResolveLifecycle` guard + derived state freeze | GREEN | snapshot-equality — нет мутаций; расширен аудитом: derived state замораживается на ClosedAt (ScheduledRemaining=0, Energy/Creatures frozen), risk отсутствует |

## SLOT

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| SLOT-001 | Exactly 7 slots | INV | `TestFirstFreeSlot_ReturnsNoneWhenAllSevenOpen` | `FirstFreeSlot` + `cfg.MaxActivePortals` | PARTIAL | domain/config часть (bound 7) покрыта; «Dashboard всегда содержит 7 фиксированных Slots» — фронтенд Stage 15/16 |
| SLOT-002 | OPEN Portal occupies a slot | BEH | `TestFirstFreeSlot_ReturnsNextAfterOccupiedPrefix` | `FirstFreeSlot` | GREEN | |
| SLOT-003 | First free slot | BEH | `TestFirstFreeSlot_FillsGap` | `FirstFreeSlot` | GREEN | |
| SLOT-004 | Portal keeps slot for lifecycle | INV | `TestFirstFreeSlot_DoesNotMutateInput` | pure helper | PARTIAL | helper не мутирует вход / не пересортирует; инвариант целиком (хранение slot_index у живого портала) — LabManager Stage 11 |
| SLOT-005 | Terminal releases slot | BEH | `TestFirstFreeSlot_TerminalPortalsDoNotOccupy` | `FirstFreeSlot` | GREEN | |
| SLOT-006 | 7/7 blocks natural spawn | BEH | `TestFirstFreeSlot_ReturnsNoneWhenAllSevenOpen` | `FirstFreeSlot` | PARTIAL | helper детектирует 7/7 (нет слота); поведение генератора «ждёт и возобновляется» — Stage 8 |
| SLOT-007 | Extraction uses regular slot | BEH | — | — | PLANNED | Extraction-создание — Stage 7; helper kind-agnostic |

## ENERGY

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| ENERGY-001 | Initial natural 10..100 | BAL | `TestNewNaturalPortal_GeneratesUnstableWithinSpec` | `NewNaturalPortal` | GREEN | draw из cfg-диапазона |
| ENERGY-002 | Decay 0.1..1.0/sec hidden | BAL | `TestNewNaturalPortal_GeneratesUnstableWithinSpec` | `NewNaturalPortal` | PARTIAL | диапазон 0.1..1.0 покрыт фабрикой; скрытость от пользователя (не отдаётся API/UI) — Stage 12/13 |
| ENERGY-003 | Current derived from baseline/time | BEH | `TestPortal_CurrentEnergy`, `TestPortal_EnergyDepletionAt` | `Portal.CurrentEnergy`, `Portal.EnergyDepletionAt` | GREEN | |
| ENERGY-004 | Clamp ≥ 0 | INV | `TestPortal_CurrentEnergy/clamped_at_zero` | `Portal.CurrentEnergy` | GREEN | |
| ENERGY-005 | 0 before close → COLLAPSED/ENERGY_DEPLETED | BEH | `TestPortal_CollapsesWhenEnergyReachesZeroBeforeNaturalClose` | `Portal.EnergyDepletionAt` | GREEN | Stage 0–1 first RED cycle |
| ENERGY-006 | Stabilize +15 | BEH | `TestPortal_StabilizeConvertsUnstableToStable` | `Portal.Stabilize` | GREEN | |
| ENERGY-007 | Stabilize re-baselines | BEH | `TestPortal_StabilizeConvertsUnstableToStable`, `TestPortal_StabilizeUsesCurrentEnergyNotBaseline` | `Portal.Stabilize` | GREEN | stale-baseline регресс покрыт |
| ENERGY-008 | Decay unchanged after Stabilize | INV | `TestPortal_StabilizeConvertsUnstableToStable` | `Portal.Stabilize` | GREEN | |
| ENERGY-009 | >85 rejects Stabilize | BEH | `TestPortal_StabilizeRejectsEnergyAbove85` | `Portal.Stabilize` | GREEN | проверка по current, не baseline |
| ENERGY-010 | Exactly 85 → 100 | BEH | `TestPortal_StabilizeAllowsExactly85Percent` | `Portal.Stabilize` | GREEN | |

## STABILITY

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| STABILITY-001 | STABLE/UNSTABLE only | INV | `TestPortal_UnstableCollapsesAtHiddenTime`, `TestNewNaturalPortal_GeneratesUnstableWithinSpec`, `TestNewNaturalPortal_StableHasNoHiddenTimestamp` | `PortalStability` enum | GREEN | обе стабильности генерируются factory |
| STABILITY-002 | Stable has no hidden timer | INV | `TestNewNaturalPortal_StableHasNoHiddenTimestamp`, `TestPortalBuilder_Defaults` | factory / builder / `Stabilize` | GREEN | |
| STABILITY-003 | Unstable gets hidden timer | BEH | `TestNewNaturalPortal_GeneratesUnstableWithinSpec`, `TestNewNaturalPortal_HiddenCollapseWithinWindow` | `NewNaturalPortal` | GREEN | random(opened+5s, close−1s) |
| STABILITY-004 | Hidden timer not exposed / not in risk | UI | `TestPortal_HiddenInstabilityTimestampDoesNotAffectRisk` | `Portal.RiskScore` | PARTIAL | доказана только domain-часть: hidden не участвует в Risk; «не exposed пользователю» (API/UI не отдаёт timestamp) — Stage 12/13 |
| STABILITY-005 | Stabilize unstable→stable | BEH | `TestPortal_StabilizeConvertsUnstableToStable` | `Portal.Stabilize` | GREEN | |
| STABILITY-006 | Stabilize clears hidden timer | BEH | `TestPortal_StabilizeConvertsUnstableToStable`, `TestPortal_StabilizedPortalLosesInstabilityCandidate` | `Portal.Stabilize` | GREEN | |
| STABILITY-007 | Stable cannot be stabilized | BEH | `TestPortal_StabilizeRejectsStablePortal` | `Portal.Stabilize` | GREEN | state unchanged при отказе |

## CREATURE

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| CREATURE-001 | Natural 0..10 | BAL | `TestNewNaturalPortal_CreaturesRespectMaxForTTL` | `NewNaturalPortal` + `MaxCreaturesForTTL` | GREEN | draw IntInclusive(0, max≤10) |
| CREATURE-002 | Passage 2 sec | BAL | `TestPortal_CreaturesInsideDecreasesEveryTwoSeconds` | `Portal.CreaturesInside` | GREEN | из cfg.CreatureTransit |
| CREATURE-003 | Margin 2 sec | BAL | `TestMaxCreaturesForTTL` | `MaxCreaturesForTTL` | GREEN | из cfg.CreatureClearanceMargin |
| CREATURE-004 | Max formula | BEH | `TestMaxCreaturesForTTL` | `MaxCreaturesForTTL` | GREEN | |
| CREATURE-005 | TTL10 → max 4 | BEH | `TestMaxCreaturesForTTL/TTL_10s_allows_four` | `MaxCreaturesForTTL` | GREEN | |
| CREATURE-006 | Current count derived | BEH | `TestPortal_CreaturesInsideDecreasesEveryTwoSeconds` | `Portal.CreaturesInside` | GREEN | таблица из плана §15 |
| CREATURE-007 | Creatures block SEND/RECALL | BEH | `TestSendObserver_RejectsCreaturesInside`, `TestRecallObserver_RejectsCreaturesInside` | `SendObserver`, `RecallObserver` | GREEN | both commands use current derived creature count before busy/eligibility/warning checks |
| CREATURE-008 | Close requires confirmation | UI | `TestPortal_ManualCloseRequiresConfirmationWithCreatures`, `TestPortal_ManualCloseWithConfirmationClosesDespiteCreatures` | `Portal.Close` | GREEN | domain-часть; modal — фронтенд |

## RISK

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| RISK-001 | energy_lifetime formula | BEH | `TestPortal_EnergyLifetime` | `Portal.EnergyLifetime` | GREEN | |
| RISK-002 | effective_lifetime = min | BEH | `TestPortal_EffectiveLifetime` | `Portal.EffectiveLifetime` | GREEN | |
| RISK-003 | base risk formula (45s) | BEH | `TestPortal_RiskScore` | `Portal.RiskScore` | GREEN | границы 0/25/50/75 покрыты |
| RISK-004 | UNSTABLE +20 | BEH | `TestPortal_UnstableAddsRiskPenalty` | `Portal.RiskScore` | GREEN | MEDIUM→HIGH сдвиг по плану §17.5 |
| RISK-005 | max 100 | INV | `TestPortal_RiskScore` (cap-кейсы) | `Portal.RiskScore` | GREEN | |
| RISK-006 | 0..25 LOW | BEH | `TestPortal_RiskLevelBoundaries` | `Portal.RiskLevel` | GREEN | score==25 → LOW |
| RISK-007 | >25..50 MEDIUM | BEH | `TestPortal_RiskLevelBoundaries` | `Portal.RiskLevel` | GREEN | score==50 → MEDIUM |
| RISK-008 | >50..75 HIGH | BEH | `TestPortal_RiskLevelBoundaries` | `Portal.RiskLevel` | GREEN | score==75 → HIGH |
| RISK-009 | >75..100 CRITICAL | BEH | `TestPortal_RiskLevelBoundaries`, `TestPortal_RiskAgreedExamples` | `Portal.RiskLevel` | GREEN | agreed 14s HIGH / 10s CRITICAL |
| RISK-010 | Hidden timer not used | INV | `TestPortal_HiddenInstabilityTimestampDoesNotAffectRisk` | `Portal.RiskScore` | GREEN | одинаковый Risk при разных hidden |
| RISK-011 | UI gets level, not score | UI | — | — | PLANNED | |
| RISK-012 | CRITICAL blocks SEND | BEH | `TestSendObserver_RejectsCriticalRisk`, `TestSendObserver_UsesDocumentedErrorPrecedence` | `SendObserver` | GREEN | current derived risk checked before direction/creatures/busy/eligibility/warning |
| RISK-013 | CRITICAL blocks RECALL | BEH | `TestRecallObserver_RejectsCriticalRisk`, `TestRecallObserver_UsesDocumentedErrorPrecedence` | `RecallObserver` | GREEN | current derived risk checked before direction/creatures/busy/eligibility/warning |

## LAB

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| LAB-001 | Integer 0..100 | INV | — | — | PLANNED | |
| LAB-002 | Tutorial starts 100 | BEH | — | — | PLANNED | |
| LAB-003 | Regen +1/sec | BEH | — | — | PLANNED | |
| LAB-004 | Cap 100 | BEH | — | — | PLANNED | |
| LAB-005 | SEND 0 | BAL | — | — | PLANNED | |
| LAB-006 | RECALL 0 | BAL | — | — | PLANNED | |
| LAB-007 | CLOSE 5 | BAL | — | — | PLANNED | |
| LAB-008 | STABILIZE 20 | BAL | — | — | PLANNED | |
| LAB-009 | EXTRACTION 30 | BAL | — | — | PLANNED | |
| LAB-010 | Insufficient energy rejects | BEH | — | — | PLANNED | |

## OBSERVER

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| OBSERVER-001 | Exactly 10 | INV | `TestNewObserverRoster_CreatesConfiguredCount`, `TestNewObserverRoster_DefaultConfigCreatesTen` | `NewObserverRoster` + `cfg.ObserverCount` | PARTIAL | roster/count=10 доказан; permanence/persistence bootstrap — Stage 10/11 |
| OBSERVER-002 | Initial AVAILABLE | BEH | `TestNewObserver_StartsAvailableInLaboratory` | `NewObserver` | GREEN | |
| OBSERVER-003 | AVAILABLE = Lab | INV | `TestObserver_AvailableCanonicalFields` | `NewObserver` | GREEN | location/transit/phase fields canonical nil |
| OBSERVER-004 | Main lifecycle | BEH | `TestNewObserver_StartsAvailableInLaboratory`, `TestObserver_StartOutboundTransitionsFromAvailable`, `TestObserver_OutboundAtDeadlineBecomesExploring`, `TestObserver_ResearchCompletionBecomesWaitingReturn`, `TestObserver_StartReturningFromWaitingReturn`, `TestObserver_ReturningAtDeadlineBecomesAvailable` | `NewObserver`, `Observer.StartOutbound`, `Observer.StartReturning`, `ResolveObserverLifecycle` | GREEN | complete main lifecycle; failure path covered separately |
| OBSERVER-005 | LOST terminal | INV | `TestObserver_LostIsTerminal`, `TestObserver_LostCannotBeRevivedByResolve`, `TestObserver_LostCannotStartOutbound`, `TestObserver_LostCannotStartReturning` | `Observer.IsTerminal`, `Observer.StartOutbound`, `Observer.StartReturning`, `ResolveObserverLifecycle` | GREEN | LOST cannot transition or revive |
| OBSERVER-006 | Transit 5..15 | BAL | `TestObserver_StartOutboundTransitMinimumFiveSeconds`, `TestObserver_StartOutboundTransitMaximumFifteenSeconds`, `TestObserver_StartReturningDrawsFreshTransitDuration` | `Observer.StartOutbound`, `Observer.StartReturning` | GREEN | each direction draws within configured inclusive range |
| OBSERVER-007 | Duration fixed once | BEH | `TestObserver_StartOutboundDrawsTransitDurationExactlyOnce`, `TestObserver_StartReturningDrawsTransitDurationExactlyOnce`, `TestObserver_ReturnResolveDoesNotConsumeRandom` | `Observer.StartOutbound`, `Observer.StartReturning`, `ResolveObserverLifecycle` | GREEN | one draw per transit start; resolver has no random dependency |
| OBSERVER-008 | Outbound → EXPLORING | BEH | `TestObserver_OutboundBeforeDeadlineRemainsOutbound`, `TestObserver_OutboundAtDeadlineBecomesExploring`, `TestObserver_OutboundArrivalUsesDeadlineAsTransitionTime`, `TestObserver_OutboundArrivalSetsCurrentPlane`, `TestObserver_OutboundArrivalClearsActivePortal`, `TestObserver_OutboundResolveDoesNotConsumeRandom`, `TestObserver_OutboundSucceedsWhenPortalClosesExactlyAtTransitEnd`, `TestObserver_PortalClosingAfterOutboundEndDoesNotRetroactivelyLoseObserver` | `ResolveObserverLifecycle` | GREEN | destination/phase invariants reject atomically; exact close tie and later close succeed |
| OBSERVER-009 | Research 20 sec | BAL | `TestObserver_OutboundArrivalStartsTwentySecondResearch`, `TestObserver_ExploringBeforeDeadlineRemainsExploring`, `TestObserver_ResearchCompletesAtDeadline` | `ResolveObserverLifecycle` | GREEN | exact 20s deadline and boundary covered |
| OBSERVER-010 | Research → WAITING_RETURN | BEH | `TestObserver_ResearchCompletionUsesDeadlineAsTransitionTime`, `TestObserver_ResearchCompletionBecomesWaitingReturn`, `TestObserver_WaitingReturnStartsAtResearchCompletion`, `TestObserver_WaitingReturnHasNoAutomaticEnd`, `TestObserver_CatchUpOutboundThroughResearchEndsWaitingReturn`, `TestObserver_CatchUpUsesEffectiveTransitionTimestamps`, `TestObserver_RepeatedResolveIsIdempotent` | `ResolveObserverLifecycle` | GREEN | waiting timestamp retained; large jumps cross multiple phases once at effective deadlines |
| OBSERVER-011 | Return → AVAILABLE | BEH | `TestObserver_StartReturningFromWaitingReturn`, `TestObserver_ReturningBeforeDeadlineRemainsReturning`, `TestObserver_ReturningAtDeadlineBecomesAvailable`, `TestObserver_ReturnUsesDeadlineAsTransitionTime`, `TestObserver_ReturnClearsCurrentPlaneAndActivePortal`, `TestObserver_ReturningSucceedsWhenPortalClosesExactlyAtTransitEnd`, `TestObserver_PortalClosingAfterReturnEndDoesNotRetroactivelyLoseObserver` | `Observer.StartReturning`, `ResolveObserverLifecycle` | GREEN | current Plane retained in transit; exact close tie/later close succeed; canonical AVAILABLE restored |
| OBSERVER-012 | CLOSED in transit → LOST | BEH | `TestObserver_OutboundLostWhenPortalClosesBeforeTransitEnd`, `TestObserver_ReturningLostWhenPortalClosesBeforeTransitEnd`, `TestObserver_LossUsesPortalClosedAtAsTransitionTime`, `TestObserver_LossClearsActiveState` | `ResolveObserverLifecycle` | GREEN | strict-before failure; LOST fields canonicalized and timestamped at `ClosedAt` |
| OBSERVER-013 | COLLAPSED in transit → LOST | BEH | `TestObserver_OutboundLostWhenPortalCollapsesBeforeTransitEnd`, `TestObserver_ReturningLostWhenPortalCollapsesBeforeTransitEnd`, `TestObserver_LossDoesNotExplorePlane` | `ResolveObserverLifecycle` | GREEN | same loss semantics for ENERGY_DEPLETED/INSTABILITY collapse |
| OBSERVER-014 | Multiple per Plane | BEH | `TestObservers_MultipleObserversMayExploreSamePlaneConcurrently` | lifecycle primitives + `ResolveObserverLifecycle` | GREEN | no uniqueness-by-Plane guard; Stage 4 separately owns per-Portal transit admission |
| OBSERVER-015 | Send to explored allowed | BEH | `TestSendObserver_AllowsAlreadyExploredPlane` | `SendObserver` | GREEN | explored destination is not an admission restriction |
| OBSERVER-016 | Recall longest-waiting | BEH | `TestLongestWaitingObserverIndex_SelectsEarliestWaitingTimestamp`, `TestLongestWaitingObserverIndex_FiltersByDestinationPlane`, `TestLongestWaitingObserverIndex_BreaksExactTieByLowestID`, `TestLongestWaitingObserverIndex_RejectsMissingWaitingTimestamp`, `TestRecallObserver_SelectsLongestWaitingInDestination`, `TestRecallObserver_BreaksWaitingTieByLowestID`, `TestRecallObserver_IgnoresWaitingObserversInOtherPlanes` | `LongestWaitingObserverIndex`, `RecallObserver` | GREEN | earliest waiting timestamp in destination wins; exact tie uses lowest ID |

## FLOW

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| FLOW-001 | Natural starts NONE | BEH | — | — | PLANNED | |
| FLOW-002 | First SEND → OUTBOUND | BEH | `TestSendObserver_FirstUseSetsOutboundFlow`, `TestSendObserver_ExistingOutboundFlowRemainsOutbound`, `TestSendObserver_UpdatesPortalOnlyWhenFlowFirstChanges` | `SendObserver` | GREEN | first successful SEND fixes flow after Observer transit starts |
| FLOW-003 | First RECALL → INBOUND | BEH | `TestRecallObserver_FirstUseSetsInboundFlow`, `TestRecallObserver_ExistingInboundFlowRemainsInbound` | `RecallObserver` | GREEN | first successful RECALL fixes flow after Observer transit starts |
| FLOW-004 | OUTBOUND rejects RECALL | BEH | `TestRecallObserver_RejectsOutboundFlow`, `TestRecallObserver_RejectionIsAtomic`, `TestRecallObserver_RejectionDoesNotConsumeRandom` | `RecallObserver` | GREEN | direction rejection precedes later admission checks and never mutates |
| FLOW-005 | INBOUND rejects SEND | BEH | `TestSendObserver_RejectsInboundFlow`, `TestSendObserver_RejectionIsAtomic`, `TestSendObserver_RejectionDoesNotConsumeRandom` | `SendObserver` | GREEN | direction rejection precedes later admission checks and never mutates |
| FLOW-006 | One transit at a time | INV | `TestActiveTransitObserverIndex_FindsOutbound`, `TestActiveTransitObserverIndex_FindsReturning`, `TestActiveTransitObserverIndex_IgnoresOtherPortals`, `TestActiveTransitObserverIndex_ReturnsNoneWhenIdle`, `TestActiveTransitObserverIndex_RejectsMultipleTransitsForSamePortal`, `TestActiveTransitObserverIndex_RejectsStaleTransitAtDeadline`, `TestSendObserver_RejectsBusyPortal`, `TestRecallObserver_RejectsBusyPortal` | `ActiveTransitObserverIndex`, `SendObserver`, `RecallObserver` | GREEN | both commands reject a busy Portal; corrupted duplicate/stale active transit is invariant error |
| FLOW-007 | Same direction after transit | BEH | — | — | PLANNED | |

## EXTRACTION

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| EXTRACTION-001 | Cost 30 | BAL | — | — | PLANNED | |
| EXTRACTION-002 | Requires waiting observer | BEH | — | — | PLANNED | |
| EXTRACTION-003 | Requires free slot | BEH | — | — | PLANNED | |
| EXTRACTION-004 | Stable/INBOUND/creatures0 | BEH | — | — | PLANNED | |
| EXTRACTION-005 | Energy 60..100 | BAL | — | — | PLANNED | |
| EXTRACTION-006 | TTL 30..60 | BAL | — | — | PLANNED | |
| EXTRACTION-007 | Sync 5 sec | BEH | — | — | PLANNED | |
| EXTRACTION-008 | First auto-return | BEH | — | — | PLANNED | |
| EXTRACTION-009 | Only one automatic | BEH | — | — | PLANNED | |
| EXTRACTION-010 | Further returns manual | BEH | — | — | PLANNED | |

## EMERGENCY

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| EMERGENCY-001 | Collapse → energy 0 | BEH | — | — | PLANNED | |
| EMERGENCY-002 | Override 20 sec | BEH | — | — | PLANNED | |
| EMERGENCY-003 | Close/Stabilize free in override | BEH | — | — | PLANNED | |
| EMERGENCY-004 | Extraction stays 30 | INV | — | — | PLANNED | |
| EMERGENCY-005 | Regen continues | BEH | — | — | PLANNED | |
| EMERGENCY-006 | New collapse resets | BEH | — | — | PLANNED | |

## EVENT

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| EVENT-001 | Meaningful transition → Event | BEH | — | — | PLANNED | |
| EVENT-002 | Risk event on level change only | BEH | — | — | PLANNED | |
| EVENT-003 | Rejected action → ACTION_REJECTED | BEH | — | — | PLANNED | |
| EVENT-004 | Portal history filters portal_id | BEH | — | — | PLANNED | |
| EVENT-005 | Global log same source | INV | — | — | PLANNED | |

## TUTORIAL

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| TUTORIAL-001 | Deterministic state machine | BEH | — | — | PLANNED | |
| TUTORIAL-002 | Advance on expected condition | BEH | — | — | PLANNED | |
| TUTORIAL-003 | Wrong reversible → same step | BEH | — | — | PLANNED | |
| TUTORIAL-004 | Wrong irreversible → recreate | BEH | — | — | PLANNED | |
| TUTORIAL-005 | Stabilize HIGH→MEDIUM guaranteed | BEH | — | — | PLANNED | |
| TUTORIAL-006 | Critical step expects rejected SEND | BEH | — | — | PLANNED | |
| TUTORIAL-007 | Exploration after return only | BEH | — | — | PLANNED | |
| TUTORIAL-008 | Energy not reset entering Live | BEH | — | — | PLANNED | |

## API / WS / UI / PERSIST

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| API-001..011 | REST endpoints, confirm flow, 409 | BEH | — | — | PLANNED | детализация Stage 12 |
| WS-001..003 | /ws/lab, ~1/sec snapshot, immediate after action | BEH | — | — | PLANNED | детализация Stage 13 |
| UI-001..007 | Dashboard/Details/Log/Worklog contracts | UI | — | — | PLANNED | детализация Stage 15–21 |
| PERSIST-001..004 | SQLite tables, no per-sec writes, recovery | BEH | — | — | PLANNED | детализация Stage 10 |

---

## Infrastructure coverage (без ID)

| Area | Test | Status | Notes |
|---|---|---|---|
| Balance config = Final Spec §37 | `TestDefault_MatchesFinalSpecBalance` (`internal/config`) | GREEN | исполняемая часть Stage 0 |
| RealClock | `TestRealClock_ReturnsCurrentTime` (`internal/clock`) | GREEN | |
| FakeClock | `testutil` clock tests | GREEN | incl. concurrent Advance/Now (race) |
| RealRandom | `internal/random` tests | GREEN | bounds, degenerate range, panic, concurrency |
| FakeRandom | `testutil` random tests | GREEN | queue order, passthrough, exhaustion panic |
| PortalBuilder | `testutil` builders tests | GREEN | defaults + fluent overrides + unstable fixture |
