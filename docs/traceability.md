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
| PLANE-003 | Multiple OPEN Portals to same Plane | BEH | `TestResolveNaturalSpawn_AllowsRepeatedDestination` | `ResolveNaturalSpawn` | GREEN | Natural generation permits multiple OPEN Portal instances with the same destination Plane |
| PLANE-004 | Seed 85, default UNEXPLORED | BEH | `TestSimulationState_RequiresExactlyEightyFivePlanes`, `TestSimulationState_RejectsDuplicatePlaneIDs` | `validateSimulationState` | PARTIAL | active aggregate enforces 85 unique Plane IDs; canonical seed/default data remains Stage 10 |
| PLANE-005 | SEND does not explore | INV | `TestObserver_StartOutboundDoesNotExplorePlane`, `TestSendObserver_DoesNotExplorePlane`, `TestSendObserver_AllowsAlreadyExploredPlane`, `TestObserverCommands_DoNotChangePlaneExplorationOnTransitStart` | `Observer.StartOutbound`, `SendObserver`, `RecallObserver` | GREEN | transit-start commands leave both unexplored and already-explored Plane state unchanged |
| PLANE-006 | Arrival does not explore | INV | `TestObserver_OutboundArrivalDoesNotExplorePlane` | `ResolveObserverLifecycle` | GREEN | successful OUTBOUND leaves Plane exploration unchanged |
| PLANE-007 | Research completion does not explore | INV | `TestObserver_ResearchCompletionDoesNotExplorePlane` | `ResolveObserverLifecycle` | GREEN | WAITING_RETURN retains Plane location but not exploration |
| PLANE-008 | Successful return explores | BEH | `TestObserver_SuccessfulReturnExploresPlane`, `TestObserver_SuccessfulReturnSetsExploredAtToArrival`, `TestSimulationTick_SuccessfulReturnExploresPlane` | `ResolveObserverLifecycle`, `resolveObserverStage` | GREEN | explored at semantic return deadline, including through the simulation tick |
| PLANE-009 | Re-return idempotent | INV | `TestObserver_ReturnToAlreadyExploredPlanePreservesOriginalExploredAt`, `TestObservers_MultipleObserversMayExploreSamePlaneConcurrently`, `TestSimulationTick_RepeatReturnPreservesExploredAt` | `ResolveObserverLifecycle`, `resolveObserverStage` | PARTIAL | Plane-state idempotence (`Explored`/`ExploredAt`) is proven through the tick; progress aggregation remains later-stage work |

## PORTAL

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| PORTAL-001 | Unique instance per opening | INV | `TestResolveNaturalSpawn_UsesNextPortalID`, `TestResolveNaturalSpawn_IncrementsSequenceExactlyOnce`, `TestResolveNaturalSpawn_AppendsWithoutRemovingTerminalHistory` | `ResolveNaturalSpawn`, `validateSimulationState` | PARTIAL | Natural generation creates a fresh unique instance and preserves history; Stage 11 must serialize all opening commands |
| PORTAL-002 | Sequential `Omenpath #XXXX` | BEH | `TestNewNaturalPortal_GeneratesUnstableWithinSpec`, `TestResolveNaturalSpawn_FormatsSequentialPortalName`, `TestResolveNaturalSpawn_IncrementsSequenceExactlyOnce` | `NewNaturalPortal`, `ResolveNaturalSpawn` | PARTIAL | Natural aggregate sequence is proven; Stage 11 must serialize all opening commands |
| PORTAL-003 | NATURAL/EXTRACTION kinds | BEH | `TestNewNaturalPortal_GeneratesUnstableWithinSpec`, `TestNewExtractionPortal_SetsExtractionKind` | `PortalKind` enum, `NewNaturalPortal`, `NewExtractionPortal` | GREEN | both kinds have concrete pure factories |
| PORTAL-004 | OPEN/CLOSED/COLLAPSED statuses | INV | `TestPortal_ScheduledRemainingIsZeroWhenTerminal`, `TestPortal_EnergyFreezesAtManualClose`, `TestPortal_EnergyFreezesAtCollapse`, `TestPortal_CreaturesFreezeAtCollapse` | `PortalStatus` enum + terminal derived state freeze | GREEN | все три статуса и разные outcomes (CLOSED vs COLLAPSED, разные termination reasons) покрыты прямо; повышено с PLANNED в corrective-пасе |
| PORTAL-005 | TTL expiry → CLOSED/NATURAL_CLOSE | BEH | `TestPortal_NaturalCloseWhenTTLExpires`, `TestPortal_LateResolutionPreservesNaturalCloseTime`, `TestSimulationTick_ResolvesNaturalClose`, `TestSimulationTick_NaturalCloseDoesNotActivateOverride` | `Portal.ResolveLifecycle`, `SimulationState.ResolveTick` | GREEN | semantic ClosedAt is preserved through the ordered simulation Portal stage; Natural Close does not activate Override |
| PORTAL-006 | Manual Close → CLOSED/MANUAL_CLOSE | BEH | `TestPortal_ManualCloseWithoutCreatures`, `TestClosePortalWithObservers_ConfirmedReturningBecomesLost`, `TestClosePortalWithLabEnergy_ChargesFive`, `TestExtractionClose_DuringSyncClosesPortal`, `TestExtractionClose_DuringReturningRequiresConfirmation`, `TestExtractionClose_ConfirmedSetsManualCloseReason` | `Portal.Close`, `ClosePortalWithObservers`, `ClosePortalWithLabEnergy` | GREEN | same atomic Close applies before sync and during Extraction return; active transit still requires confirmation |
| PORTAL-007 | Energy 0 → COLLAPSED/ENERGY_DEPLETED | BEH | `TestPortal_CollapsesWhenEnergyReachesZeroBeforeNaturalClose`, `TestPortal_NaturalCloseWinsEnergyTie`, `TestPortal_NaturalCloseBeforeEnergyDepletion`, `TestResolvePortalLifecycleWithLabEmergency_EnergyDepletionResetsLab`, `TestResolvePortalLifecycleWithLabEmergency_UsesPortalClosedAt` | `Portal.ResolveLifecycle`, `ResolvePortalLifecycleWithLabEmergency` | GREEN | tie → NATURAL_CLOSE; energy Collapse now atomically triggers Lab emergency at semantic `ClosedAt` |
| PORTAL-008 | Hidden instability → COLLAPSED/INSTABILITY | BEH | `TestPortal_UnstableCollapsesAtHiddenTime`, `TestPortal_NaturalCloseWinsBeforeHiddenCollapse`, `TestPortal_NaturalCloseTiesHiddenCollapse`, `TestPortal_EnergyDepletionWinsInstabilityTie`, `TestResolvePortalLifecycleWithLabEmergency_InstabilityResetsLab`, `TestResolvePortalLifecycleWithLabEmergency_InstabilityUsesHiddenCollapseTime`, `TestResolvePortalLifecycleWithLabEmergency_PreservesEnergyDepletedTieReason` | `Portal.ResolveLifecycle`, `ResolvePortalLifecycleWithLabEmergency` | GREEN | existing tie rules preserved; instability Collapse atomically triggers Lab emergency at hidden semantic time |
| PORTAL-009 | Terminal cannot return OPEN | INV | `TestPortal_TerminalStateCannotReopen`, `TestPortal_ScheduledRemainingIsZeroWhenTerminal`, `TestPortal_EnergyFreezesAtManualClose`, `TestPortal_EnergyFreezesAtCollapse`, `TestPortal_CreaturesFreezeAtManualClose`, `TestPortal_CreaturesFreezeAtCollapse` | `Portal.IsTerminal` + `ResolveLifecycle` guard + derived state freeze | GREEN | snapshot-equality — нет мутаций; расширен аудитом: derived state замораживается на ClosedAt (ScheduledRemaining=0, Energy/Creatures frozen), risk отсутствует |

## SLOT

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| SLOT-001 | Exactly 7 slots | INV | `TestFirstFreeSlot_ReturnsNoneWhenAllSevenOpen` | `FirstFreeSlot` + `cfg.MaxActivePortals` | PARTIAL | domain/config часть (bound 7) покрыта; «Dashboard всегда содержит 7 фиксированных Slots» — фронтенд Stage 15/16 |
| SLOT-002 | OPEN Portal occupies a slot | BEH | `TestFirstFreeSlot_ReturnsNextAfterOccupiedPrefix` | `FirstFreeSlot` | GREEN | |
| SLOT-003 | First free slot | BEH | `TestFirstFreeSlot_FillsGap` | `FirstFreeSlot` | GREEN | |
| SLOT-004 | Portal keeps slot for lifecycle | INV | `TestFirstFreeSlot_DoesNotMutateInput` | pure helper | PARTIAL | helper не мутирует вход / не пересортирует; инвариант целиком (хранение slot_index у живого портала) — LabManager Stage 11 |
| SLOT-005 | Terminal releases slot | BEH | `TestFirstFreeSlot_TerminalPortalsDoNotOccupy` | `FirstFreeSlot` | GREEN | |
| SLOT-006 | 7/7 blocks natural spawn | BEH | `TestFirstFreeSlot_ReturnsNoneWhenAllSevenOpen`, `TestResolveNaturalSpawn_FullCapacityPausesImmediately`, `TestResolveNaturalSpawn_FirstTickAfterFreeOnlySchedules`, `TestResolveNaturalSpawn_SeventhPortalPausesWithoutNextDelayDraw`, `TestResolveNaturalSpawn_SixthPortalDrawsNextDelay` | `FirstFreeSlot`, `ResolveNaturalSpawn` | GREEN | backend Natural Generator re-checks capacity, pauses at 7/7, and starts a fresh delay after a Slot becomes free |
| SLOT-007 | Extraction uses regular slot | BEH | `TestOpenExtractionPortal_UsesFirstFreeRegularSlot`, `TestOpenExtractionPortal_ReusesTerminalPortalSlot`, `TestOpenExtractionPortal_RejectsAllSevenSlotsOccupied` | `OpenExtractionPortal`, `FirstFreeSlot` | GREEN | same first-free pool and terminal release rules as Natural portals |

## ENERGY

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| ENERGY-001 | Initial natural 10..100 | BAL | `TestNewNaturalPortal_GeneratesUnstableWithinSpec` | `NewNaturalPortal` | GREEN | draw из cfg-диапазона |
| ENERGY-002 | Decay 0.1..1.0/sec hidden | BAL | `TestNewNaturalPortal_GeneratesUnstableWithinSpec` | `NewNaturalPortal` | PARTIAL | диапазон 0.1..1.0 покрыт фабрикой; скрытость от пользователя (не отдаётся API/UI) — Stage 12/13 |
| ENERGY-003 | Current derived from baseline/time | BEH | `TestPortal_CurrentEnergy`, `TestPortal_EnergyDepletionAt` | `Portal.CurrentEnergy`, `Portal.EnergyDepletionAt` | GREEN | |
| ENERGY-004 | Clamp ≥ 0 | INV | `TestPortal_CurrentEnergy/clamped_at_zero` | `Portal.CurrentEnergy` | GREEN | |
| ENERGY-005 | 0 before close → COLLAPSED/ENERGY_DEPLETED | BEH | `TestPortal_CollapsesWhenEnergyReachesZeroBeforeNaturalClose` | `Portal.EnergyDepletionAt` | GREEN | Stage 0–1 first RED cycle |
| ENERGY-006 | Stabilize +15 | BEH | `TestPortal_StabilizeConvertsUnstableToStable`, `TestStabilizePortalWithLabEnergy_SuccessPreservesPortalSemantics`, `TestStabilizePortalWithLabEnergy_ActiveOverridePreservesPortalBoost` | `Portal.Stabilize`, `StabilizePortalWithLabEnergy` | GREEN | ordinary and free-Override wrappers preserve independent Portal Energy boost |
| ENERGY-007 | Stabilize re-baselines | BEH | `TestPortal_StabilizeConvertsUnstableToStable`, `TestPortal_StabilizeUsesCurrentEnergyNotBaseline`, `TestStabilizePortalWithLabEnergy_SuccessPreservesPortalSemantics`, `TestStabilizePortalWithLabEnergy_ActiveOverridePreservesPortalBoost` | `Portal.Stabilize`, `StabilizePortalWithLabEnergy` | GREEN | stale-baseline regression covered for ordinary and Override paths |
| ENERGY-008 | Decay unchanged after Stabilize | INV | `TestPortal_StabilizeConvertsUnstableToStable`, `TestStabilizePortalWithLabEnergy_SuccessPreservesPortalSemantics`, `TestStabilizePortalWithLabEnergy_ActiveOverridePreservesPortalDecay` | `Portal.Stabilize`, `StabilizePortalWithLabEnergy` | GREEN | neither Lab debit nor free Override changes Portal decay |
| ENERGY-009 | >85 rejects Stabilize | BEH | `TestPortal_StabilizeRejectsEnergyAbove85` | `Portal.Stabilize` | GREEN | проверка по current, не baseline |
| ENERGY-010 | Exactly 85 → 100 | BEH | `TestPortal_StabilizeAllowsExactly85Percent` | `Portal.Stabilize` | GREEN | |

## STABILITY

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| STABILITY-001 | STABLE/UNSTABLE only | INV | `TestPortal_UnstableCollapsesAtHiddenTime`, `TestNewNaturalPortal_GeneratesUnstableWithinSpec`, `TestNewNaturalPortal_StableHasNoHiddenTimestamp` | `PortalStability` enum | GREEN | обе стабильности генерируются factory |
| STABILITY-002 | Stable has no hidden timer | INV | `TestNewNaturalPortal_StableHasNoHiddenTimestamp`, `TestPortalBuilder_Defaults` | factory / builder / `Stabilize` | GREEN | |
| STABILITY-003 | Unstable gets hidden timer | BEH | `TestNewNaturalPortal_GeneratesUnstableWithinSpec`, `TestNewNaturalPortal_HiddenCollapseWithinWindow` | `NewNaturalPortal` | GREEN | random(opened+5s, close−1s) |
| STABILITY-004 | Hidden timer not exposed / not in risk | UI | `TestPortal_HiddenInstabilityTimestampDoesNotAffectRisk` | `Portal.RiskScore` | PARTIAL | доказана только domain-часть: hidden не участвует в Risk; «не exposed пользователю» (API/UI не отдаёт timestamp) — Stage 12/13 |
| STABILITY-005 | Stabilize unstable→stable | BEH | `TestPortal_StabilizeConvertsUnstableToStable`, `TestStabilizePortalWithLabEnergy_SuccessPreservesPortalSemantics`, `TestStabilizePortalWithLabEnergy_ActiveOverrideAllowsZeroEnergy` | `Portal.Stabilize`, `StabilizePortalWithLabEnergy` | GREEN | ordinary and free Override paths delegate the same Portal transition |
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
| LAB-001 | Integer 0..100 | INV | `TestNewLabState_AcceptsZero`, `TestNewLabState_AcceptsMaximum`, `TestNewLabState_RejectsBelowZero`, `TestNewLabState_RejectsAboveMaximum`, `TestLabState_CurrentEnergyIsInteger`, `TestLabState_SpendEnergyAllowsExactBalance`, `TestLabState_SpendEnergyRejectsMalformedBaseline` | `NewLabState`, `LabState.CurrentEnergy`, `LabState.SpendEnergy` | GREEN | inclusive constructor bounds; derived reads clamp; mutations reject malformed baselines and never produce negative energy |
| LAB-002 | Tutorial starts 100 | BEH | `TestNewTutorialLabState_StartsAtMaximum`, `TestNewTutorialLabState_UsesProvidedTimestamp`, `TestNewTutorialLabState_HasNoOverride` | `NewTutorialLabState` | PARTIAL | domain constructor proven; Tutorial bootstrap remains Stage 14 |
| LAB-003 | Regen +1/sec | BEH | `TestLabState_CurrentEnergyAtBaseline`, `TestLabState_CurrentEnergyUsesCompletedWholeSeconds`, `TestLabState_CurrentEnergyAtExactSecond`, `TestLabState_CurrentEnergyUsesConfiguredRate`, `TestLabState_CurrentEnergyBeforeBaselineDoesNotRegenerate`, `TestLabState_CurrentEnergyDoesNotMutateBaseline`, `TestLabEnergyCommands_RepeatedReadDoesNotWriteBaseline`, `TestResolvePortalLifecycleWithLabEmergency_LateResolutionIncludesRegeneration`, `TestResolvePortalLifecycleWithLabEmergency_LateInstabilityResolutionRegenerates`, `TestLabState_LabEnergyRegeneratesDuringOverride`, `TestLabState_OverrideRegenerationUsesCompletedWholeSeconds`, `TestLabState_FreeCloseDoesNotInterruptRegeneration`, `TestLabState_FreeStabilizeDoesNotInterruptRegeneration` | `LabState.CurrentEnergy`, `ResolvePortalLifecycleWithLabEmergency` | GREEN | same completed-second derivation runs throughout Override and survives free actions |
| LAB-004 | Cap 100 | BEH | `TestLabState_CurrentEnergyCapsAtMaximum` | `LabState.CurrentEnergy` | GREEN | configured maximum cap |
| LAB-005 | SEND 0 | BAL | `TestSendObserverWithLabEnergy_AllowsZeroEnergy`, `TestSendObserverWithLabEnergy_CostsZero`, `TestSendObserverWithLabEnergy_DoesNotRebaseEnergy` | `SendObserverWithLabEnergy` | GREEN | zero-cost command validates LabState but never rebases it |
| LAB-006 | RECALL 0 | BAL | `TestRecallObserverWithLabEnergy_AllowsZeroEnergy`, `TestRecallObserverWithLabEnergy_CostsZero`, `TestRecallObserverWithLabEnergy_DoesNotRebaseEnergy` | `RecallObserverWithLabEnergy` | GREEN | zero-cost command validates LabState but never rebases it |
| LAB-007 | CLOSE 5 | BAL | `TestClosePortalWithLabEnergy_ChargesFive`, `TestClosePortalWithLabEnergy_AllowsExactFive`, `TestClosePortalWithLabEnergy_UsesRegeneratedEnergy`, `TestClosePortalWithLabEnergy_ConfirmedTransitDebitsOnce`, `TestClosePortalWithLabEnergy_ActiveOverrideCostsZero`, `TestClosePortalWithLabEnergy_AtOverrideDeadlineChargesNormalCost` | `ClosePortalWithLabEnergy`, `effectiveCloseCost` | GREEN | ordinary cost 5; active Override selects 0; exact deadline restores configured cost |
| LAB-008 | STABILIZE 20 | BAL | `TestStabilizePortalWithLabEnergy_ChargesTwenty`, `TestStabilizePortalWithLabEnergy_AllowsExactTwenty`, `TestStabilizePortalWithLabEnergy_UsesRegeneratedEnergy`, `TestStabilizePortalWithLabEnergy_DebitsExactlyOnce`, `TestStabilizePortalWithLabEnergy_ActiveOverrideCostsZero`, `TestStabilizePortalWithLabEnergy_AtDeadlineChargesNormalCost` | `StabilizePortalWithLabEnergy`, `effectiveStabilizeCost` | GREEN | ordinary cost 20; active Override selects 0; exact deadline restores configured cost |
| LAB-009 | EXTRACTION 30 | BAL | `TestLabState_ExtractionCostIsThirty`, `TestOpenExtractionPortal_ChargesThirty`, `TestOpenExtractionPortal_AllowsExactThirty`, `TestOpenExtractionPortal_UsesRegeneratedEnergy`, `TestOpenExtractionPortal_ChargesThirtyDuringOverride`, `TestOpenExtractionPortal_PreservesOverrideDeadline` | `cfg.ExtractionCost`, `LabState.SpendEnergy`, `OpenExtractionPortal` | GREEN | real opening transaction always charges 30 and preserves the Override deadline |
| LAB-010 | Insufficient energy rejects | BEH | `TestLabState_SpendEnergyRejectsInsufficientBalance`, `TestClosePortalWithLabEnergy_InsufficientLeavesPortalUnchanged`, `TestStabilizePortalWithLabEnergy_InsufficientLeavesPortalUnchanged`, `TestOpenExtractionPortal_RejectsTwentyNine`, `TestOpenExtractionPortal_InsufficientEnergyLeavesLabUnchanged`, `TestOpenExtractionPortal_InsufficientEnergyLeavesPortalsUnchanged`, `TestOpenExtractionPortal_InsufficientEnergyLeavesObserversUnchanged` | `LabState.CanAfford`, `LabState.SpendEnergy`, `ClosePortalWithLabEnergy`, `StabilizePortalWithLabEnergy`, `OpenExtractionPortal` | GREEN | all real paid domain actions reject insufficient energy before aggregate mutation |

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
| OBSERVER-012 | CLOSED in transit → LOST | BEH | `TestObserver_OutboundLostWhenPortalClosesBeforeTransitEnd`, `TestObserver_ReturningLostWhenPortalClosesBeforeTransitEnd`, `TestObserver_LossUsesPortalClosedAtAsTransitionTime`, `TestObserver_LossClearsActiveState`, `TestClosePortalWithObservers_ConfirmedOutboundBecomesLost`, `TestClosePortalWithObservers_ConfirmedReturningBecomesLost`, `TestClosePortalWithObservers_ConfirmedCloseUsesNowForBothTransitions`, `TestClosePortalWithObservers_LossClearsObserverCurrentState`, `TestClosePortalWithObservers_LossDoesNotExplorePlane`, `TestClosePortalWithLabEnergy_ConfirmedTransitClosesAndLosesObserver`, `TestClosePortalWithLabEnergy_ActiveOverrideStillRequiresTransitConfirmation`, `TestClosePortalWithLabEnergy_ActiveOverrideConfirmedTransitLosesObserver` | `ResolveObserverLifecycle`, `ClosePortalWithObservers`, `ClosePortalWithLabEnergy` | GREEN | free Override Close preserves confirmation and canonical LOST semantics |
| OBSERVER-013 | COLLAPSED in transit → LOST | BEH | `TestObserver_OutboundLostWhenPortalCollapsesBeforeTransitEnd`, `TestObserver_ReturningLostWhenPortalCollapsesBeforeTransitEnd`, `TestObserver_LossDoesNotExplorePlane`, `TestExtractionLifecycle_CollapseDuringReturningBecomesLost` | `ResolveObserverLifecycle` | GREEN | same loss semantics apply to Extraction RETURNING transit |
| OBSERVER-014 | Multiple per Plane | BEH | `TestObservers_MultipleObserversMayExploreSamePlaneConcurrently`, `TestObserverCommands_MultipleObserversMayRemainInSamePlane` | lifecycle primitives + `RecallObserver` + `ResolveObserverLifecycle` | GREEN | no uniqueness-by-Plane guard; Stage 4 only limits active transit per Portal |
| OBSERVER-015 | Send to explored allowed | BEH | `TestSendObserver_AllowsAlreadyExploredPlane` | `SendObserver` | GREEN | explored destination is not an admission restriction |
| OBSERVER-016 | Recall longest-waiting | BEH | `TestLongestWaitingObserverIndex_SelectsEarliestWaitingTimestamp`, `TestRecallObserver_SelectsLongestWaitingInDestination`, `TestResolveExtractionSynchronization_StartsLongestWaitingReturn`, `TestResolveExtractionSynchronization_BreaksWaitingTieByLowestID`, `TestResolveExtractionSynchronization_OriginalLongestGoneSelectsNextWaiting` | `LongestWaitingObserverIndex`, `RecallObserver`, `RecallObserverWithLabEnergy`, `ResolveExtractionSynchronization` | GREEN | manual and automatic return both select the current longest waiter; exact tie uses lowest ID |

## FLOW

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| FLOW-001 | Natural starts NONE | BEH | `TestNaturalPortal_ObserverFlowStartsNone` | `NewNaturalPortal` | GREEN | Stage 2 factory behavior explicitly characterized at Stage 4 boundary |
| FLOW-002 | First SEND → OUTBOUND | BEH | `TestSendObserver_FirstUseSetsOutboundFlow`, `TestSendObserver_ExistingOutboundFlowRemainsOutbound`, `TestSendObserver_UpdatesPortalOnlyWhenFlowFirstChanges`, `TestPortalFlow_DoesNotResetAfterSuccessfulOutbound`, `TestPortalFlow_DoesNotResetAfterObserverLost`, `TestSendObserverWithLabEnergy_DelegatesStage4Flow` | `SendObserver`, `SendObserverWithLabEnergy` | GREEN | first successful SEND fixes flow after Observer transit starts; lifecycle outcomes never reset it |
| FLOW-003 | First RECALL → INBOUND | BEH | `TestRecallObserver_FirstUseSetsInboundFlow`, `TestRecallObserver_ExistingInboundFlowRemainsInbound`, `TestPortalFlow_DoesNotResetAfterSuccessfulReturn`, `TestNewExtractionPortal_StartsInbound` | `RecallObserver`, `RecallObserverWithLabEnergy`, `NewExtractionPortal` | GREEN | Natural first RECALL fixes direction; Extraction is prescribed INBOUND at creation |
| FLOW-004 | OUTBOUND rejects RECALL | BEH | `TestRecallObserver_RejectsOutboundFlow`, `TestRecallObserver_RejectionIsAtomic`, `TestRecallObserver_RejectionDoesNotConsumeRandom`, `TestPortalFlow_RejectsRecallAfterOutboundTransitEnds`, `TestRecallObserverWithLabEnergy_RejectionDoesNotChangeLab`, `TestRecallObserverWithLabEnergy_RejectionDoesNotConsumeRandom` | `RecallObserver`, `RecallObserverWithLabEnergy` | GREEN | direction rejection persists after Portal becomes idle and never mutates Lab/Portal/Observer or random state |
| FLOW-005 | INBOUND rejects SEND | BEH | `TestSendObserver_RejectsInboundFlow`, `TestSendObserver_RejectionIsAtomic`, `TestSendObserver_RejectionDoesNotConsumeRandom`, `TestPortalFlow_RejectsSendAfterInboundTransitEnds`, `TestSendObserverWithLabEnergy_RejectionDoesNotChangeLab`, `TestSendObserverWithLabEnergy_RejectionDoesNotConsumeRandom` | `SendObserver`, `SendObserverWithLabEnergy` | GREEN | direction rejection persists after Portal becomes idle and never mutates Lab/Portal/Observer or random state |
| FLOW-006 | One transit at a time | INV | `TestActiveTransitObserverIndex_FindsOutbound`, `TestActiveTransitObserverIndex_FindsReturning`, `TestActiveTransitObserverIndex_IgnoresOtherPortals`, `TestActiveTransitObserverIndex_ReturnsNoneWhenIdle`, `TestActiveTransitObserverIndex_RejectsMultipleTransitsForSamePortal`, `TestActiveTransitObserverIndex_RejectsStaleTransitAtDeadline`, `TestSendObserver_RejectsBusyPortal`, `TestRecallObserver_RejectsBusyPortal`, `TestPortalBusy_IsScopedToPortalID`, `TestPortalBusy_DoesNotBlockAnotherPortalToSamePlane`, `TestClosePortalWithObservers_RejectsMultipleActiveTransits`, `TestClosePortalWithObservers_RejectsStaleTransitBeforeMutation`, `TestObserverCommands_DifferentPortalsOperateIndependently`, `TestObserverCommands_DoNotMutateOnStructuralInvariantError`, `TestLabEnergyCommands_DoNotChangeStage4ErrorIdentity` | `ActiveTransitObserverIndex`, `SendObserver`, `RecallObserver`, `ClosePortalWithObservers`, `SendObserverWithLabEnergy` | GREEN | busy scope is exact Portal ID; energy wrapper preserves error identity and does not weaken arbitration |
| FLOW-007 | Same direction after transit | BEH | `TestPortalFlow_AllowsNextOutboundAfterPreviousTransitEnds`, `TestPortalFlow_AllowsNextInboundAfterPreviousTransitEnds`, `TestPortalFlow_DoesNotResetAfterSuccessfulOutbound`, `TestPortalFlow_DoesNotResetAfterSuccessfulReturn`, `TestPortalFlow_DoesNotResetAfterObserverLost`, `TestObserverCommands_DoNotResetFlowDuringLifecycleCatchUp` | `SendObserver`, `RecallObserver`, `ResolveObserverLifecycle` | GREEN | next same-direction transit starts after active transit resolves; direction is permanent across catch-up |

## EXTRACTION

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| EXTRACTION-001 | Cost 30 | BAL | `TestOpenExtractionPortal_ChargesThirty`, `TestOpenExtractionPortal_AllowsExactThirty`, `TestOpenExtractionPortal_ChargesThirtyDuringOverride` | `OpenExtractionPortal`, `cfg.ExtractionCost` | GREEN | one transactional debit on successful open |
| EXTRACTION-002 | Requires waiting observer | BEH | `TestExtractionPlaneEligible_ReturnsTrueForWaitingObserver`, `TestExtractionPlaneEligible_FiltersBySelectedPlane`, `TestExtractionPlaneEligible_RejectsDuplicateObserverIDs` | `ExtractionPlaneEligible`, `validateObserverRoster` | GREEN | eligibility uses current canonical WAITING_RETURN state in the selected Plane |
| EXTRACTION-003 | Requires free slot | BEH | `TestOpenExtractionPortal_UsesFirstFreeRegularSlot`, `TestOpenExtractionPortal_RejectsAllSevenSlotsOccupied` | `OpenExtractionPortal`, `FirstFreeSlot` | GREEN | no dedicated Extraction capacity |
| EXTRACTION-004 | Stable/INBOUND/creatures0 | BEH | `TestNewExtractionPortal_IsStableWithoutHiddenCollapse`, `TestNewExtractionPortal_StartsInbound`, `TestNewExtractionPortal_HasNoCreatures` | `NewExtractionPortal` | GREEN | canonical Extraction creation state |
| EXTRACTION-005 | Energy 60..100 | BAL | `TestNewExtractionPortal_AcceptsMinimumEnergy`, `TestNewExtractionPortal_AcceptsMaximumEnergy` | `NewExtractionPortal` | GREEN | inclusive configured factory draw |
| EXTRACTION-006 | TTL 30..60 | BAL | `TestNewExtractionPortal_AcceptsMinimumTTL`, `TestNewExtractionPortal_AcceptsMaximumTTL` | `NewExtractionPortal` | GREEN | inclusive configured whole-second TTL draw |
| EXTRACTION-007 | Sync 5 sec | BEH | `TestResolveExtractionSynchronization_BeforeDeadlineChangesNothing`, `TestResolveExtractionSynchronization_AtDeadlineCompletes`, `TestResolveExtractionSynchronization_LateResolutionUsesSemanticDeadline`, `TestSimulationTick_ExtractionBeforeSyncRemainsPending`, `TestSimulationTick_ExtractionAtSyncCompletes`, `TestSimulationTick_ExtractionMarkerUsesSemanticDeadline` | `ResolveExtractionSynchronization`, `resolveExtractionStage`, `Portal.ExtractionSynchronizedAt` | GREEN | half-open five-second sync uses semantic deadline through the ordered tick and terminal portals remain no-op |
| EXTRACTION-008 | First auto-return | BEH | `TestResolveExtractionSynchronization_StartsLongestWaitingReturn`, `TestResolveExtractionSynchronization_OriginalLongestGoneSelectsNextWaiting`, `TestResolveExtractionSynchronization_UsesSyncDeadlineAsPhaseStart`, `TestResolveExtractionSynchronization_DrawsTransitDurationExactlyOnce`, `TestSimulationTick_ResearchCompletionFeedsSameTickExtraction`, `TestSimulationTick_ExtractionSelectsCurrentLongestWaiting`, `TestSimulationTick_ExtractionOriginalCandidateGoneSelectsNext` | `ResolveExtractionSynchronization`, `resolveExtractionStage`, `RecallObserver`, `Observer.StartReturning` | GREEN | Observer catch-up feeds same-tick sync; sync reselects the current longest waiter and starts one semantic-time transit |
| EXTRACTION-009 | Only one automatic | BEH | `TestResolveExtractionSynchronization_ReplayDoesNotStartSecondObserver`, `TestResolveExtractionSynchronization_ObserverAddedLaterIsNotAutomatic`, `TestExtractionPortal_OnlyFirstReturnIsAutomatic`, `TestExtraction_LateSynchronizationReplayIsIdempotent`, `TestSimulationTick_CompletedExtractionDoesNotReturnSecondObserver` | `Portal.ExtractionSynchronizedAt`, `ResolveExtractionSynchronization`, `resolveExtractionStage` | GREEN | completed marker makes every later resolver/tick call a no-op, including after another Observer becomes eligible |
| EXTRACTION-010 | Further returns manual | BEH | `TestRecallObserver_ExtractionBeforeSyncRejects`, `TestRecallObserver_ExtractionAfterSyncAllowsManualReturn`, `TestExtractionPortal_AfterAutomaticReturnAllowsNextManualRecall` | `RecallObserver`, `RecallObserverWithLabEnergy`, `ErrExtractionSynchronizing` | GREEN | manual recall is gated until sync and remains ordinary zero-cost RECALL afterward |

## EMERGENCY

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| EMERGENCY-001 | Collapse → energy 0 | BEH | `TestLabState_ActivateLeylineOverrideResetsEnergyToZero`, `TestLabState_ActivateLeylineOverrideUsesCollapseTimestampAsBaseline`, `TestResolvePortalLifecycleWithLabEmergency_EnergyDepletionResetsLab`, `TestResolvePortalLifecycleWithLabEmergency_EnergyDepletionStartsOverride`, `TestResolvePortalLifecycleWithLabEmergency_InstabilityResetsLab`, `TestResolvePortalLifecycleWithLabEmergency_InstabilityStartsOverride`, `TestResolvePortalLifecycleWithLabEmergency_NaturalCloseDoesNotStartOverride`, `TestResolvePortalLifecycleWithLabEmergency_NaturalCloseTieDoesNotStartOverride`, `TestResolvePortalLifecycleWithLabEmergency_AlreadyCollapsedDoesNotResetAgain`, `TestLabEmergency_OnlyCollapsedTransitionActivatesOverride`, `TestLabEmergency_ClosedPortalNeverActivatesOverride`, `TestLabEmergency_AlreadyTerminalPortalIsIdempotent`, `TestLabEmergency_InvalidTransitionLeavesLabAndPortalUnchanged` | `LabState.ActivateLeylineOverride`, `ResolvePortalLifecycleWithLabEmergency` | GREEN | both Collapse causes reset Lab at semantic `ClosedAt`; CLOSED, invalid and replayed terminal paths remain atomic and do not activate emergency |
| EMERGENCY-002 | Override 20 sec | BEH | `TestLabState_ActivateLeylineOverrideSetsConfiguredDeadline`, `TestLabState_LeylineOverrideActiveAtStart`, `TestLabState_LeylineOverrideActiveBeforeDeadline`, `TestLabState_LeylineOverrideInactiveAtDeadline`, `TestLabState_LeylineOverrideInactiveAfterDeadline`, `TestLabState_LeylineOverrideActiveDoesNotMutateState` | `LabState.ActivateLeylineOverride`, `LabState.LeylineOverrideActive`, `cfg.EmergencyDuration` | GREEN | half-open `[collapseAt, deadline)` window; expiry is derived and does not write state |
| EMERGENCY-003 | Close/Stabilize free in override | BEH | `TestClosePortalWithLabEnergy_ActiveOverrideCostsZero`, `TestClosePortalWithLabEnergy_ActiveOverrideAllowsZeroEnergy`, `TestClosePortalWithLabEnergy_ActiveOverrideDoesNotRebaseEnergy`, `TestClosePortalWithLabEnergy_ActiveOverrideStillRequiresCreatureConfirmation`, `TestClosePortalWithLabEnergy_ActiveOverrideStillRequiresTransitConfirmation`, `TestClosePortalWithLabEnergy_AtOverrideDeadlineChargesNormalCost`, `TestStabilizePortalWithLabEnergy_ActiveOverrideCostsZero`, `TestStabilizePortalWithLabEnergy_ActiveOverrideAllowsZeroEnergy`, `TestStabilizePortalWithLabEnergy_ActiveOverrideDoesNotRebaseLabEnergy`, `TestStabilizePortalWithLabEnergy_ActiveOverrideStillRejectsStablePortal`, `TestStabilizePortalWithLabEnergy_ActiveOverrideStillRejectsOvercharge`, `TestStabilizePortalWithLabEnergy_AtDeadlineChargesNormalCost`, `TestLabEmergency_OrdinaryCostsRemainAfterExpiry`, `TestLabEmergency_CloseFailureDoesNotConsumeRegeneratedEnergy`, `TestLabEmergency_StabilizeFailureDoesNotConsumeRegeneratedEnergy` | `LabState.LeylineOverrideActive`, `effectiveCloseCost`, `effectiveStabilizeCost`, `ClosePortalWithLabEnergy`, `StabilizePortalWithLabEnergy` | GREEN | only the two costs become zero in half-open active window; expiry restores ordinary costs and failed commands leave regenerated energy uncommitted |
| EMERGENCY-004 | Extraction stays 30 | INV | `TestLabState_ExtractionCostDuringOverrideIsThirty`, `TestOpenExtractionPortal_ChargesThirtyDuringOverride`, `TestOpenExtractionPortal_PreservesOverrideDeadline` | `cfg.ExtractionCost`, `LabState.SpendEnergy`, `OpenExtractionPortal`, `LabState.LeylineOverrideActive` | GREEN | real Extraction opening receives no Override discount and does not move its deadline |
| EMERGENCY-005 | Regen continues | BEH | `TestLabState_LabEnergyRegeneratesDuringOverride`, `TestLabState_OverrideRegenerationUsesCompletedWholeSeconds`, `TestLabState_OverrideRegenerationCapsAtMaximum`, `TestLabState_FreeCloseDoesNotInterruptRegeneration`, `TestLabState_FreeStabilizeDoesNotInterruptRegeneration` | `LabState.CurrentEnergy`, `effectiveCloseCost`, `effectiveStabilizeCost` | GREEN | Override reuses normal capped integer regeneration; free actions never re-baseline it |
| EMERGENCY-006 | New collapse resets | BEH | `TestLabState_SecondCollapseResetsRegeneratedEnergyToZero`, `TestLabState_SecondCollapseRebasesAtSecondCollapse`, `TestLabState_SecondCollapseReplacesDeadline`, `TestLabState_SecondCollapseDoesNotExtendFromOldDeadline`, `TestResolvePortalLifecycleWithLabEmergency_SecondCollapseMayUseDifferentCause`, `TestLabState_OutOfOrderCollapseRejectsWithoutMutation`, `TestResolvePortalLifecycleWithLabEmergency_TwoPortalsResetTwice`, `TestResolvePortalLifecycleWithLabEmergency_ReplayOfSecondPortalIsIdempotent` | `LabState.ActivateLeylineOverride`, `ResolvePortalLifecycleWithLabEmergency` | GREEN | each chronologically new Collapse re-baselines to zero and replaces deadline; terminal replay is idempotent |

## SPAWN

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| SPAWN-001 | Inclusive random delay 0..20 sec | BAL | `TestNewNaturalSpawnState_AcceptsZeroDelay`, `TestNewNaturalSpawnState_AcceptsMaximumDelay`, `TestNewNaturalSpawnState_DrawsExactlyOnce` | `NewNaturalSpawnState` | GREEN | inclusive whole-second scheduler draw from config |
| SPAWN-002 | Fresh delay after successful spawn | BEH | `TestResolveNaturalSpawn_SchedulesNextDelayAfterSuccess`, `TestResolveNaturalSpawn_NextScheduleUsesSpawnTick`, `TestResolveNaturalSpawn_SeventhPortalPausesWithoutNextDelayDraw` | `ResolveNaturalSpawn`, `NewNaturalSpawnState` | GREEN | successful non-capacity spawn draws one fresh delay from the spawn tick; reaching 7/7 pauses instead |
| SPAWN-003 | 7/7 pause; free Slot restarts delay | BEH | `TestResolveNaturalSpawn_FullCapacityPausesImmediately`, `TestResolveNaturalSpawn_FullCapacityConsumesNoRandom`, `TestResolveNaturalSpawn_FirstTickAfterFreeOnlySchedules`, `TestResolveNaturalSpawn_FirstTickAfterFreeDoesNotSpawnAtZeroDelay`, `TestResolveNaturalSpawn_SeventhPortalPausesWithoutNextDelayDraw` | `ResolveNaturalSpawn`, `resolveNaturalSpawnPrepared` | GREEN | full capacity discards the timer; first later free-capacity tick draws a fresh delay without spawning |
| SPAWN-004 | Random destination among 85 Planes | BEH | `TestResolveNaturalSpawn_SelectsFirstPlaneIndex`, `TestResolveNaturalSpawn_SelectsLastPlaneIndex`, `TestResolveNaturalSpawn_DestinationDrawPrecedesFactoryDraws` | `ResolveNaturalSpawn` | GREEN | destination index is drawn uniformly through `Random.IntInclusive(0, 84)` before factory draws |
| SPAWN-005 | Repeated Plane destinations allowed | BEH | `TestResolveNaturalSpawn_AllowsRepeatedDestination` | `ResolveNaturalSpawn` | GREEN | destination selection does not exclude Planes already targeted by OPEN Portals |
| SPAWN-006 | At most one spawn per tick | BEH | `TestResolveNaturalSpawn_LateTickSpawnsOnlyOnePortal`, `TestResolveNaturalSpawn_NextZeroDelayDoesNotSpawnTwice`, `TestSimulationTick_ZeroDelayProducesAtMostOneSpawn`, `TestSimulationTick_SevenOpenPortalsNeverBecomeEight`, `TestSimulationTick_LargeJumpDoesNotBackfillNaturalSpawns` | `naturalSpawnDue`, `ResolveNaturalSpawn`, `SimulationState.ResolveTick` | GREEN | one tick creates at most one Portal even after a large jump; zero-delay schedules require a strictly later tick and capacity never exceeds seven OPEN |

## SIMULATION

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| SIMULATION-001 | One-second supplied-time domain step | BEH | `TestSimulationState_InvalidAggregateIsAtomic`, `TestSimulationTick_FullOrderPortalObserverExtractionSpawnAttention`, `TestSimulationTick_LargeJumpDoesNotBackfillNaturalSpawns`, `TestSimulationTick_ResultIsDerivedFromCommittedState` | `SimulationState.ResolveTick` | PARTIAL | complete deterministic supplied-time domain step exists; recurring one-second ticker ownership remains Stage 11 |
| SIMULATION-002 | Ordered tick transition stages | BEH | `TestSimulationTick_FullOrderPortalObserverExtractionSpawnAttention`, `TestSimulationTick_PortalLifecycleRunsBeforeObserverLifecycle`, `TestSimulationTick_ResearchCompletionFeedsSameTickExtraction`, `TestSimulationTick_RandomOrderExtractionBeforeNatural`, `TestSimulationTick_NeedsAttentionUsesPostSpawnState`, `TestSimulationTick_DoesNotReorderAnyAggregateSlice` | `resolvePortalStage`, `resolveObserverStage`, `resolveExtractionStage`, `resolveNaturalSpawnPrepared`, `deriveSimulationTickResult` | GREEN | Portal → Observer-by-ID → Extraction-by-ID → Natural → post-spawn Attention order is deterministic without reordering stored slices |
| SIMULATION-003 | Atomic, monotonic, idempotent tick | INV | `TestSimulationTick_SameTimestampReplayIsNoOp`, `TestSimulationTick_SameTimestampReplayConsumesNoRandom`, `TestSimulationTick_BackwardTimeRejectsAtomically`, `TestSimulationTick_RejectsInvalidRequiredConfig`, `TestSimulationTick_InvalidConfigConsumesNoRandom`, `TestSimulationTick_InvalidStateConsumesNoRandom`, `TestSimulationTick_InvalidStateDoesNotPartiallyResolvePortal`, `TestSimulationTick_InvalidStateDoesNotPartiallyResolveObserver`, `TestSimulationTick_InvalidStateDoesNotPartiallySynchronizeExtraction`, `TestSimulationTick_InvalidStateDoesNotPartiallySpawn` | `validSimulationConfig`, `validateSimulationState`, `SimulationState.ResolveTick`, `cloneSimulationState` | GREEN | complete config/aggregate preflight and copy-then-commit make failures atomic and draw-free; equal timestamps replay as no-op and reverse time is rejected |
| SIMULATION-004 | Realtime values stay derived | INV | `TestSimulationTick_DerivedPortalEnergyIsNotPersisted`, `TestSimulationTick_DerivedCreaturesAreNotPersisted`, `TestSimulationTick_DerivedLabEnergyIsNotRebased`, `TestSimulationTick_DoesNotPersistDerivedRealtimeValues`, `TestSimulationTick_ResultIsDerivedFromCommittedState` | `SimulationState.ResolveTick`, `deriveSimulationTickResult` | GREEN | Portal Energy, creatures, Lab Energy and Attention remain derived from committed baselines/state |

## ATTENTION

| ID | Rule | Type | Test | Implementation | Status | Notes |
|---|---|---|---|---|---|---|
| ATTENTION-001 | Highest risk_score | BEH | `TestNeedsAttentionPortalIndex_SelectsHighestRiskScore`, `TestNeedsAttentionPortalIndex_RiskScoreBeatsInstabilityTieBreaker`, `TestNeedsAttentionPortalIndex_UsesCurrentTimeDerivedRisk`, `TestSimulationTick_NeedsAttentionUsesPostSpawnState`, `TestSimulationTick_ResultIsDerivedFromCommittedState` | `NeedsAttentionPortalIndex`, `attentionBefore`, `deriveSimulationTickResult` | GREEN | current derived score is the first comparison and tick selection uses final post-spawn committed state |
| ATTENTION-002 | Risk tie prefers UNSTABLE | BEH | `TestNeedsAttentionPortalIndex_ExactRiskTiePrefersUnstable` | `attentionBefore` | GREEN | applies only after exact numeric score tie |
| ATTENTION-003 | Then lower effective_lifetime | BEH | `TestNeedsAttentionPortalIndex_ExactRiskAndStabilityTiePrefersLowerLifetime` | `attentionBefore` | GREEN | current derived lifetime |
| ATTENTION-004 | Then older opened_at | BEH | `TestNeedsAttentionPortalIndex_LifetimeTiePrefersOlderOpening` | `attentionBefore` | GREEN | stable timestamp comparison |
| ATTENTION-005 | OPEN candidates only | INV | `TestNeedsAttentionPortalIndex_ReturnsNoneWithoutOpenPortals`, `TestNeedsAttentionPortalIndex_IgnoresClosedPortal`, `TestNeedsAttentionPortalIndex_IgnoresCollapsedPortal` | `NeedsAttentionPortalIndex` | GREEN | terminal portals are excluded |
| ATTENTION-006 | Complete tie uses lower Portal ID | INV | `TestNeedsAttentionPortalIndex_CompleteTiePrefersLowerPortalID` | `attentionBefore` | GREEN | final deterministic tie only |
| ATTENTION-007 | No sorting or mutation | INV | `TestNeedsAttentionPortalIndex_ReturnsOriginalSliceIndex`, `TestNeedsAttentionPortalIndex_DoesNotReorderInput`, `TestNeedsAttentionPortalIndex_DoesNotMutatePortals` | `NeedsAttentionPortalIndex` | GREEN | returns original index and leaves slice unchanged |

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
