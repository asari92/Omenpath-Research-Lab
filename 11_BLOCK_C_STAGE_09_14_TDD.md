# Omenpath Research Lab — Block C (Stages 9–14) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) or
> `superpowers:executing-plans` to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** довести backend от чистой Stage 8 simulation до сохраняемого,
конкурентно-безопасного приложения с Events, SQLite, REST, WebSocket и полным
Tutorial lifecycle.

**Architecture:** domain остаётся детерминированным и не знает о SQLite/HTTP;
SQLite атомарно хранит snapshots и Events; `LabManager` единолично сериализует
время, random, команды и persistence под mutex; REST и WebSocket используют
один публичный DTO builder; Tutorial — mode-aware orchestration поверх тех же
domain-команд и событий.

**Tech Stack:** Go 1.26, `database/sql`, pure-Go `modernc.org/sqlite`,
`github.com/go-chi/chi/v5`, `github.com/coder/websocket`, `testify`, SQLite,
HTTP/JSON, WebSocket.

---

## 1. Основание и порядок исполнения

Приоритет документов:

1. `00_FINAL_SPEC_v5.md`, включая согласованные §§26.1 и 28.1.
2. `01_AI_WORKLOG_CURRENT.md`.
3. `02_IMPLEMENTATION_ROADMAP_TDD.md`.
4. Этот execution-plan.
5. `docs/requirements.md` и `docs/traceability.md`.

Block C выполняется строго по стадиям:

```text
Stage 9 Events
  → Stage 10 SQLite
  → Stage 11 LabManager
  → Stage 12 REST
  → Stage 13 WebSocket
  → Stage 14 Tutorial
  → полная проверка и остановка
```

Внутри блока дополнительное подтверждение между стадиями не требуется. Нельзя
начинать Stage 15 или создавать frontend. Recommendation decision table всё
ещё не определена: Stage 12 не придумывает алгоритм и честно оставляет эту
часть Portal Details как `PARTIAL`.

Для каждого checkpoint:

```text
requirements → tests → RED commit → minimal implementation
→ focused GREEN → stage suite → docs/traceability → GREEN commit
```

RED-коммит содержит только новые/изменённые tests и минимальные test helpers.
Он обязан падать по ожидаемой причине отсутствующей функциональности, а не из-за
syntax/import error. GREEN-коммит содержит production implementation и
documentation status текущего checkpoint.

## 2. Сквозные инварианты Block C

- Все lifecycle timestamps нового Block C кода берутся из внедрённого
  `clock.Clock`; orchestration не вызывает `time.Now()`.
- Все random decisions идут через существующий `random.Random`.
- Сначала под тем же manager lock выполняется full `ResolveTick(now)`, затем
  команда. Если lifecycle сделал Portal terminal, команда получает
  `ErrPortalNotOpen`, но выигравшие catch-up transitions сохраняются.
- In-memory semantic state меняется только после успешной SQLite transaction.
  Единственное исключение — ephemeral `LastTickAt` при тике без meaningful
  transition: он может продвинуться в памяти без DB write и безопасно
  восстанавливается после restart повторным deterministic catch-up.
- Rejected domain command не применяет свою мутацию, но сохраняет catch-up
  transitions и один `ACTION_REJECTED`.
- Malformed JSON, синтаксически неверный path, wrong method и неизвестный route
  не являются domain actions и не создают `ACTION_REJECTED`. Valid command URL
  с отсутствующим Portal/Plane доходит до manager, считается domain rejection
  и создаёт `ACTION_REJECTED` с запрошенным entity ID.
- Derived energy, creatures, remaining time и risk score не записываются
  каждую секунду и не хранятся как отдельные current-value columns.
- State и DTO возвращаются глубокими копиями; aliases и pointer fields не
  должны позволять вызывающему менять manager state.
- REST и WebSocket сериализуют один тип snapshot и никогда не раскрывают
  numeric risk score, energy lifetime, decay rate или hidden collapse time.
- Каждый persisted Event имеет валидный enum, непустое message, UTC timestamp и
  `payload_json`, являющийся JSON object.
- Event Log и Portal History читают одну таблицу `events`, сортировка
  `created_at ASC, id ASC`, без пагинации.
- Tutorial отключает Natural generator. Live включает новый Natural schedule.
- `go test -race -count=1 ./...` не допускает data races и duplicate Events.

## 3. Целевая карта файлов

```text
data/embed.go
internal/domain/event.go
internal/domain/event_transition.go
internal/domain/event_test.go
internal/domain/event_transition_test.go
internal/domain/simulation.go
internal/domain/simulation_event_test.go
internal/domain/tutorial.go
internal/domain/tutorial_test.go
internal/persistence/migrations/001_initial.sql
internal/persistence/migrations/002_tutorial_context.sql
internal/persistence/migrations.go
internal/persistence/codec.go
internal/persistence/store.go
internal/persistence/store_migration_test.go
internal/persistence/store_roundtrip_test.go
internal/persistence/store_atomic_test.go
internal/persistence/store_tutorial_test.go
internal/engine/repository.go
internal/engine/manager.go
internal/engine/manager_commands.go
internal/engine/manager_test.go
internal/engine/manager_commands_test.go
internal/engine/manager_concurrency_test.go
internal/engine/manager_tutorial.go
internal/engine/manager_tutorial_test.go
internal/transport/dto.go
internal/transport/dto_test.go
internal/httpapi/router.go
internal/httpapi/read_handlers.go
internal/httpapi/command_handlers.go
internal/httpapi/tutorial_handlers.go
internal/httpapi/router_test.go
internal/httpapi/command_handlers_test.go
internal/httpapi/tutorial_handlers_test.go
internal/realtime/hub.go
internal/realtime/handler.go
internal/realtime/hub_test.go
internal/realtime/handler_test.go
cmd/server/main.go
docs/requirements.md
docs/traceability.md
01_AI_WORKLOG_CURRENT.md
go.mod
go.sum
```

Допускается объединить маленькие файлы внутри того же package, но нельзя
переносить ответственность между стадиями или создавать package import cycle.

## 4. Согласованные контракты

### 4.1 Events

Расширить `internal/domain/event.go`:

```go
type EventDraft struct {
    EventType   EventType
    PortalID    *int64
    ObserverID  *int64
    PlaneID     *int64
    Message     string
    PayloadJSON string
    CreatedAt   time.Time
}

func (d EventDraft) Validate() error
func (e Event) ValidatePersisted() error
func IsEventType(value EventType) bool
func PortalHistory(events []Event, portalID int64) []Event
func EventsForStateTransition(
    before, after SimulationState,
    previousAt, now time.Time,
    cfg config.Config,
) ([]EventDraft, error)
func NewActionRejectedEvent(
    now time.Time,
    action string,
    portalID, observerID, planeID *int64,
    cause error,
) (EventDraft, error)
```

Добавить в `SimulationTickResult`:

```go
Events []EventDraft
```

`ResolveTick` возвращает drafts, но никогда не присваивает Event ID и не пишет
SQLite. Повтор `ResolveTick` с тем же `now` возвращает пустой `Events`.

Порядок events сначала определяется semantic `CreatedAt`, затем стабильным
порядком при одинаковом timestamp:

1. `PORTAL_CLOSED` / `PORTAL_COLLAPSED`;
2. `LEYLINE_OVERRIDE_ENDED`, затем `LEYLINE_OVERRIDE_STARTED`;
3. Observer transit events; для прибытия в Plane строго
   `OBSERVER_ARRIVED`, затем `RESEARCH_STARTED`;
4. `RESEARCH_COMPLETED`;
5. `EXTRACTION_SYNCHRONIZED`, затем `OBSERVER_RETURN_STARTED`;
6. `OBSERVER_RETURNED` / `OBSERVER_LOST`, затем `PLANE_EXPLORED`;
7. `RISK_LEVEL_CHANGED`;
8. новое `PORTAL_OPENED`.

Late tick восстанавливает все пересечённые semantic transitions. Например,
OUTBOUND, который успел прибыть и закончить research между тиками, создаёт
`OBSERVER_ARRIVED`, `RESEARCH_STARTED`, `RESEARCH_COMPLETED` с timestamps
границ фаз, даже если итоговый статус уже `WAITING_RETURN`.

### 4.2 Persistence

`data/embed.go` экспортирует seed через `//go:embed` без зависимости от cwd.

Публичный persistence API:

```go
type Snapshot struct {
    Simulation domain.SimulationState
    App        domain.AppState
}

type Store struct { db *sql.DB }

func Open(ctx context.Context, path string) (*Store, error)
func (s *Store) Close() error
func (s *Store) Migrate(ctx context.Context) error
func (s *Store) Bootstrap(ctx context.Context, now time.Time, cfg config.Config) error
func (s *Store) Load(ctx context.Context) (Snapshot, error)
func (s *Store) Commit(
    ctx context.Context,
    snapshot Snapshot,
    events []domain.EventDraft,
) ([]domain.Event, error)
func (s *Store) ListEvents(ctx context.Context, portalID *int64) ([]domain.Event, error)
```

`Commit` — одна SQL transaction для baseline/state upserts и Event inserts.
При любой ошибке откатываются и state, и events. Назначенные SQLite IDs
возвращаются только после commit.

`001_initial.sql` создаёт:

- `schema_migrations(version PRIMARY KEY, applied_at)`;
- `planes(id, name, aliases_json, catalog_tier, explored, explored_at)`;
- `portals` со всеми полями Final Spec §34 плюс
  `extraction_synchronized_at`;
- `observers` по Final Spec §34;
- `events` по Final Spec §26;
- singleton `lab_state` с `energy_base`, `energy_base_at`, `override_until`;
- singleton `app_state` с `mode`, `tutorial_step`, `next_portal_id`,
  `spawn_scheduled_at`, `spawn_due_at`, `spawn_paused`, `last_tick_at`.

Ограничения schema:

- partial unique index на `portals(slot_index) WHERE status = 'OPEN'`;
- unique Plane/Observer IDs;
- indexes `events(created_at, id)` и
  `events(portal_id, created_at, id)`;
- booleans через `CHECK (value IN (0,1))`;
- enum/status checks там, где migration остаётся читаемой;
- timestamps — UTC Unix nanoseconds, nullable timestamps — SQL NULL.

Bootstrap идемпотентен: первая пустая база получает ровно 85 Planes из embedded
seed, 10 AVAILABLE Observers, empty Portals/Events, Tutorial Step 0, Energy 100,
`NextPortalID = 1`. Повторный bootstrap не перезаписывает состояние.

### 4.3 Manager и repository boundary

`internal/engine/repository.go` задаёт узкий interface, реализуемый `Store` и
fake repository tests:

```go
type Repository interface {
    Load(context.Context) (persistence.Snapshot, error)
    Commit(context.Context, persistence.Snapshot, []domain.EventDraft) ([]domain.Event, error)
    ListEvents(context.Context, *int64) ([]domain.Event, error)
}
```

Добавить стабильные ошибки `ErrPortalNotFound`, `ErrPlaneNotFound`,
`ErrTutorialNotReady` и `ErrInvalidTutorialSignal` без HTTP semantics внутри
domain/engine. HTTP mapping остаётся только в `internal/httpapi`.

`LabManager` хранит mutex, snapshot, config, clock, random, repository и
coalescing update channel capacity 1. Constructor загружает существующий
snapshot; bootstrap остаётся обязанностью composition root.

```go
func NewLabManager(
    ctx context.Context,
    cfg config.Config,
    clk clock.Clock,
    rnd random.Random,
    repo Repository,
) (*LabManager, error)

func (m *LabManager) Tick(ctx context.Context) error
func (m *LabManager) Run(ctx context.Context, ticks <-chan time.Time) error
func (m *LabManager) State(ctx context.Context) (persistence.Snapshot, error)
func (m *LabManager) Portal(ctx context.Context, id int64) (domain.Portal, []domain.Event, error)
func (m *LabManager) Events(ctx context.Context) ([]domain.Event, error)
func (m *LabManager) Stabilize(ctx context.Context, id int64) error
func (m *LabManager) ClosePortal(ctx context.Context, id int64, confirm bool) error
func (m *LabManager) SendObserver(ctx context.Context, id int64, confirm bool) error
func (m *LabManager) RecallObserver(ctx context.Context, id int64, confirm bool) error
func (m *LabManager) OpenExtraction(ctx context.Context, planeID int64) error
func (m *LabManager) Updates() <-chan struct{}
```

Каждый state/portal read использует exclusive lock, потому что сначала делает
time catch-up. `Events` может читать repository без изменения state. Все
команды используют один unexported transaction template: clone → resolve →
command → event diff/rejection → repository commit → memory publish → update
signal.

Manager добавляет `tick.Events` как catch-up events и вызывает
`EventsForStateTransition` только для разницы resolved-state → post-command,
поэтому одно событие не создаётся дважды. Tick без event, scheduler/app/baseline
или entity transition не пишет SQLite: обновляются только in-memory
`LastTickAt`, derived snapshot и WebSocket signal. Изменение Natural schedule,
даже без открытия Portal, считается meaningful и persist.

### 4.4 REST DTO

`internal/transport` — единственный публичный JSON contract для REST и WS.

State snapshot содержит:

- `generated_at`, `app`, `lab`, exploration progress;
- observer counts по status;
- portal counts и `needs_attention_portal_id`;
- всегда ровно семь `slots`, свободный slot имеет `portal: null`;
- Plane catalog.

Slot portal содержит только разрешённые Final Spec §5 поля и quick-action
availability. Portal Details дополнительно содержит destination, `risk_level`
только для OPEN portal и history. Numeric risk и Recommendation до product
decision table отсутствуют; `API-002` поэтому остаётся `PARTIAL` только по
Recommendation.

### 4.5 REST errors

Единый envelope:

```json
{
  "error": {
    "code": "CONFIRMATION_REQUIRED",
    "message": "confirmation required",
    "confirmable": true
  }
}
```

Mapping:

- `400`: malformed JSON, unknown field, extra JSON value, invalid path/body;
- `404`: отсутствующий Portal или Plane;
- `409`: domain conflict, включая confirmation-required;
- `500`: invariant, persistence и неожиданная внутренняя ошибка;
- `405`: известный route с неверным method.

Успешный command возвращает `200` и свежий authoritative state snapshot.

### 4.6 WebSocket

`GET /ws/lab` только server → client snapshots. После upgrade клиент сразу
получает current snapshot, затем snapshot на каждый simulation tick и сразу
после каждого важного action, включая persisted domain rejection.

Hub имеет отдельную writer goroutine и queue capacity 1 на клиента. Новая
версия snapshot заменяет непрочитанную старую; медленный клиент не блокирует
manager, ticker и остальных клиентов. Reconnect всегда получает latest state.

### 4.7 Tutorial state

Расширить `AppState`:

```go
type TutorialSignal string

const (
    TutorialSignalPortalDetailsOpened TutorialSignal = "PORTAL_DETAILS_OPENED"
    TutorialSignalEventLogOpened      TutorialSignal = "EVENT_LOG_OPENED"
)

type AppState struct {
    Mode               AppMode
    TutorialStep       int
    TutorialPortalID   *int64
    TutorialPlaneID    *int64
    TutorialObserverID *int64
}
```

`002_tutorial_context.sql` добавляет target IDs. Restart обязан продолжить тот
же step и targets. Tutorial tick не вызывает Natural spawn; он управляет
prepared Portal через те же lifecycle/event primitives.

Step contract:

| Step | Подготовка | Единственное completion condition |
|---|---|---|
| 0 | Empty State | первый tick создаёт Tutorial Portal → 1 |
| 1 | target Portal существует | matching `PORTAL_DETAILS_OPENED` → 2 |
| 2 | тот же Portal содержит creatures | creatures стали 0 → 3 |
| 3 | доступен SEND | успешный SEND target Observer → 4 |
| 4 | prepared UNSTABLE Portal с Risk HIGH | успешный STABILIZE даёт MEDIUM → 5 |
| 5 | prepared CRITICAL Portal | отклонённый SEND создаёт event и → 6 |
| 6 | Observer завершает research; новый Portal к тому же Plane | успешный RECALL → 7 |
| 7 | Observer вернулся | Observer AVAILABLE и Plane EXPLORED → 8 |
| 8 | Event Log доступен | `EVENT_LOG_OPENED` → 9 |
| 9 | Training Complete | успешный `live/start` → Live |

Wrong reversible action оставляет step без изменения. Terminal target Portal
пересоздаётся эквивалентным с новым ID. LOST в Step 7 возвращает к Step 6:
выбирается другой AVAILABLE Observer и создаётся эквивалентный prepared state
с этим Observer в WAITING_RETURN в том же пока UNEXPLORED Plane; затем создаётся
новый inbound Portal и повторяется RECALL. Такое восстановление является частью
детерминированного Tutorial scenario, не Live gameplay.

`tutorial/reset` одной transaction заменяет snapshot начальным Tutorial state
и удаляет старые events; для этого Stage 14 расширяет repository методом
`ResetTutorial(context.Context, persistence.Snapshot) error`. `live/start`
закрывает оставшиеся Tutorial Portals как `MANUAL_CLOSE` без energy debit,
создаёт обычные close events с payload source `TUTORIAL_COMPLETION`, сохраняет
Energy/Events/Observers/Planes, включает Mode Live и создаёт свежий Natural
schedule. После перехода OPEN Portals = 0.

## 5. Stage 9 — Event system

### Checkpoint 9A — model, validation и shared history

- [ ] Добавить `EventDraft`, enum validation, persisted validation и
  `PortalHistory` в `internal/domain/event.go`.
- [ ] Создать `internal/domain/event_test.go` с тестами:
  `TestEventDraftValidate_AcceptsCanonicalEvent`,
  `TestEventDraftValidate_RejectsUnknownType`,
  `TestEventDraftValidate_RequiresJSONObjectPayload`,
  `TestEventValidatePersisted_RequiresPositiveID`,
  `TestPortalHistory_FiltersAndKeepsChronology`.
- [ ] Запустить `go test -count=1 ./internal/domain -run 'Test(Event|PortalHistory)'`
  и зафиксировать ожидаемый RED.
- [ ] Commit RED:
  `test(stage9): RED event validation and shared portal history`.
- [ ] Реализовать минимум до focused GREEN.
- [ ] Commit GREEN:
  `feat(stage9): GREEN event validation and shared portal history`.

### Checkpoint 9B — deterministic transition stream

- [ ] Создать `event_transition.go`, `event_transition_test.go` и
  `simulation_event_test.go`.
- [ ] Покрыть:
  `TestEventsForTransition_NaturalOpenAndDedicatedExtractionOpen`,
  `TestEventsForTransition_CloseAndCollapse`,
  `TestEventsForTransition_OverrideRestartAndEndExactlyOnce`,
  `TestEventsForTransition_OutboundArrivalStartsResearch`,
  `TestEventsForTransition_LateTickReconstructsObserverPhases`,
  `TestEventsForTransition_ReturnAndLoss`,
  `TestEventsForTransition_ExtractionSyncBeforeAutomaticReturn`,
  `TestEventsForTransition_RiskBandChangeOnly`,
  `TestEventsForTransition_MultiBandJumpIsSingleEvent`,
  `TestEventsForTransition_OpenAndTerminalDoNotEmitRiskChange`,
  `TestSimulationTick_ReplayHasNoEvents`,
  `TestSimulationTick_EventOrderUsesSemanticTime`,
  `TestNewActionRejectedEvent_EncodesDomainCause`.
- [ ] Заменить прежний Stage 8 guard об отсутствии Events на доказательство,
  что tick возвращает drafts, но ничего не persist.
- [ ] Запустить focused suite и зафиксировать RED.
- [ ] Commit RED:
  `test(stage9): RED deterministic simulation event stream`.
- [ ] Реализовать event diff/reconstruction и заполнение
  `SimulationTickResult.Events` без SQLite.
- [ ] Выполнить `gofmt`, `go test -count=1 ./internal/domain`, затем
  `go test -count=1 ./...`.
- [ ] Обновить EVENT rows: `EVENT-002`, `004..009`, `011` → GREEN;
  `EVENT-001`, `003`, `010` → PARTIAL до command orchestration Stage 11.
- [ ] Дополнить worklog фактическими RED/GREEN hashes.
- [ ] Commit GREEN:
  `feat(stage9): GREEN deterministic domain event stream`.

Stage 9 не импортирует `database/sql`, chi, websocket или engine.

## 6. Stage 10 — SQLite persistence

### Checkpoint 10A — migrations и bootstrap

- [ ] Подключить `modernc.org/sqlite` и embedded seed.
- [ ] Добавить migration/bootstrap tests:
  `TestStoreMigrate_CreatesRequiredTablesAndIndexes`,
  `TestStoreMigrate_IsIdempotent`,
  `TestStoreBootstrap_SeedsExactly85PlanesAnd10Observers`,
  `TestStoreBootstrap_UsesTutorialEnergy100AndStep0`,
  `TestStoreBootstrap_DoesNotOverwriteExistingState`,
  `TestStoreBootstrap_DoesNotDependOnWorkingDirectory`.
- [ ] RED commit:
  `test(stage10): RED sqlite migrations and canonical bootstrap`.
- [ ] Реализовать `data/embed.go`, migration runner, `Open`, `Close`,
  `Bootstrap` и schema constraints.
- [ ] Проверить только на временных DB через `t.TempDir()`; in-memory tests
  должны использовать уникальный DSN на test.
- [ ] Commit GREEN:
  `feat(stage10): GREEN sqlite migrations and canonical bootstrap`.

### Checkpoint 10B — atomic round-trip и restart

- [ ] Добавить tests:
  `TestStoreCommitAndLoad_RoundTripsCompleteSnapshot`,
  `TestStoreCommit_PreservesExtractionAndOverrideTimestamps`,
  `TestStoreCommit_PersistsSchedulerAndNextPortalID`,
  `TestStoreCommit_AssignsEventIDsInChronologicalOrder`,
  `TestStoreListEvents_GlobalAndPortalUseSameRows`,
  `TestStoreCommit_RollsBackStateWhenEventInsertFails`,
  `TestStoreLoad_RecoversOverdueStateWithoutResolvingIt`,
  `TestSchema_HasNoDerivedRealtimeColumns`.
- [ ] RED commit:
  `test(stage10): RED atomic persistence and restart recovery`.
- [ ] Реализовать codecs, `Load`, `Commit`, `ListEvents`; invalid decoded enum,
  timestamp или JSON должен вернуть ошибку, а не silently normalize.
- [ ] Закрыть все prepared statements/rows; настроить SQLite foreign keys,
  busy timeout и один writer connection.
- [ ] Выполнить focused persistence suite и полный обычный suite.
- [ ] `PERSIST-001`, `PERSIST-002` → GREEN; `PERSIST-003`, `PERSIST-004` →
  PARTIAL до manager startup/meaningful-write proof.
- [ ] Worklog + GREEN commit:
  `feat(stage10): GREEN atomic sqlite persistence and recovery`.

Stage 10 не запускает ticker, HTTP server или WebSocket.

## 7. Stage 11 — LabManager / concurrency

### Checkpoint 11A — ownership, resolve-first, atomic commit

- [ ] Заменить skeleton `manager.go`; добавить `repository.go` и
  `manager_commands.go`.
- [ ] Создать controllable fake repository и tests:
  `TestNewLabManager_LoadsPersistedSnapshot`,
  `TestManagerTick_CommitsStateAndEventsAtomically`,
  `TestManagerTick_PersistenceFailureKeepsMemoryUnchanged`,
  `TestManagerCommand_ResolvesWholeSimulationBeforeAction`,
  `TestManagerCommand_DueTerminalPortalWins`,
  `TestManagerRejectedCommand_CommitsCatchupAndActionRejected`,
  `TestManagerMissingEntity_CreatesActionRejectedWithRequestedID`,
  `TestManagerMalformedTransportIsOutsideDomainBoundary`,
  `TestManagerState_ReturnsDeepCopy`,
  `TestManagerPortal_ReturnsFilteredSharedHistory`.
- [ ] RED commit:
  `test(stage11): RED manager ownership and resolve-first transactions`.
- [ ] Реализовать clone/apply/commit/publish template и все команды.
- [ ] Не откатывать catch-up state при domain rejection; возвращать исходную
  domain error после успешного persistence.
- [ ] Commit GREEN:
  `feat(stage11): GREEN manager ownership and resolve-first transactions`.

### Checkpoint 11B — ticker, signals и races

- [ ] Добавить tests:
  `TestManagerRun_ConsumesInjectedTicksUntilContextCancel`,
  `TestManagerTick_SignalsEvenWithoutMeaningfulDatabaseWrite`,
  `TestManagerAction_SignalsAfterSuccessAndRejection`,
  `TestManagerUpdates_CoalesceWithoutBlocking`,
  `TestManagerConcurrentCommands_OnlyOneTransitionWins`,
  `TestManagerConcurrentTicks_DoNotDuplicateEvents`,
  `TestManagerConcurrentOpenings_KeepUniqueIDsAndSlots`,
  `TestManagerConcurrentReadsAndWrites_ReturnConsistentSnapshots`.
- [ ] RED commit:
  `test(stage11): RED manager ticker updates and concurrency`.
- [ ] Реализовать `Run`, coalescing signal и synchronization.
- [ ] Запустить:
  `go test -count=1 ./internal/engine`,
  `go test -race -count=1 ./internal/engine`,
  `go test -count=1 ./...`.
- [ ] `EVENT-001`, `EVENT-003`, `EVENT-010`, `PERSIST-003`, `PERSIST-004`,
  `SIMULATION-001`, `PORTAL-001`, `PORTAL-002`, `WS-002` backend boundary
  обновить до фактически доказанного GREEN/PARTIAL.
- [ ] Worklog + GREEN commit:
  `feat(stage11): GREEN serialized lab manager and simulation loop`.

Stage 11 не содержит HTTP status codes или JSON DTO.

## 8. Stage 12 — REST API

### Checkpoint 12A — authoritative read DTOs

- [ ] Подключить chi v5.
- [ ] Создать `internal/transport/dto.go`, DTO tests и read handlers.
- [ ] Tests:
  `TestBuildStateSnapshot_AlwaysReturnsSevenFixedSlots`,
  `TestBuildStateSnapshot_DerivesCurrentValuesAtGeneratedAt`,
  `TestBuildStateSnapshot_DoesNotExposeHiddenFields`,
  `TestBuildPortalDetails_OpenIncludesRiskAndHistory`,
  `TestBuildPortalDetails_TerminalHasNullRisk`,
  `TestGetState_ReturnsAuthoritativeSnapshot`,
  `TestGetPortal_ReturnsDetailsOr404`,
  `TestGetEvents_ReturnsChronologicalGlobalLog`,
  `TestReadEndpoints_DoNotAdvanceTutorial`.
- [ ] RED commit: `test(stage12): RED REST reads and public DTO contract`.
- [ ] Реализовать DTO builder/router/read handlers.
- [ ] GREEN commit: `feat(stage12): GREEN REST reads and public DTO contract`.

### Checkpoint 12B — commands, strict JSON и errors

- [ ] Реализуемые routes Stage 12:
  `POST /api/portals/{id}/stabilize`, `/close`, `/send-observer`,
  `/recall-observer`, `POST /api/extraction/open`.
- [ ] Bodies: `{}` или `{"confirm": boolean}` для portal commands;
  `{"plane_id": integer}` для extraction. Unknown fields и второй JSON value
  отклоняются как 400.
- [ ] Tests:
  `TestPortalCommandRoutes_ReturnFreshState`,
  `TestCloseConfirmationFlow_Returns409ThenSucceeds`,
  `TestDomainConflictMapping`,
  `TestCommandRoute_UnknownPortalIs404`,
  `TestExtractionRoute_UnknownPlaneIs404`,
  `TestCommandRoute_StrictJSONRejectsUnknownAndTrailingValues`,
  `TestMalformedRequest_DoesNotCreateActionRejected`,
  `TestWrongMethod_Returns405`,
  `TestInternalFailure_ReturnsOpaque500`.
- [ ] RED commit: `test(stage12): RED REST commands and domain errors`.
- [ ] Реализовать handlers/error mapper без дублирования domain rules.
- [ ] Проверить `go test -count=1 ./internal/transport ./internal/httpapi`
  и весь suite.
- [ ] API-001, API-003..008, API-010, API-011 → GREEN; API-002 → PARTIAL
  только из-за отложенной Recommendation; API-009/API-012 остаются PLANNED до
  Stage 14.
- [ ] Worklog + GREEN commit:
  `feat(stage12): GREEN REST reads commands and error mapping`.

Stage 12 не реализует Tutorial commands и WebSocket.

## 9. Stage 13 — WebSocket realtime

### Checkpoint 13A — initial snapshot и tick broadcast

- [ ] Подключить `github.com/coder/websocket` и создать realtime package.
- [ ] Tests:
  `TestWebSocket_ImmediatelyReceivesCurrentSnapshot`,
  `TestWebSocket_TickBroadcastUsesRESTSnapshotSchema`,
  `TestWebSocket_DoesNotExposeHiddenFields`,
  `TestHub_RemovesDisconnectedClient`.
- [ ] RED commit:
  `test(stage13): RED websocket initial and tick snapshots`.
- [ ] Реализовать Hub/handler с `wsjson` и common DTO builder.
- [ ] GREEN commit:
  `feat(stage13): GREEN websocket initial and tick snapshots`.

### Checkpoint 13B — actions, reconnect, slow clients

- [ ] Tests:
  `TestWebSocket_SuccessfulActionBroadcastsImmediately`,
  `TestWebSocket_RejectedDomainActionBroadcastsPersistedEvent`,
  `TestWebSocket_ReconnectGetsLatestSnapshot`,
  `TestHub_SlowClientDoesNotBlockFastClientOrManager`,
  `TestHub_CoalescesPendingSnapshots`,
  `TestWebSocket_ConcurrentConnectBroadcastDisconnect`.
- [ ] RED commit:
  `test(stage13): RED websocket action reconnect and backpressure`.
- [ ] Реализовать per-client queue/write loop и bridge из `Manager.Updates()`.
- [ ] Зарегистрировать `/ws/lab` в router composition без создания второго DTO.
- [ ] Запустить focused suite, полный suite и race для realtime+engine.
- [ ] WS-001..003 → GREEN.
- [ ] Worklog + GREEN commit:
  `feat(stage13): GREEN websocket authoritative realtime snapshots`.

## 10. Stage 14 — Tutorial engine

### Checkpoint 14A — deterministic state machine и persistence context

- [ ] Добавить `tutorial.go`, migration 002 и repository round-trip target IDs.
- [ ] Tests:
  `TestTutorial_FirstTickCreatesStep1Portal`,
  `TestTutorial_OnlyExpectedTargetAndConditionAdvance`,
  `TestTutorial_WrongReversibleActionKeepsStep`,
  `TestTutorial_DetailsAndEventLogRequireExplicitSignal`,
  `TestTutorial_Step2AdvancesWhenCorridorClears`,
  `TestTutorial_StabilizeGuaranteesHighToMedium`,
  `TestTutorial_CriticalRejectedSendCompletesStep5`,
  `TestTutorial_TerminalPreparedPortalRecreatesFreshID`,
  `TestTutorial_TimedPortalAutoRecreatesAfterExpiry`,
  `TestTutorial_TickNeverSpawnsNaturalPortal`,
  `TestTutorialContext_RoundTripsAcrossRestart`.
- [ ] RED commit:
  `test(stage14): RED deterministic tutorial state and restart context`.
- [ ] Реализовать mode-aware tick, preparations и transition guards.
- [ ] Critical rejected SEND должен одной transaction сохранить
  `ACTION_REJECTED` и Step 6, вернуть 409 transport-слою и отправить update.
- [ ] GREEN commit:
  `feat(stage14): GREEN deterministic tutorial state and restart context`.

### Checkpoint 14B — return retry, reset, Live continuity и API

- [ ] Добавить manager Tutorial commands:
  `StartTutorial`, `ResetTutorial`, `TutorialSignal`, `StartLive`.
- [ ] Зарегистрировать три существующих Tutorial/Live routes и новый signal.
- [ ] Tests:
  `TestTutorial_ReturnExploresPlaneOnlyAfterObserverReturned`,
  `TestTutorial_LostReturnRepeatsStep6WithDifferentObserver`,
  `TestTutorialStart_IsIdempotent`,
  `TestTutorialReset_RestoresEnergyObserversPlanesAndClearsHistory`,
  `TestStartLive_BeforeStep9IsRejected`,
  `TestStartLive_PreservesEnergyEventsObserversAndExploration`,
  `TestStartLive_ClosesTutorialPortalsForFreeAndStartsZeroOpen`,
  `TestStartLive_SchedulesNaturalGenerator`,
  `TestTutorialSignal_RejectsUnknownEnum`,
  `TestTutorialAPI_GETsNeverMutateProgress`,
  `TestTutorialAPI_CriticalRejectionPersistsProgressAndEvent`,
  `TestTutorialWebSocket_ReconnectShowsPersistedStep`,
  `TestTutorial_FullColdStartRestartAndLiveJourney`.
- [ ] RED commit:
  `test(stage14): RED tutorial retry reset live continuity and API`.
- [ ] Реализовать atomic reset operation, Live transition и HTTP handlers.
- [ ] Проверить, что старые Tutorial Events удаляются только reset-командой;
  обычный start и Live transition сохраняют историю.
- [ ] Выполнить focused domain/engine/persistence/httpapi/realtime suites и
  затем полный обычный suite.
- [ ] TUTORIAL-001..015, API-009, API-012, LAB-002 → GREEN.
- [ ] Worklog + GREEN commit:
  `feat(stage14): GREEN tutorial lifecycle and live continuity`.

Stage 14 заканчивает Block C. Не добавлять React/Vite, UI pages или Stage 15.

## 11. Composition root и smoke test

К концу Stage 14 `cmd/server/main.go` обязан:

- [ ] прочитать `OMENPATH_DB_PATH` и `OMENPATH_ADDR` из env с безопасными
  defaults `./omenpath.db` и `:8080`;
- [ ] открыть Store, выполнить migration/bootstrap, загрузить Manager;
- [ ] запустить 1-second ticker через `Manager.Run`;
- [ ] подключить REST router и WebSocket Hub;
- [ ] корректно остановить HTTP server, Hub и DB по context/signal;
- [ ] не содержать gameplay logic.

Добавить process-level smoke test там, где он остаётся быстрым и
детерминированным: временная SQLite DB → bootstrap → manager → `httptest.Server`
→ `/api/state` → `/ws/lab` initial snapshot.

## 12. Stage-level verification и Git discipline

После каждой стадии:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
git diff --check
git status --short
```

Если `gofmt -l .` выводит paths, выполнить `gofmt -w` только для затронутых Go
files и повторить проверку. Нельзя коммитить generated DB files, binaries или
временные sockets.

Каждая стадия должна иметь минимум один RED и один GREEN commit. Если стадия
разделена на два checkpoints, сохранить оба RED commits; объединять их задним
числом нельзя.

## 13. Block C final verification

Перед отчётом выполнить из чистого worktree:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
go test -race -count=1 ./...
git diff --check
git status --short
```

Дополнительно:

- [ ] Запустить автоматическую проверку всех test names из
  `docs/traceability.md` на существование.
- [ ] Проверить cold start → Tutorial progress → process restart → Step 9 →
  Live; после Live 0 OPEN Portals и Natural generator scheduled.
- [ ] Проверить одну SQLite transaction на action/state/events через injected
  failure.
- [ ] Проверить concurrent tick + REST command + WS broadcast под race detector.
- [ ] Проверить отсутствие hidden fields в REST и WS JSON.
- [ ] Проверить global/portal event ordering и отсутствие duplicates.
- [ ] Проверить `rg`-ом отсутствие React/Vite/frontend source и Stage 15
  implementation в Block C diff.
- [ ] Обновить `docs/traceability.md`: только доказанные границы GREEN;
  Recommendation остаётся PLANNED/PARTIAL согласно product gate.
- [ ] Дополнить `01_AI_WORKLOG_CURRENT.md`: стадии, решения, ошибки,
  verification, RED/GREEN hashes и вклад пользователя.
- [ ] Сделать финальный docs/verification commit без изменения semantics.

## 14. Definition of Done Block C

- [ ] Stage 9: полный deterministic event stream, shared history и rejection
  drafts доказаны tests.
- [ ] Stage 10: schema, 85-plane seed, atomic persistence и restart recovery
  доказаны tests.
- [ ] Stage 11: resolve-first manager, exactly-one transitions, ticker и race
  safety доказаны tests.
- [ ] Stage 12: reads/commands/errors реализованы; Recommendation не придумана.
- [ ] Stage 13: authoritative initial/tick/action snapshots, reconnect и
  backpressure доказаны tests.
- [ ] Stage 14: весь Tutorial 0–9, retry/recreate/reset/restart/Live continuity
  доказаны unit и integration tests.
- [ ] Обязательные gofmt/vet/build/test/race проходят с exit 0.
- [ ] Requirements, traceability и worklog соответствуют коду и tests.
- [ ] Git history содержит различимые RED/GREEN commits Stages 9–14.
- [ ] Worktree чист.
- [ ] Stage 15 не начат; работа останавливается для пользовательской сверки.
