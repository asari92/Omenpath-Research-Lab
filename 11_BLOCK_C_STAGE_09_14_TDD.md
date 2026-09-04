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
начинать Stage 15 или создавать frontend. Recommendation decision table уже
зафиксирована в Final Spec §23.1: Stage 12 реализует чистый backend algorithm и
Portal Details DTO, а Stage 17 позже только отображает готовое значение.

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
internal/domain/recommendation.go
internal/domain/recommendation_test.go
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
только для OPEN portal, `recommendation` и history. Для terminal Portal поля
`risk_level` и `recommendation` равны `null`. Numeric risk отсутствует.

### 4.5 Recommendation Engine

Создать `internal/domain/recommendation.go` по Final Spec §23.1 и approved
design `docs/superpowers/specs/2026-09-04-recommendation-engine-design.md`:

```go
type Recommendation string

const (
    RecommendationLeaveOpen       Recommendation = "LEAVE OPEN"
    RecommendationWaitForCorridor Recommendation = "WAIT FOR CORRIDOR"
    RecommendationStabilize       Recommendation = "STABILIZE"
    RecommendationClose           Recommendation = "CLOSE"
    RecommendationSendObserver    Recommendation = "SEND OBSERVER"
    RecommendationRecallObserver  Recommendation = "RECALL OBSERVER"
)

func RecommendationForPortal(
    state SimulationState,
    portalID int64,
    now time.Time,
    cfg config.Config,
) (Recommendation, bool, error)
```

`bool=false` допустим только для terminal Portal. Функция pure: не вызывает
random, не мутирует aggregate, не читает `InstabilityCollapseAt` и не влияет на
command eligibility.

Safety horizon сравнивается строго (`effective_lifetime > required`), потому
что при exact deadline tie Portal lifecycle выполняется раньше Observer
lifecycle. Required horizon:

```text
SEND/RECALL now       ObserverTransitMax
WAIT corridor         clearance remaining + ObserverTransitMax
active transit        PhaseEndsAt - now
future return         research remaining + ObserverTransitMax
```

Hypothetical Stabilize выполняется на копии Portal и предлагается только если
обычная Lab Energy cost доступна (или Override делает её нулевой) и результат
реально удовлетворяет horizon. Правила применяются сверху вниз: Extraction
pre-sync; active transit; WAITING_RETURN; EXPLORING; useful SEND только в
UNEXPLORED Plane без другого/направляющегося Observer; EXPLORED close; затем
HIGH/CRITICAL close или LEAVE OPEN fallback.

### 4.6 REST errors

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

### 4.7 WebSocket

`GET /ws/lab` только server → client snapshots. После upgrade клиент сразу
получает current snapshot, затем snapshot на каждый simulation tick и сразу
после каждого важного action, включая persisted domain rejection.

Hub имеет отдельную writer goroutine и queue capacity 1 на клиента. Новая
версия snapshot заменяет непрочитанную старую; медленный клиент не блокирует
manager, ticker и остальных клиентов. Reconnect всегда получает latest state.

### 4.8 Tutorial state

Расширить `AppState`:

```go
type TutorialSignal string

const (
    TutorialSignalIntroCompleted      TutorialSignal = "TUTORIAL_INTRO_COMPLETED"
    TutorialSignalPortalDetailsOpened TutorialSignal = "PORTAL_DETAILS_OPENED"
    TutorialSignalEventLogOpened      TutorialSignal = "EVENT_LOG_OPENED"
)

type TutorialPhase string

const (
    TutorialPhaseNone            TutorialPhase = ""
    TutorialPhaseSendReplacement TutorialPhase = "SEND_REPLACEMENT"
    TutorialPhaseWaitResearch    TutorialPhase = "WAIT_RESEARCH"
    TutorialPhaseRecallReady     TutorialPhase = "RECALL_READY"
)

type TutorialExpectedAction string

const (
    TutorialActionCompleteIntro TutorialExpectedAction = "COMPLETE_INTRO"
    TutorialActionOpenDetails   TutorialExpectedAction = "OPEN_PORTAL_DETAILS"
    TutorialActionWaitCorridor  TutorialExpectedAction = "WAIT_CORRIDOR"
    TutorialActionSend          TutorialExpectedAction = "SEND_OBSERVER"
    TutorialActionStabilize     TutorialExpectedAction = "STABILIZE"
    TutorialActionCriticalSend  TutorialExpectedAction = "ATTEMPT_CRITICAL_SEND"
    TutorialActionWaitResearch  TutorialExpectedAction = "WAIT_RESEARCH"
    TutorialActionRecall        TutorialExpectedAction = "RECALL_OBSERVER"
    TutorialActionWaitReturn    TutorialExpectedAction = "WAIT_RETURN"
    TutorialActionOpenEventLog  TutorialExpectedAction = "OPEN_EVENT_LOG"
    TutorialActionStartLive     TutorialExpectedAction = "START_LIVE"
)

type AppState struct {
    Mode               AppMode
    TutorialStep       int
    TutorialPhase      TutorialPhase
    TutorialPortalID   *int64
    TutorialPlaneID    *int64
    TutorialObserverID *int64
}

func ExpectedTutorialAction(app AppState) (TutorialExpectedAction, error)
```

`002_tutorial_context.sql` добавляет phase и target IDs. `expected_action`
выводится из step/phase и отдельно не хранится. Restart обязан продолжить тот
же step/phase/targets без duplicate Portal/Event.

Step contract соответствует Final Spec §28.2 и approved design
`docs/superpowers/specs/2026-09-04-tutorial-guidance-system-design.md`:

| Step | Player action | System transition | Completion |
|---|---|---|---|
| 0 | «Начать практику» | до signal 0 OPEN/paused generator; intro signal создаёт safe Step 1 target | `TUTORIAL_INTRO_COMPLETED` |
| 1 | открыть target Details | GET read-only; signal проверяет `portal_id` | matching `PORTAL_DETAILS_OPENED` |
| 2 | ждать | ticks очищают creatures; broken target получает fresh equivalent ID | creatures = 0 |
| 3 | SEND target | normal command сохраняет Observer/Plane и создаёт Step 4 target | Observer OUTBOUND |
| 4 | STABILIZE target | normal debit/events; создать Step 5 target | STABLE и HIGH/CRITICAL→MEDIUM/LOW |
| 5 | SEND в CRITICAL target | persist rejection + Step 6 в одной transaction | `ErrPortalCriticalRisk` |
| 6 | ждать research, затем RECALL; retry сначала SEND replacement | normal outbound/research/inbound lifecycle и safe return target | Observer RETURNING |
| 7 | ждать | return → AVAILABLE+EXPLORED; LOST → Step 6/SEND_REPLACEMENT | successful return |
| 8 | открыть Event Log | GET read-only; explicit signal | `EVENT_LOG_OPENED` |
| 9 | начать Live | free cleanup, continuity, fresh Natural schedule | Mode LIVE |

Step 0 не меняется от ticks и содержит только лор/цель/карту интерфейса. Цены и
rules выдаются контекстно на Steps 1–9; Stage 14 snapshot возвращает step,
phase, targets и expected action, а точный UI copy остаётся Stage 20.

Prepared properties проверяются behavior tests:

- Step 1: STABLE, creatures > 0, safe clearance + SEND horizon;
- Step 4: UNSTABLE, Energy ≤85%, Risk HIGH/CRITICAL, после Stabilize LOW/MEDIUM;
- Step 5: STABLE CRITICAL из-за scheduled lifetime, Energy depletion позже
  NATURAL_CLOSE;
- Step 6: STABLE, clear, lifetime строго больше ObserverTransitMax.

Tutorial tick выполняет normal lifecycle существующих entities, но Natural
Generator остаётся paused. Terminal target до ожидаемого action сохраняется в
history и заменяется fresh equivalent Portal. Старый Portal не resurrect.

LOST retry не создаёт WAITING_RETURN напрямую: Step 6/SEND_REPLACEMENT открывает
safe outbound Portal, игрок отправляет другого AVAILABLE Observer, ждёт normal
research и только затем получает новый inbound-capable Portal. Если tracked
Observer был потерян ещё на Steps 4–5, вход в Step 6 выбирает тот же retry flow.

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

- [x] Подключить `modernc.org/sqlite` и embedded seed.
- [x] Добавить migration/bootstrap tests:
  `TestStoreMigrate_CreatesRequiredTablesAndIndexes`,
  `TestStoreMigrate_IsIdempotent`,
  `TestStoreBootstrap_SeedsExactly85PlanesAnd10Observers`,
  `TestStoreBootstrap_UsesTutorialEnergy100AndStep0`,
  `TestStoreBootstrap_DoesNotOverwriteExistingState`,
  `TestStoreBootstrap_DoesNotDependOnWorkingDirectory`.
- [x] RED commit:
  `test(stage10): RED sqlite migrations and canonical bootstrap`.
- [x] Реализовать `data/embed.go`, migration runner, `Open`, `Close`,
  `Bootstrap` и schema constraints.
- [x] Проверить только на временных DB через `t.TempDir()`; in-memory tests
  должны использовать уникальный DSN на test.
- [x] Commit GREEN:
  `feat(stage10): GREEN sqlite migrations and canonical bootstrap`.

### Checkpoint 10B — atomic round-trip и restart

- [x] Добавить tests:
  `TestStoreCommitAndLoad_RoundTripsCompleteSnapshot`,
  `TestStoreCommit_PreservesExtractionAndOverrideTimestamps`,
  `TestStoreCommit_PersistsSchedulerAndNextPortalID`,
  `TestStoreCommit_AssignsEventIDsInChronologicalOrder`,
  `TestStoreListEvents_GlobalAndPortalUseSameRows`,
  `TestStoreCommit_RollsBackStateWhenEventInsertFails`,
  `TestStoreLoad_RecoversOverdueStateWithoutResolvingIt`,
  `TestSchema_HasNoDerivedRealtimeColumns`.
- [x] RED commit:
  `test(stage10): RED atomic persistence and restart recovery`.
- [x] Реализовать codecs, `Load`, `Commit`, `ListEvents`; invalid decoded enum,
  timestamp или JSON должен вернуть ошибку, а не silently normalize.
- [x] Закрыть все prepared statements/rows; настроить SQLite foreign keys,
  busy timeout и один writer connection.
- [x] Выполнить focused persistence suite и полный обычный suite.
- [x] `PERSIST-001`, `PERSIST-002` → GREEN; `PERSIST-003`, `PERSIST-004` →
  PARTIAL до manager startup/meaningful-write proof.
- [x] Worklog + GREEN commit:
  `feat(stage10): GREEN atomic sqlite persistence and recovery`.

Stage 10 не запускает ticker, HTTP server или WebSocket.

## 7. Stage 11 — LabManager / concurrency

### Checkpoint 11A — ownership, resolve-first, atomic commit

- [x] Заменить skeleton `manager.go`; добавить `repository.go` и
  `manager_commands.go`.
- [x] Создать controllable fake repository и tests:
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
- [x] RED commit:
  `test(stage11): RED manager ownership and resolve-first transactions`.
- [x] Реализовать clone/apply/commit/publish template и все команды.
- [x] Не откатывать catch-up state при domain rejection; возвращать исходную
  domain error после успешного persistence.
- [x] Commit GREEN:
  `feat(stage11): GREEN manager ownership and resolve-first transactions`.

### Checkpoint 11B — ticker, signals и races

- [x] Добавить tests:
  `TestManagerRun_ConsumesInjectedTicksUntilContextCancel`,
  `TestManagerTick_SignalsEvenWithoutMeaningfulDatabaseWrite`,
  `TestManagerAction_SignalsAfterSuccessAndRejection`,
  `TestManagerUpdates_CoalesceWithoutBlocking`,
  `TestManagerConcurrentCommands_OnlyOneTransitionWins`,
  `TestManagerConcurrentTicks_DoNotDuplicateEvents`,
  `TestManagerConcurrentOpenings_KeepUniqueIDsAndSlots`,
  `TestManagerConcurrentReadsAndWrites_ReturnConsistentSnapshots`.
- [x] RED commit:
  `test(stage11): RED manager ticker updates and concurrency`.
- [x] Реализовать `Run`, coalescing signal и synchronization.
- [x] Запустить:
  `go test -count=1 ./internal/engine`,
  `go test -race -count=1 ./internal/engine`,
  `go test -count=1 ./...`.
- [x] `EVENT-001`, `EVENT-003`, `EVENT-010`, `PERSIST-003`, `PERSIST-004`,
  `SIMULATION-001`, `PORTAL-001`, `PORTAL-002`, `WS-002` backend boundary
  обновить до фактически доказанного GREEN/PARTIAL.
- [x] Worklog + GREEN commit:
  `feat(stage11): GREEN serialized lab manager and simulation loop`.

Stage 11 не содержит HTTP status codes или JSON DTO.

## 8. Stage 12 — REST API

### Checkpoint 12A — authoritative read DTOs

- [x] Подключить chi v5.
- [x] Создать `internal/transport/dto.go`, DTO tests и read handlers.
- [x] Tests:
  `TestBuildStateSnapshot_AlwaysReturnsSevenFixedSlots`,
  `TestBuildStateSnapshot_DerivesCurrentValuesAtGeneratedAt`,
  `TestBuildStateSnapshot_DoesNotExposeHiddenFields`,
  `TestBuildPortalDetails_OpenIncludesRiskRecommendationAndHistory`,
  `TestBuildPortalDetails_TerminalHasNullRiskAndRecommendation`,
  `TestBuildStateSnapshot_SlotsNeverContainRecommendation`,
  `TestGetState_ReturnsAuthoritativeSnapshot`,
  `TestGetPortal_ReturnsDetailsOr404`,
  `TestGetEvents_ReturnsChronologicalGlobalLog`,
  `TestReadEndpoints_DoNotAdvanceTutorial`.
- [x] RED commit: `test(stage12): RED REST reads and public DTO contract`.
- [x] Реализовать DTO builder/router/read handlers.
- [x] GREEN commit: `feat(stage12): GREEN REST reads and public DTO contract`.

### Checkpoint 12B — deterministic Recommendation Engine

- [x] Создать `internal/domain/recommendation_test.go` как table-driven suite:
  `TestRecommendation_TerminalHasNoValue`,
  `TestRecommendation_ExtractionBeforeSyncLeavesOpen`,
  `TestRecommendation_ActiveSafeTransitLeavesOpen`,
  `TestRecommendation_ActiveUnsafeTransitStabilizesOnlyWhenHelpful`,
  `TestRecommendation_ActiveTransitNeverCloses`,
  `TestRecommendation_WaitingObserverRecalls`,
  `TestRecommendation_WaitingObserverWaitsForSafeCorridor`,
  `TestRecommendation_WaitingObserverStabilizesForSafeReturn`,
  `TestRecommendation_ExploringObserverPreservesInboundPath`,
  `TestRecommendation_OutboundPathIsNotPreservedForRecall`,
  `TestRecommendation_UnexploredPlaneSendsAvailableObserver`,
  `TestRecommendation_DoesNotSendToExploredOrOccupiedPlane`,
  `TestRecommendation_OtherOutboundToSamePlaneBlocksDuplicateSend`,
  `TestRecommendation_ExactDeadlineTieIsUnsafe`,
  `TestRecommendation_DangerFallbackClosesWhenAffordable`,
  `TestRecommendation_InsufficientLabEnergyLeavesOpen`,
  `TestRecommendation_HypotheticalStabilizeDoesNotMutateState`,
  `TestRecommendation_DoesNotConsumeRandom`,
  `TestRecommendation_HiddenCollapseTimestampDoesNotAffectResult`,
  `TestRecommendation_DoesNotRestrictDomainCommands`.
- [x] Добавить REST assertions: Details возвращает enum/null; state/slots не
  содержат JSON key `recommendation`.
- [x] Запустить
  `go test -count=1 ./internal/domain ./internal/transport ./internal/httpapi -run 'TestRecommendation|TestBuildPortalDetails'`
  и получить RED по отсутствующему domain type/function/DTO field.
- [x] RED commit:
  `test(stage12): RED safe mission recommendation decision table`.
- [x] Реализовать exact priority table из Final Spec §23.1 чистыми helpers:
  entity lookup, portal/observer horizon, hypothetical Stabilize eligibility и
  close fallback. Не вызывать command на реальном aggregate.
- [x] Добавить `recommendation *domain.Recommendation` только в Details DTO.
- [x] Повторить focused command; expected PASS.
- [x] GREEN commit:
  `feat(stage12): GREEN deterministic recommendation engine`.

### Checkpoint 12C — commands, strict JSON и errors

- [x] Реализуемые routes Stage 12:
  `POST /api/portals/{id}/stabilize`, `/close`, `/send-observer`,
  `/recall-observer`, `POST /api/extraction/open`.
- [x] Bodies: `{}` или `{"confirm": boolean}` для portal commands;
  `{"plane_id": integer}` для extraction. Unknown fields и второй JSON value
  отклоняются как 400.
- [x] Tests:
  `TestPortalCommandRoutes_ReturnFreshState`,
  `TestCloseConfirmationFlow_Returns409ThenSucceeds`,
  `TestDomainConflictMapping`,
  `TestCommandRoute_UnknownPortalIs404`,
  `TestExtractionRoute_UnknownPlaneIs404`,
  `TestCommandRoute_StrictJSONRejectsUnknownAndTrailingValues`,
  `TestMalformedRequest_DoesNotCreateActionRejected`,
  `TestWrongMethod_Returns405`,
  `TestInternalFailure_ReturnsOpaque500`.
- [x] RED commit: `test(stage12): RED REST commands and domain errors`.
- [x] Реализовать handlers/error mapper без дублирования domain rules.
- [x] Проверить `go test -count=1 ./internal/transport ./internal/httpapi`
  и весь suite.
- [x] API-001..008, API-010, API-011 → GREEN; API-009/API-012 остаются PLANNED
  до Stage 14. RECOMMENDATION-001,002,004..012 → GREEN;
  RECOMMENDATION-003 остаётся PARTIAL до фактического rendering в Stage 17.
- [x] Worklog + GREEN commit:
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

### Checkpoint 14A — persisted context, Step 0 и prepared Steps 1–5

- [ ] Расширить migration 002 колонками `tutorial_phase`,
  `tutorial_portal_id`, `tutorial_plane_id`, `tutorial_observer_id`; добавить
  round-trip и migration idempotency tests.
- [ ] Добавить `TutorialSignal`, `TutorialPhase`, `TutorialExpectedAction`,
  `ExpectedTutorialAction(app AppState)` и prepared factory helpers в
  `internal/domain/tutorial.go`.
- [ ] Tests:
  `TestTutorial_TicksDoNotAdvanceStep0OrCreatePortals`,
  `TestTutorial_IntroSignalAtomicallyCreatesOneStep1TargetAndEvent`,
  `TestTutorial_IntroSignalReplayIsRejectedWithoutDuplicate`,
  `TestTutorial_DetailsSignalRequiresMatchingTargetID`,
  `TestTutorial_ReadsNeverAdvanceProgress`,
  `TestTutorial_Step1PreparedPortalHasSafeCorridorAndSendProperties`,
  `TestTutorial_Step2AdvancesOnlyWhenCreaturesReachZero`,
  `TestTutorial_Step3UsesNormalSendAndTracksObserverPlane`,
  `TestTutorial_Step4PreparedPortalSatisfiesStabilizeContract`,
  `TestTutorial_StabilizeAllowsHighOrCriticalToMediumOrLow`,
  `TestTutorial_Step5CriticalPortalEndsByNaturalCloseNotCollapse`,
  `TestTutorial_CriticalRejectedSendCommitsActionRejectedAndStep6`,
  `TestTutorial_WrongReversibleActionKeepsStep`,
  `TestTutorial_TerminalTargetRecreatesEquivalentFreshID`,
  `TestTutorial_TickNeverSpawnsNaturalPortal`,
  `TestTutorialContext_RoundTripsStepPhaseAndTargetsAcrossRestart`.
- [ ] Запустить
  `go test -count=1 ./internal/domain ./internal/persistence ./internal/engine -run 'TestTutorial|TestTutorialContext'`
  и получить RED по отсутствующим types/state-machine behavior.
- [ ] RED commit:
  `test(stage14): RED tutorial context intro and prepared steps one to five`.
- [ ] Реализовать mode-aware tick: normal lifecycle/events без вызова Natural
  spawn; intro signal создаёт первый Portal, а Step 2 auto-progress выполняется
  только при creatures = 0.
- [ ] Prepared factory принимает semantic profile, создаёт новый sequential ID
  и проверяет перечисленные свойства; production code не зависит от fixture ID.
- [ ] Critical rejection одной transaction сохраняет catch-up,
  `ACTION_REJECTED`, Step 6/phase и возвращает исходный domain error для 409.
- [ ] Повторить focused command; expected PASS.
- [ ] GREEN commit:
  `feat(stage14): GREEN tutorial context intro and prepared steps one to five`.

### Checkpoint 14B — Step 6 phases, real retry и Step 8

- [ ] Добавить tests:
  `TestTutorial_EnterStep6DerivesWaitResearchPhase`,
  `TestTutorial_EnterStep6WaitingObserverCreatesSafeReturnPortal`,
  `TestTutorial_Step6ResearchCompletionCreatesFreshSamePlanePortal`,
  `TestTutorial_Step6RecallUsesLongestWaitingAndAdvancesStep7`,
  `TestTutorial_ReturnExploresPlaneOnlyAfterObserverReturned`,
  `TestTutorial_LostReturnUsesDifferentAvailableObserver`,
  `TestTutorial_LostRetryRequiresNormalSendResearchRecall`,
  `TestTutorial_LostRetryNeverTeleportsObserverToPlane`,
  `TestTutorial_EarlierTrackedLossEntersSendReplacementPhase`,
  `TestTutorial_NoReplacementObserverStaysRecoverableUntilReset`,
  `TestTutorial_EventLogGETDoesNotAdvanceStep8`,
  `TestTutorial_EventLogSignalAdvancesStep8ToStep9`,
  `TestTutorial_PhaseAndTargetRestartDoesNotDuplicatePortalOrEvents`.
- [ ] RED commit:
  `test(stage14): RED tutorial research return and honest lost retry`.
- [ ] Реализовать phase transitions:

```text
SEND_REPLACEMENT --successful SEND--> WAIT_RESEARCH
WAIT_RESEARCH --WAITING_RETURN--> create safe same-plane Portal → RECALL_READY
RECALL_READY --successful RECALL--> Step 7 / WAIT_RETURN
Step 7 --LOST--> Step 6 / SEND_REPLACEMENT
Step 7 --AVAILABLE + EXPLORED--> Step 8 / OPEN_EVENT_LOG
```

- [ ] Replacement выбирает AVAILABLE Observer обычным lowest-ID rule и всегда
  отличается от LOST terminal Observer. Все status changes проходят через
  normal domain commands/lifecycle и создают normal Events.
- [ ] Повторить focused engine/domain/persistence tests; expected PASS.
- [ ] GREEN commit:
  `feat(stage14): GREEN tutorial research return and honest lost retry`.

### Checkpoint 14C — Tutorial API, reset и Live continuity

- [ ] Расширить repository:

```go
ResetTutorial(context.Context, persistence.Snapshot) error
```

  Transaction удаляет prior Portals/Events, полностью заменяет Plane/Observer/
  Lab/App state и откатывается целиком при injected failure.
- [ ] Добавить manager methods:

```go
func (m *LabManager) StartTutorial(context.Context) error
func (m *LabManager) ResetTutorial(context.Context) error
func (m *LabManager) TutorialSignal(context.Context, domain.TutorialSignal, *int64) error
func (m *LabManager) StartLive(context.Context) error
```

- [ ] `POST /api/tutorial/signal` strict body:

```json
{"signal":"PORTAL_DETAILS_OPENED","portal_id":42}
```

  `portal_id` обязателен ровно для `PORTAL_DETAILS_OPENED`; intro/event-log
  signals запрещают это поле. Unknown enum и shape → 400 transport validation
  без Event; valid unexpected signal → 409 + `ACTION_REJECTED`.
- [ ] Tests:
  `TestTutorialStart_IsIdempotent`,
  `TestTutorialReset_RestoresEnergyObserversPlanesAndClearsHistory`,
  `TestTutorialReset_TransactionFailureRollsBackEverything`,
  `TestStartLive_BeforeStep9IsRejected`,
  `TestStartLive_PreservesEnergyEventsObserversAndExploration`,
  `TestStartLive_ClosesTutorialPortalsForFreeAndStartsZeroOpen`,
  `TestStartLive_SchedulesNaturalGenerator`,
  `TestTutorialSignal_StrictShapeAndClosedEnum`,
  `TestTutorialSignal_UnexpectedValidSignalCreatesActionRejected`,
  `TestTutorialSnapshot_ExposesStepPhaseTargetsAndExpectedAction`,
  `TestTutorialSnapshot_DoesNotExposePreparedHiddenValues`,
  `TestTutorialAPI_GETsNeverMutateProgress`,
  `TestTutorialAPI_CriticalRejectionPersistsProgressAndEvent`,
  `TestTutorialWebSocket_ReconnectShowsPersistedStepAndPhase`,
  `TestTutorial_FullColdStartRestartAndLiveJourney`.
- [ ] RED commit:
  `test(stage14): RED tutorial API reset and live continuity`.
- [ ] Реализовать atomic reset, strict signal handler, Live free cleanup и fresh
  scheduler. Старые Events удаляет только reset; start/Live сохраняют history.
- [ ] REST и WS используют один DTO для step/phase/target/expected action.
- [ ] Выполнить focused domain/engine/persistence/httpapi/realtime suites и
  затем полный обычный suite.
- [ ] TUTORIAL-001..015, TUTORIAL-017, TUTORIAL-018, API-009, API-012,
  LAB-002 → GREEN. TUTORIAL-016 → PARTIAL по backend contract;
  TUTORIAL-019 → PLANNED до Stage 20 UI copy/rendering.
- [ ] Worklog + GREEN commit:
  `feat(stage14): GREEN tutorial API reset and live continuity`.

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
  Recommendation backend rows GREEN, а UI-only `RECOMMENDATION-003` остаётся
  PARTIAL до Stage 17.
- [ ] Дополнить `01_AI_WORKLOG_CURRENT.md`: стадии, решения, ошибки,
  verification, RED/GREEN hashes и вклад пользователя.
- [ ] Сделать финальный docs/verification commit без изменения semantics.

## 14. Definition of Done Block C

- [ ] Stage 9: полный deterministic event stream, shared history и rejection
  drafts доказаны tests.
- [x] Stage 10: schema, 85-plane seed, atomic persistence и restart recovery
  доказаны tests.
- [x] Stage 11: resolve-first manager, exactly-one transitions, ticker и race
  safety доказаны tests.
- [x] Stage 12: reads/commands/errors и полная deterministic Recommendation
  decision table реализованы; значение присутствует только в Portal Details.
- [ ] Stage 13: authoritative initial/tick/action snapshots, reconnect и
  backpressure доказаны tests.
- [ ] Stage 14: весь backend Tutorial 0–9, phases, prepared system transitions,
  retry/recreate/reset/restart/Live continuity доказаны unit/integration tests;
  contextual UI copy честно остаётся Stage 20.
- [ ] Обязательные gofmt/vet/build/test/race проходят с exit 0.
- [ ] Requirements, traceability и worklog соответствуют коду и tests.
- [ ] Git history содержит различимые RED/GREEN commits Stages 9–14.
- [ ] Worktree чист.
- [ ] Stage 15 не начат; работа останавливается для пользовательской сверки.
