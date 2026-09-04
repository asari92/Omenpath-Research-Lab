# План реализации corrective pass Stage 8

> **Для agentic workers:** обязательный sub-skill — `superpowers:subagent-driven-development` (рекомендуется) или `superpowers:executing-plans`. Выполнять задачи последовательно, отмечая checkbox.

**Цель:** исправить три подтверждённых дефекта интеграции Stage 8, синхронизировать документы и остановиться до Stage 9.

**Архитектура:** публичные domain API и порядок стадий тика сохраняются. Observer stage отдельно сводит самый ранний успешный return для каждого ранее UNEXPLORED Plane; Simulation использует внутренние prepared-path функции Extraction после aggregate preflight; validation расширяется в существующем `simulation.go` без persistence/manager зависимостей.

**Стек:** Go, стандартный `testing`, `testify/require`, существующие `FakeRandom` и domain fixtures, Git TDD history.

---

## Карта файлов

- Изменить `internal/domain/simulation.go`: exploration reduction,
  prepared Extraction orchestration и полная aggregate validation.
- Изменить `internal/domain/extraction.go`: общий strict/prepared
  synchronization core.
- Изменить `internal/domain/observer_commands.go`: выделить
  `recallObserverPrepared`, сохранив публичный command admission.
- Изменить `internal/domain/simulation_observer_test.go`: chronological
  exploration regression.
- Изменить `internal/domain/simulation_extraction_test.go`: late
  multi-Extraction regressions.
- Изменить `internal/domain/simulation_state_test.go`: Plane, scheduler,
  Portal и Observer invariant regressions.
- Изменить `10_STAGE_08_SIMULATION_TDD.md`: исправленная семантика и evidence.
- Изменить `docs/requirements.md`: Extraction eligibility и известные
  Recommendation contracts.
- Изменить `docs/traceability.md`: новые tests/status notes.
- Изменить `01_AI_WORKLOG_CURRENT.md`: corrective audit и RED/GREEN history.
- Изменить
  `docs/superpowers/specs/2026-09-04-stage8-corrective-design.md`: только
  русский перевод без изменения утверждённого дизайна.

### Task 1: Зафиксировать chronological Plane exploration

**Файлы:**

- Изменить: `internal/domain/simulation_observer_test.go:158`
- Изменить: `internal/domain/simulation.go:154`
- Изменить: `10_STAGE_08_SIMULATION_TDD.md:445`

- [ ] **Шаг 1: заменить ошибочную expectation новым regression test**

Переименовать существующий тест и ожидать самый ранний return:

```go
func TestSimulationTick_EarliestSuccessfulReturnSetsExploredAtRegardlessOfObserverID(t *testing.T) {
	p := observerPortal()
	p.ObserverFlow = domain.PortalFlowInbound
	lowerIDLater := tickReturningObserver(1, 7*time.Second)
	higherIDEarlier := tickReturningObserver(2, 5*time.Second)
	state := observerTickState(p, lowerIDLater, higherIDEarlier)
	state.Observers[0], state.Observers[1] = state.Observers[1], state.Observers[0]

	_, err := state.ResolveTick(testutil.BaseTime.Add(7*time.Second), nil, config.Default())

	require.NoError(t, err)
	require.True(t, state.Planes[0].Explored)
	require.NotNil(t, state.Planes[0].ExploredAt)
	require.Equal(t, testutil.BaseTime.Add(5*time.Second), *state.Planes[0].ExploredAt)
	require.Equal(t, domain.ObserverAvailable, state.Observers[0].Status)
	require.Equal(t, domain.ObserverAvailable, state.Observers[1].Status)
}
```

- [ ] **Шаг 2: подтвердить RED**

Запустить:

```bash
go test -count=1 -run TestSimulationTick_EarliestSuccessfulReturnSetsExploredAtRegardlessOfObserverID ./internal/domain
```

Ожидание: FAIL, actual `ExploredAt = BaseTime+7s`, expected
`BaseTime+5s`.

- [ ] **Шаг 3: закоммитить RED**

```bash
git add internal/domain/simulation_observer_test.go
git commit -m "test(stage8): RED earliest plane exploration timestamp"
```

- [ ] **Шаг 4: реализовать минимальный reducer в Observer stage**

До обхода запомнить Plane, которые были UNEXPLORED. Перед каждым resolver
сохранить исходные status, Plane ID и return deadline. После успешного
`RETURNING -> AVAILABLE` собрать минимум, затем после обхода заменить
установленный первым Observer timestamp:

```go
initiallyUnexplored := make(map[int64]bool, len(state.Planes))
earliestReturnByPlane := make(map[int64]time.Time)
for i := range state.Planes {
	initiallyUnexplored[state.Planes[i].ID] = !state.Planes[i].Explored
}

// Внутри существующего цикла перед ResolveObserverLifecycle:
beforeStatus := observer.Status
var returnPlaneID int64
var returnedAt time.Time
hasReturnCandidate := beforeStatus == ObserverReturning &&
	observer.CurrentPlaneID != nil && observer.PhaseEndsAt != nil
if hasReturnCandidate {
	returnPlaneID = *observer.CurrentPlaneID
	returnedAt = *observer.PhaseEndsAt
}

// Сразу после успешного ResolveObserverLifecycle:
if hasReturnCandidate && observer.Status == ObserverAvailable &&
	initiallyUnexplored[returnPlaneID] {
	current, exists := earliestReturnByPlane[returnPlaneID]
	if !exists || returnedAt.Before(current) {
		earliestReturnByPlane[returnPlaneID] = returnedAt
	}
}

// После обхода Observers:
for planeID, exploredAt := range earliestReturnByPlane {
	index := planeIndexByID[planeID]
	state.Planes[index].Explored = true
	at := exploredAt
	state.Planes[index].ExploredAt = &at
}
```

Не менять `ResolveObserverLifecycle`: standalone Stage 3 contract повторного
return остаётся прежним.

- [ ] **Шаг 5: подтвердить GREEN и отсутствие регрессии**

```bash
go test -count=1 -run 'TestSimulationTick_(EarliestSuccessfulReturnSetsExploredAtRegardlessOfObserverID|RepeatReturnPreservesExploredAt|SuccessfulReturnExploresPlane|ObserverTraversalDoesNotReorderSlice)' ./internal/domain
go test -count=1 ./internal/domain
```

Ожидание: оба запуска PASS.

- [ ] **Шаг 6: закоммитить GREEN**

```bash
git add internal/domain/simulation.go
git commit -m "fix(stage8): GREEN chronological plane exploration"
```

### Task 2: Исправить несколько поздних Extraction sync

**Файлы:**

- Изменить: `internal/domain/simulation_extraction_test.go:116`
- Изменить: `internal/domain/extraction.go:139`
- Изменить: `internal/domain/observer_commands.go:133`
- Изменить: `internal/domain/simulation.go:206`

- [ ] **Шаг 1: добавить failing integration test**

```go
func TestSimulationTick_MultipleLateExtractionsDoNotRejectDueTransitCreatedInSameStage(t *testing.T) {
	state := extractionTickState(
		[]domain.Portal{
			tickExtractionPortal(1, 1, 1, testutil.BaseTime),
			tickExtractionPortal(2, 1, 2, testutil.BaseTime),
		},
		tickWaitingObserver(1, -time.Minute),
		tickWaitingObserver(2, -30*time.Second),
	)
	rnd := testutil.NewFakeRandom().QueueInt(5, 6)

	_, err := state.ResolveTick(testutil.BaseTime.Add(20*time.Second), rnd, config.Default())

	require.NoError(t, err)
	require.Equal(t, domain.ObserverReturning, state.Observers[0].Status)
	require.Equal(t, domain.ObserverReturning, state.Observers[1].Status)
	require.Equal(t, int64(1), *state.Observers[0].ActivePortalID)
	require.Equal(t, int64(2), *state.Observers[1].ActivePortalID)
	require.Equal(t, testutil.BaseTime.Add(10*time.Second), *state.Observers[0].PhaseEndsAt)
	require.Equal(t, testutil.BaseTime.Add(11*time.Second), *state.Observers[1].PhaseEndsAt)
}
```

- [ ] **Шаг 2: подтвердить RED**

```bash
go test -count=1 -run TestSimulationTick_MultipleLateExtractionsDoNotRejectDueTransitCreatedInSameStage ./internal/domain
```

Ожидание: FAIL с `ErrObserverInvariant`; исходный `SimulationState`
остаётся неизменным из-за copy-then-commit.

- [ ] **Шаг 3: закоммитить RED**

```bash
git add internal/domain/simulation_extraction_test.go
git commit -m "test(stage8): RED multiple late extraction sync"
```

- [ ] **Шаг 4: выделить prepared recall**

Оставить aggregate validation в публичном `RecallObserver`, а существующее
поведение после неё перенести без изменения порядка проверок:

```go
func RecallObserver(
	portal *Portal,
	plane *Plane,
	observers []Observer,
	now time.Time,
	confirmUnstable bool,
	rnd random.Random,
	cfg config.Config,
) (observerID int64, err error) {
	if err := validateObserverCommandAggregate(portal, plane, observers, now); err != nil {
		return 0, err
	}
	return recallObserverPrepared(portal, plane, observers, now, confirmUnstable, rnd, cfg)
}

func recallObserverPrepared(
	portal *Portal,
	plane *Plane,
	observers []Observer,
	now time.Time,
	confirmUnstable bool,
	rnd random.Random,
	cfg config.Config,
) (observerID int64, err error) {
	if portal.Status != PortalStatusOpen {
		return 0, ErrPortalNotOpen
	}
	if portal.Kind == PortalKindExtraction && portal.ExtractionSynchronizedAt == nil {
		return 0, ErrExtractionSynchronizing
	}
	risk, _ := portal.RiskLevel(now, cfg)
	if risk == RiskCritical {
		return 0, ErrPortalCriticalRisk
	}
	if portal.ObserverFlow == PortalFlowOutbound {
		return 0, ErrPortalDirectionConflict
	}
	if portal.CreaturesInside(now, cfg) > 0 {
		return 0, ErrPortalCreaturesPresent
	}
	_, busy, err := ActiveTransitObserverIndex(observers, portal.ID, now)
	if err != nil {
		return 0, err
	}
	if busy {
		return 0, ErrPortalBusy
	}
	index, ok, err := LongestWaitingObserverIndex(observers, portal.DestinationPlaneID)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrNoWaitingObserver
	}
	if portal.Stability == PortalUnstable && !confirmUnstable {
		return 0, ErrConfirmationRequired
	}
	if err = observers[index].StartReturning(now, portal.ID, rnd, cfg); err != nil {
		return 0, err
	}
	if portal.ObserverFlow == PortalFlowNone {
		portal.ObserverFlow = PortalFlowInbound
		portal.UpdatedAt = now
	}
	return observers[index].ID, nil
}
```

- [ ] **Шаг 5: выделить strict/prepared Extraction core**

```go
func ResolveExtractionSynchronization(
	portal *Portal,
	plane *Plane,
	observers []Observer,
	now time.Time,
	rnd random.Random,
	cfg config.Config,
) (int64, bool, error) {
	return resolveExtractionSynchronization(portal, plane, observers, now, rnd, cfg, true)
}

func resolveExtractionSynchronizationPrepared(
	portal *Portal,
	plane *Plane,
	observers []Observer,
	now time.Time,
	rnd random.Random,
	cfg config.Config,
) (int64, bool, error) {
	return resolveExtractionSynchronization(portal, plane, observers, now, rnd, cfg, false)
}
```

В общем core сохранить текущие preliminary/terminal/marker/deadline checks.
Выполнять `validateObserverRoster(observers, now)` только при
`validateRoster == true`. После установки marker вызывать
`recallObserverPrepared(..., syncAt, ...)`, а не публичный command wrapper.

- [ ] **Шаг 6: переключить только Simulation на prepared-path**

В `resolveExtractionStage` заменить вызов на:

```go
if _, _, err := resolveExtractionSynchronizationPrepared(
	portal,
	&state.Planes[planeIndex],
	state.Observers,
	now,
	rnd,
	cfg,
); err != nil {
	return err
}
```

- [ ] **Шаг 7: подтвердить GREEN и неизменность Stage 4/7 API**

```bash
go test -count=1 -run 'TestSimulationTick_(MultipleLateExtractionsDoNotRejectDueTransitCreatedInSameStage|MultipleExtractionsVisitPortalIDOrder|CompletedExtractionDoesNotReturnSecondObserver)' ./internal/domain
go test -count=1 -run 'Test(RecallObserver|ResolveExtractionSynchronization|Extraction)' ./internal/domain
go test -count=1 ./internal/domain
```

Ожидание: все запуски PASS; strict public stale-transit tests остаются GREEN.

- [ ] **Шаг 8: закоммитить GREEN**

```bash
git add internal/domain/extraction.go internal/domain/observer_commands.go internal/domain/simulation.go
git commit -m "fix(stage8): GREEN late extraction orchestration"
```

### Task 3: Закрыть Plane и scheduler invariants

**Файлы:**

- Изменить: `internal/domain/simulation_state_test.go`
- Изменить: `internal/domain/simulation.go:251`

- [ ] **Шаг 1: добавить RED tests**

Добавить table test для `Explored/ExploredAt`:

```go
func TestSimulationState_RejectsNonCanonicalPlaneExploration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.Plane)
	}{
		{"explored without timestamp", func(p *domain.Plane) { p.Explored = true }},
		{"unexplored with timestamp", func(p *domain.Plane) {
			at := testutil.BaseTime
			p.ExploredAt = &at
		}},
		{"future exploration", func(p *domain.Plane) {
			at := testutil.BaseTime.Add(time.Second)
			p.Explored = true
			p.ExploredAt = &at
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := simulationState(testutil.BaseTime)
			tt.mutate(&state.Planes[0])
			before := state
			rnd := &countingRandom{}
			err := resolveStateForValidation(&state, testutil.BaseTime, rnd, config.Default())
			require.ErrorIs(t, err, domain.ErrSimulationInvariant)
			require.Equal(t, before, state)
			require.Zero(t, rnd.intCalls)
			require.Zero(t, rnd.floatCalls)
		})
	}
}
```

Добавить scheduler test:

```go
func TestSimulationState_RejectsFutureSpawnScheduleOrigin(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime.Add(time.Second), time.Second)
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}
```

- [ ] **Шаг 2: подтвердить RED и закоммитить**

```bash
go test -count=1 -run 'TestSimulationState_Rejects(NonCanonicalPlaneExploration|FutureSpawnScheduleOrigin)' ./internal/domain
git add internal/domain/simulation_state_test.go
git commit -m "test(stage8): RED plane and scheduler invariants"
```

- [ ] **Шаг 3: реализовать minimal validation**

Перед заполнением `planeIDs`:

```go
for _, plane := range state.Planes {
	if plane.ID <= 0 ||
		plane.Explored != (plane.ExploredAt != nil) ||
		(plane.ExploredAt != nil && plane.ExploredAt.After(now)) {
		return ErrSimulationInvariant
	}
	// Существующая duplicate-ID проверка остаётся.
}
```

Изменить scheduler helper:

```go
func validNaturalSpawnState(spawn NaturalSpawnState, now time.Time) bool {
	if spawn.Paused {
		return spawn.ScheduledAt == nil && spawn.DueAt == nil
	}
	return spawn.ScheduledAt != nil && spawn.DueAt != nil &&
		!spawn.ScheduledAt.After(now) &&
		!spawn.DueAt.Before(*spawn.ScheduledAt)
}
```

- [ ] **Шаг 4: подтвердить GREEN и закоммитить**

```bash
go test -count=1 -run 'TestSimulationState_|TestResolveNaturalSpawn_' ./internal/domain
git add internal/domain/simulation.go
git commit -m "fix(stage8): GREEN plane and scheduler invariants"
```

### Task 4: Закрыть Portal aggregate invariants

**Файлы:**

- Изменить: `internal/domain/simulation_state_test.go`
- Изменить: `internal/domain/simulation.go:277`

- [ ] **Шаг 1: добавить table-driven RED test**

Добавить helper для terminal fixtures и полный набор malformed cases:

```go
func makeTerminalPortal(
	p *domain.Portal,
	status domain.PortalStatus,
	reason domain.TerminationReason,
	closedAt time.Time,
) {
	p.Status = status
	p.TerminationReason = reason
	p.ClosedAt = &closedAt
	p.UpdatedAt = closedAt
}

func TestSimulationState_RejectsNonCanonicalPortalFields(t *testing.T) {
	cfg := config.Default()
	now := testutil.BaseTime.Add(10 * time.Second)
	tests := []struct {
		name   string
		mutate func(*domain.Portal)
	}{
		{"terminal slot outside range", func(p *domain.Portal) {
			makeTerminalPortal(p, domain.PortalStatusClosed, domain.TerminationManualClose, testutil.BaseTime.Add(time.Second))
			p.SlotIndex = 0
		}},
		{"non-positive lifecycle", func(p *domain.Portal) {
			p.ScheduledCloseAt = p.OpenedAt
		}},
		{"negative energy", func(p *domain.Portal) { p.EnergyBase = -1 }},
		{"energy above one hundred", func(p *domain.Portal) { p.EnergyBase = 101 }},
		{"non-positive decay", func(p *domain.Portal) { p.EnergyDecayRate = 0 }},
		{"negative creatures", func(p *domain.Portal) { p.CreaturesInitial = -1 }},
		{"creatures above configured maximum", func(p *domain.Portal) {
			p.CreaturesInitial = cfg.CreatureMax + 1
		}},
		{"stable with hidden collapse", func(p *domain.Portal) {
			at := p.OpenedAt.Add(6 * time.Second)
			p.InstabilityCollapseAt = &at
		}},
		{"unstable without hidden collapse", func(p *domain.Portal) {
			p.Stability = domain.PortalUnstable
		}},
		{"unstable hidden collapse before legal window", func(p *domain.Portal) {
			p.Stability = domain.PortalUnstable
			at := p.OpenedAt.Add(cfg.InstabilityMinLifetime - time.Nanosecond)
			p.InstabilityCollapseAt = &at
		}},
		{"natural with extraction marker", func(p *domain.Portal) {
			at := p.OpenedAt.Add(cfg.ExtractionSync)
			p.ExtractionSynchronizedAt = &at
		}},
		{"extraction with outbound flow", func(p *domain.Portal) {
			p.Kind = domain.PortalKindExtraction
			p.ObserverFlow = domain.PortalFlowOutbound
		}},
		{"extraction with creatures", func(p *domain.Portal) {
			p.Kind = domain.PortalKindExtraction
			p.ObserverFlow = domain.PortalFlowInbound
			p.CreaturesInitial = 1
		}},
		{"extraction unstable", func(p *domain.Portal) {
			p.Kind = domain.PortalKindExtraction
			p.ObserverFlow = domain.PortalFlowInbound
			p.Stability = domain.PortalUnstable
			at := p.OpenedAt.Add(6 * time.Second)
			p.InstabilityCollapseAt = &at
		}},
		{"extraction marker differs from deadline", func(p *domain.Portal) {
			p.Kind = domain.PortalKindExtraction
			p.ObserverFlow = domain.PortalFlowInbound
			at := p.OpenedAt.Add(cfg.ExtractionSync + time.Second)
			p.ExtractionSynchronizedAt = &at
		}},
		{"created after opening", func(p *domain.Portal) {
			p.CreatedAt = p.OpenedAt.Add(time.Second)
		}},
		{"energy baseline before opening", func(p *domain.Portal) {
			p.EnergyBaseAt = p.OpenedAt.Add(-time.Second)
		}},
		{"energy baseline after now", func(p *domain.Portal) {
			p.EnergyBaseAt = now.Add(time.Second)
		}},
		{"update before creation", func(p *domain.Portal) {
			p.UpdatedAt = p.CreatedAt.Add(-time.Second)
		}},
		{"update after now", func(p *domain.Portal) {
			p.UpdatedAt = now.Add(time.Second)
		}},
		{"closed before opening", func(p *domain.Portal) {
			makeTerminalPortal(p, domain.PortalStatusClosed, domain.TerminationManualClose, p.OpenedAt.Add(-time.Second))
		}},
		{"closed after now", func(p *domain.Portal) {
			makeTerminalPortal(p, domain.PortalStatusClosed, domain.TerminationManualClose, now.Add(time.Second))
		}},
		{"natural close with wrong semantic timestamp", func(p *domain.Portal) {
			makeTerminalPortal(p, domain.PortalStatusClosed, domain.TerminationNaturalClose, p.OpenedAt.Add(5*time.Second))
		}},
		{"energy collapse with wrong semantic timestamp", func(p *domain.Portal) {
			makeTerminalPortal(p, domain.PortalStatusCollapsed, domain.TerminationEnergyDepleted, p.OpenedAt.Add(5*time.Second))
		}},
		{"instability collapse with wrong semantic timestamp", func(p *domain.Portal) {
			p.Stability = domain.PortalUnstable
			hidden := p.OpenedAt.Add(6 * time.Second)
			p.InstabilityCollapseAt = &hidden
			makeTerminalPortal(p, domain.PortalStatusCollapsed, domain.TerminationInstability, hidden.Add(time.Second))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := simulationState(now)
			p := tickNaturalClosePortal(1, 1, time.Minute)
			tt.mutate(&p)
			state.Portals = []domain.Portal{p}
			state.NextPortalID = 2
			before := state
			rnd := &countingRandom{}
			err := resolveStateForValidation(&state, now, rnd, cfg)
			require.ErrorIs(t, err, domain.ErrSimulationInvariant)
			require.Equal(t, before, state)
			require.Zero(t, rnd.intCalls)
			require.Zero(t, rnd.floatCalls)
		})
	}
}
```

- [ ] **Шаг 2: подтвердить RED и закоммитить**

```bash
go test -count=1 -run TestSimulationState_RejectsNonCanonicalPortalFields ./internal/domain
git add internal/domain/simulation_state_test.go
git commit -m "test(stage8): RED canonical portal aggregate"
```

- [ ] **Шаг 3: расширить `validSimulationPortal`**

Добавить `math` в imports `simulation.go`, передать `now/cfg` из
`validateSimulationState` и реализовать:

```go
func validSimulationPortal(portal *Portal, now time.Time, cfg config.Config) bool {
	if portal == nil ||
		portal.SlotIndex < 1 || portal.SlotIndex > cfg.MaxActivePortals ||
		portal.OpenedAt.IsZero() || portal.CreatedAt.IsZero() ||
		portal.EnergyBaseAt.IsZero() || portal.UpdatedAt.IsZero() ||
		portal.OpenedAt.After(now) ||
		!portal.ScheduledCloseAt.After(portal.OpenedAt) ||
		portal.CreatedAt.After(portal.OpenedAt) ||
		portal.EnergyBaseAt.Before(portal.OpenedAt) ||
		portal.EnergyBaseAt.After(portal.UpdatedAt) ||
		portal.EnergyBaseAt.After(now) ||
		portal.UpdatedAt.Before(portal.OpenedAt) ||
		portal.UpdatedAt.Before(portal.CreatedAt) ||
		portal.UpdatedAt.After(now) ||
		math.IsNaN(portal.EnergyBase) || math.IsInf(portal.EnergyBase, 0) ||
		math.IsNaN(portal.EnergyDecayRate) || math.IsInf(portal.EnergyDecayRate, 0) ||
		portal.EnergyBase < 0 ||
		portal.EnergyDecayRate <= 0 ||
		portal.CreaturesInitial < 0 || portal.CreaturesInitial > cfg.CreatureMax {
		return false
	}
	maxEnergy := math.Max(
		math.Max(cfg.PortalEnergyMax, cfg.ExtractionEnergyMax),
		cfg.StabilizeMaxStartEnergy+cfg.StabilizeBoost,
	)
	if portal.EnergyBase > maxEnergy {
		return false
	}
	if portal.ObserverFlow != PortalFlowNone &&
		portal.ObserverFlow != PortalFlowOutbound &&
		portal.ObserverFlow != PortalFlowInbound {
		return false
	}

	switch portal.Stability {
	case PortalStable:
		if portal.InstabilityCollapseAt != nil {
			return false
		}
	case PortalUnstable:
		if portal.InstabilityCollapseAt == nil ||
			portal.InstabilityCollapseAt.Before(
				portal.OpenedAt.Add(cfg.InstabilityMinLifetime),
			) ||
			!portal.InstabilityCollapseAt.Before(portal.ScheduledCloseAt) {
			return false
		}
	default:
		return false
	}

	switch portal.Kind {
	case PortalKindNatural:
		if portal.ExtractionSynchronizedAt != nil {
			return false
		}
	case PortalKindExtraction:
		if portal.Stability != PortalStable ||
			portal.ObserverFlow != PortalFlowInbound ||
			portal.CreaturesInitial != 0 {
			return false
		}
		if portal.ExtractionSynchronizedAt != nil {
			syncAt := portal.OpenedAt.Add(cfg.ExtractionSync)
			if !portal.ExtractionSynchronizedAt.Equal(syncAt) ||
				portal.ExtractionSynchronizedAt.After(now) {
				return false
			}
		}
	default:
		return false
	}

	switch portal.Status {
	case PortalStatusOpen:
		return portal.ClosedAt == nil &&
			portal.TerminationReason == TerminationNone
	case PortalStatusClosed, PortalStatusCollapsed:
		return validSimulationTerminalPortal(portal, now)
	default:
		return false
	}
}

func validSimulationTerminalPortal(portal *Portal, now time.Time) bool {
	if portal.ClosedAt == nil ||
		portal.ClosedAt.Before(portal.OpenedAt) ||
		portal.ClosedAt.After(now) ||
		portal.UpdatedAt.Before(*portal.ClosedAt) ||
		portal.EnergyBaseAt.After(*portal.ClosedAt) ||
		(portal.ExtractionSynchronizedAt != nil &&
			portal.ExtractionSynchronizedAt.After(*portal.ClosedAt)) {
		return false
	}
	if portal.Status == PortalStatusClosed &&
		portal.TerminationReason == TerminationManualClose {
		return true
	}
	if portal.TerminationReason == TerminationManualClose ||
		portal.TerminationReason == TerminationNone {
		return false
	}

	candidate := *portal
	candidate.Status = PortalStatusOpen
	candidate.TerminationReason = TerminationNone
	candidate.ClosedAt = nil
	changed, err := candidate.ResolveLifecycle(*portal.ClosedAt)
	return err == nil && changed &&
		candidate.Status == portal.Status &&
		candidate.TerminationReason == portal.TerminationReason &&
		candidate.ClosedAt != nil &&
		candidate.ClosedAt.Equal(*portal.ClosedAt)
}
```

- [ ] **Шаг 4: привести старую synthetic fixture скрытого Collapse к canonical окну**

В `simulation_portal_lifecycle_test.go` заменить synthetic hidden collapse
`+2s` на `+5s` или позже там, где тест проходит через aggregate validation.
Ожидаемый lifecycle outcome не менять.

- [ ] **Шаг 5: подтвердить GREEN и закоммитить**

```bash
go test -count=1 -run 'TestSimulationState_|TestSimulationTick_(Resolves|MultipleCollapses|Instability|Energy|Natural)' ./internal/domain
go test -count=1 ./internal/domain
git add internal/domain/simulation.go internal/domain/simulation_state_test.go internal/domain/simulation_portal_lifecycle_test.go
git commit -m "fix(stage8): GREEN canonical portal aggregate"
```

### Task 5: Закрыть Observer aggregate invariants

**Файлы:**

- Изменить: `internal/domain/simulation_state_test.go`
- Изменить: `internal/domain/simulation.go:368`

- [ ] **Шаг 1: добавить RED tests**

Добавить полный table test. Каждый case проверяет `ErrSimulationInvariant`,
отсутствие мутации и нулевой random consumption:

```go
func TestSimulationState_RejectsNonCanonicalObserverFields(t *testing.T) {
	now := testutil.BaseTime.Add(5 * time.Second)
	tests := []struct {
		name     string
		observer func() domain.Observer
		flow     domain.PortalFlow
		mutate   func(*domain.Observer)
	}{
		{"created after now", func() domain.Observer {
			return tickOutboundObserver(1, 10*time.Second)
		}, domain.PortalFlowOutbound, func(o *domain.Observer) {
			o.CreatedAt = now.Add(time.Second)
		}},
		{"updated before creation", func() domain.Observer {
			return tickOutboundObserver(1, 10*time.Second)
		}, domain.PortalFlowOutbound, func(o *domain.Observer) {
			o.UpdatedAt = o.CreatedAt.Add(-time.Second)
		}},
		{"updated after now", func() domain.Observer {
			return tickOutboundObserver(1, 10*time.Second)
		}, domain.PortalFlowOutbound, func(o *domain.Observer) {
			o.UpdatedAt = now.Add(time.Second)
		}},
		{"phase starts after now", func() domain.Observer {
			return tickOutboundObserver(1, 10*time.Second)
		}, domain.PortalFlowOutbound, func(o *domain.Observer) {
			start := now.Add(time.Second)
			end := start.Add(time.Second)
			o.PhaseStartedAt = &start
			o.PhaseEndsAt = &end
		}},
		{"outbound through none flow", func() domain.Observer {
			return tickOutboundObserver(1, 10*time.Second)
		}, domain.PortalFlowNone, func(*domain.Observer) {}},
		{"outbound through inbound flow", func() domain.Observer {
			return tickOutboundObserver(1, 10*time.Second)
		}, domain.PortalFlowInbound, func(*domain.Observer) {}},
		{"returning through none flow", func() domain.Observer {
			return tickReturningObserver(1, 10*time.Second)
		}, domain.PortalFlowNone, func(*domain.Observer) {}},
		{"returning through outbound flow", func() domain.Observer {
			return tickReturningObserver(1, 10*time.Second)
		}, domain.PortalFlowOutbound, func(*domain.Observer) {}},
		{"returning plane differs from portal destination", func() domain.Observer {
			return tickReturningObserver(1, 10*time.Second)
		}, domain.PortalFlowInbound, func(o *domain.Observer) {
			planeID := int64(2)
			o.CurrentPlaneID = &planeID
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := observerPortal()
			p.ObserverFlow = tt.flow
			state := observerTickState(p, tt.observer())
			tt.mutate(&state.Observers[0])
			before := state
			rnd := &countingRandom{}
			err := resolveStateForValidation(&state, now, rnd, config.Default())
			require.ErrorIs(t, err, domain.ErrSimulationInvariant)
			require.Equal(t, before, state)
			require.Zero(t, rnd.intCalls)
			require.Zero(t, rnd.floatCalls)
		})
	}
}
```

Отдельный positive regression обязан сохранить due phase:

```go
func TestSimulationState_AllowsDueObserverPhaseForTickCatchUp(t *testing.T) {
	state := observerTickState(observerPortal(), tickOutboundObserver(1, 5*time.Second))
	err := resolveStateForValidation(
		&state,
		testutil.BaseTime.Add(6*time.Second),
		nil,
		config.Default(),
	)
	require.NoError(t, err)
	require.Equal(t, domain.ObserverExploring, state.Observers[0].Status)
}
```

- [ ] **Шаг 2: подтвердить RED только для malformed cases**

```bash
go test -count=1 -run 'TestSimulationState_RejectsNonCanonicalObserver|TestSimulationState_AllowsDueObserverPhaseForTickCatchUp' ./internal/domain
```

Ожидание: malformed cases FAIL, positive due-phase case PASS.

- [ ] **Шаг 3: закоммитить RED**

```bash
git add internal/domain/simulation_state_test.go
git commit -m "test(stage8): RED canonical observer aggregate"
```

- [ ] **Шаг 4: реализовать timestamp и flow validation**

Перед status switch:

```go
if observer.CreatedAt.After(now) ||
	observer.UpdatedAt.Before(observer.CreatedAt) ||
	observer.UpdatedAt.After(now) ||
	(hasStart && observer.PhaseStartedAt.After(now)) {
	return ErrSimulationInvariant
}
```

После разрешения active Portal:

```go
switch observer.Status {
case ObserverOutbound:
	if portals[portalIDs[*observer.ActivePortalID]].ObserverFlow != PortalFlowOutbound {
		return ErrSimulationInvariant
	}
case ObserverReturning:
	portal := portals[portalIDs[*observer.ActivePortalID]]
	if portal.ObserverFlow != PortalFlowInbound ||
		observer.CurrentPlaneID == nil ||
		portal.DestinationPlaneID != *observer.CurrentPlaneID {
		return ErrSimulationInvariant
	}
}
```

Не добавлять `now.Before(PhaseEndsAt)`: due transit обязан пройти preflight и
разрешиться текущим tick.

- [ ] **Шаг 5: подтвердить GREEN и закоммитить**

```bash
go test -count=1 -run 'TestSimulationState_|TestSimulationTick_.*Observer|TestSimulationTick_.*Transit' ./internal/domain
go test -count=1 ./internal/domain
git add internal/domain/simulation.go
git commit -m "fix(stage8): GREEN canonical observer aggregate"
```

### Task 6: Синхронизировать Final-Spec mappings и планы

**Файлы:**

- Изменить: `10_STAGE_08_SIMULATION_TDD.md`
- Изменить: `docs/requirements.md`
- Изменить: `docs/traceability.md`
- Изменить: `01_AI_WORKLOG_CURRENT.md`
- Изменить:
  `docs/superpowers/specs/2026-09-04-stage8-corrective-design.md`

- [ ] **Шаг 1: исправить Stage 8 execution-spec**

В §10 заменить правило, по которому ID-order определяет Plane timestamp:

```text
Observers посещаются по возрастанию ID только для детерминированного обхода.
Для Plane, который был UNEXPLORED в начале Observer stage, ExploredAt равен
минимальному semantic return deadline среди всех успешных RETURNING,
разрешённых этим тиком. Уже исследованный Plane сохраняет исходный ExploredAt.
```

Добавить corrective regression names и prepared Extraction rule. В state
invariants перечислить фактически реализованные Plane/Portal/Observer проверки.

- [ ] **Шаг 2: уточнить Extraction requirements**

Сохранить:

```text
EXTRACTION-002 — открытие доступно, только когда хотя бы один WAITING_RETURN
Observer существует где-либо.
```

Добавить:

```text
EXTRACTION-011 — выбранный Plane должен содержать WAITING_RETURN Observer;
Plane без него disabled/невалиден для открытия.
```

Обе строки после GREEN получают конкретные test и implementation mappings.

- [ ] **Шаг 3: добавить известные Recommendation requirements**

Добавить family:

```text
RECOMMENDATION-001 — deterministic, runtime LLM не используется.
RECOMMENDATION-002 — enum: LEAVE OPEN, WAIT FOR CORRIDOR, STABILIZE, CLOSE,
SEND OBSERVER, RECALL OBSERVER.
RECOMMENDATION-003 — показывается только в Portal Details.
RECOMMENDATION-004 — информационная подсказка, не hard restriction.
```

Все четыре строки имеют `PLANNED`. В Notes traceability явно записать:
`selection decision table отсутствует в Final Spec; Stage 12/17 plan
заблокирован до product amendment`.

- [ ] **Шаг 4: обновить traceability**

Добавить новые regression tests к `PLANE-009`, `EXTRACTION-008`,
`SIMULATION-002` и `SIMULATION-003`. Убрать завышенное утверждение о
complete aggregate preflight, пока каждый новый rejection test не GREEN; после
полного GREEN описать точный покрытый набор invariants.

`PLANE-002` перевести минимум в PARTIAL с evidence multiple Portal
destinations и честной границей persistence.

- [ ] **Шаг 5: записать corrective worklog**

Добавить отдельный раздел с:

- исходными тремя дефектами;
- утверждённым дизайном;
- каждым RED и GREEN hash;
- изменёнными semantics;
- Recommendation gate;
- командами итоговой верификации;
- подтверждением, что Stage 9 не начат.

- [ ] **Шаг 6: проверить документацию и закоммитить**

```bash
rg -n "TB[D]|TO[D]O|FIXM[E]" docs/superpowers/specs/2026-09-04-stage8-corrective-design.md docs/superpowers/plans/2026-09-04-stage8-corrective.md
git diff --check
git add 10_STAGE_08_SIMULATION_TDD.md docs/requirements.md docs/traceability.md 01_AI_WORKLOG_CURRENT.md docs/superpowers/specs/2026-09-04-stage8-corrective-design.md
git commit -m "docs(stage8): record corrective evidence and contracts"
```

Ожидание: placeholder scan пуст, `git diff --check` exit 0.

### Task 7: Полная верификация и scope audit

**Файлы:**

- Только чтение; при обнаружении дефекта вернуться к соответствующему RED
  checkpoint.

- [ ] **Шаг 1: formatting**

```bash
gofmt -w internal/domain/simulation.go internal/domain/extraction.go internal/domain/observer_commands.go internal/domain/simulation_observer_test.go internal/domain/simulation_extraction_test.go internal/domain/simulation_state_test.go internal/domain/simulation_portal_lifecycle_test.go
gofmt -l .
```

Ожидание: второй вызов не печатает файлов.

- [ ] **Шаг 2: обязательный suite**

```bash
go vet ./...
go build ./...
go test -count=1 ./...
go test -race -count=1 ./...
```

Ожидание: каждая команда exit 0.

- [ ] **Шаг 3: проверить trace references**

Извлечь все `Test...` из `docs/traceability.md` и подтвердить, что каждое
имя существует среди top-level Go tests. Ожидание: missing count = 0.

- [ ] **Шаг 4: проверить scope**

```bash
git diff d97c3f3..HEAD -- internal/domain/event.go internal/engine/manager.go
git status --short
```

Ожидание: Event/manager diff пуст; worktree чистый.

- [ ] **Шаг 5: финальная точка**

Не создавать Stage 9 plan или implementation. В отчёте привести:

- реализованные corrections;
- изменённые файлы;
- все RED/GREEN hashes;
- результаты gofmt/vet/build/test/race;
- traceability/requirements изменения;
- Recommendation product gate;
- подтверждение, что Stage 9 не начат.
