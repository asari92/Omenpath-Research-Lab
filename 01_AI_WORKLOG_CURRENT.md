# AI Worklog — Current
## Stage: Blocks A–D / Stages 0–21 implementation

> Это честный журнал процесса. Его нужно дополнять по мере реализации. Не переписывать задним числом под «идеальную историю».

## AI tools

На этапе анализа и проектирования основной AI-инструмент — ChatGPT.

Использование:
- разбор задания;
- сравнение вариантов;
- обсуждение domain-модели;
- поиск edge cases;
- проверка противоречий;
- исследование сеттинга;
- архитектура;
- формализация итогового ТЗ и планов.

Точное количество токенов не фиксировал: интерфейс не предоставляет удобную общую статистику по всему диалогу.

Общее время разработки будет зафиксировано отдельно после завершения проекта.

## Как формировалось решение

### Выбор варианта
Выбран вариант «Лаборатория нестабильных порталов». AI помог сравнить его с вариантом привидений; решение принял я.

### Frontend-only → backend
Первое предложение AI было сделать небольшой frontend-only React demo. Я отказался: после обсуждения стало ясно, что основная ценность проекта — domain rules, state transitions, realtime simulation и persistence.

Сначала backend планировался на FastAPI, позже стек был пересмотрен.

### Live simulation
Идею живой системы предложил я: порталы появляются, теряют энергию, закрываются и иногда аварийно схлопываются.

Я сначала думал про cron. AI правильно указал, что для случайных секундных интервалов лучше постоянный simulation loop.

### Tutorial + Live
Идею Tutorial перед Live Mode предложил я. Позже Tutorial был сделан state machine: неверное действие не завершает step, а сломанный scenario пересоздаётся.

### Сеттинг
AI сначала предлагал собственную вселенную. Я хотел существующую.

Первым вариантом был Stargate; даже был собран dataset. Позже я заметил, что исходная тема магическая/сказочная, поэтому sci-fi сеттинг не подходит. AI это сначала не учёл.

После сравнения нескольких фэнтези-вселенных AI предложил Magic: The Gathering / Omenpaths. Для проекта собран dataset из 85 Planes.

### Plane vs Portal
Я поднял вопрос, является ли повторное соединение с одним миром тем же Portal.

Финально:
- Plane — постоянный мир;
- Portal — уникальный instance открытия;
- `Plane 1:N Portal`.

AI хорошо формализовал эту модель.

### Creatures
AI сначала предлагал считать потери/смерти существ при закрытии. Я убрал эту механику как искусственную.

Финально Creatures:
- занимают corridor;
- проходят по одному за 2 sec;
- блокируют Observer transit;
- Close требует warning;
- death statistics нет.

Позже я заметил boundary issue `TTL=10, creatures=5`: последнее существо заканчивает ровно в момент close. Добавлен `CREATURE_CLEARANCE_MARGIN=2 sec`.

### Observers
Идею сделать Observers ограниченным ресурсом предложил я.

Всего 10. Они физически перемещаются между Lab и Planes.

Я также предложил считать Plane EXPLORED только после успешного Return, а не после SEND/arrival/research.

### Observer model
AI сначала предложил слишком много отдельных timestamp fields. Я заметил, что Observer постоянно переиспользуется.

Модель упрощена до:
```text
status
current_plane_id
active_portal_id
phase_started_at
phase_ends_at
```

История хранится в Event Log.

### Observer flow
Идею фиксировать направление Portal предложил я.

Первое перемещение:
```text
NONE → OUTBOUND
```
или:
```text
NONE → INBOUND
```

После этого direction не меняется. AI помог оформить правило.

### Transit
AI предложил фиксированные 10 sec. Я заменил на random `5..15 sec`, иначе риск был слишком предсказуем.

### Portal Energy
Идею динамической Energy предложил я.

Energy decay скрыт; user видит realtime значение и сам оценивает скорость.

AI предложил не писать значение в DB каждую секунду, а вычислять его из baseline/time.

### Stability
Я предложил бинарные STABLE/UNSTABLE вместо числового параметра.

AI предложил удачную механику: UNSTABLE Portal при создании получает скрытый `instability_collapse_at`; Stabilize его отменяет.

### Stabilize
Финально:
```text
cost 20
UNSTABLE → STABLE
Portal Energy +15
```

Я поднял overcharge edge case. Вместо случайного взрыва сделано restriction: при Energy >85 Stabilize запрещён.

### Risk
Первые формулы AI складывали Energy/Time/Stability как проценты. Мне это не понравилось: величины плохо сопоставлялись.

Я предложил перевести Energy в lifetime.

AI помог оформить:
```text
energy_lifetime = current_energy / decay
effective_lifetime = min(scheduled_remaining, energy_lifetime)
base_risk = max(0, (45-effective_lifetime)/45*100)
UNSTABLE +20
```

Hidden instability timer намеренно не используется в Risk.

### Laboratory Energy
Идею отдельной Lab Energy предложил я.

Финальные costs:
```text
SEND 0
RECALL 0
CLOSE 5
STABILIZE 20
EXTRACTION 30
```

Ранний Extraction cost 60 был слишком дорогим и был снижен.

### Extraction
Идею ручного Extraction Portal предложил я, чтобы Observer не зависел полностью от случайного Natural Portal.

AI сначала предлагал автоматически вернуть всех Observers. Я оставил automatic только первого; следующих user возвращает вручную.

### CLOSED vs COLLAPSED
В одной итерации AI предложил считать любое автоматическое завершение Collapse. Я отклонил это.

Финально:
```text
CLOSED: NATURAL_CLOSE / MANUAL_CLOSE
COLLAPSED: ENERGY_DEPLETED / INSTABILITY
```

### Leyline Override
Идею Emergency Mode после Collapse предложил я, чтобы не возникало death spiral после `Lab Energy=0`.

AI нашёл тематическую MTG-отсылку и предложил название Leyline Override.

### 7 Portal Slots
Идею 7 постоянных slots предложил я как MTG-отсылку.

AI сначала предлагал сортируемый список. Вместо сортировки сохранены стабильные позиции и добавлен `Needs Attention`.

### Dashboard vs Details
Я решил, что Risk/Recommendation/History должны быть только в Portal Details, а Dashboard — оперативный.

Quick Actions доступны и там, и там.

### Realtime architecture
Сначала обсуждался polling. Я предложил WebSocket.

AI развил решение:
- active state in-memory;
- REST commands;
- WS broadcast;
- SQLite persistence/audit only.

### FastAPI → Go
Backend сначала был FastAPI.

После роста concurrency/realtime механики я решил перейти на Go:
```text
Go + chi + WebSocket + SQLite/database/sql
```

Причины: goroutines, timers, WebSocket clients, simulation, state transitions и мой выбор более естественной concurrency model.

### TDD
После фиксации domain-модели я предложил TDD.

AI предложил Clock/Random abstractions:
- FakeClock вместо sleep;
- FakeRandom вместо flaky random tests;
- requirement → test → RED → implementation → GREEN → refactor.

Это принято.

## Мои ключевые решения

1. Live Simulation.
2. Tutorial перед Live.
3. Использование существующей fantasy-вселенной.
4. Отказ от Stargate.
5. Observers как ресурс.
6. EXPLORED только после Return.
7. Multiple Observers per Plane.
8. One-way Portal Flow.
9. Random transit 5–15 sec.
10. Dynamic hidden-decay Portal Energy.
11. Laboratory Energy.
12. Extraction.
13. Automatic first extraction only.
14. Risk via effective lifetime.
15. Seven fixed Portal Slots.
16. CLOSED vs COLLAPSED.
17. Creature clearance margin.
18. Simplified Observer state.
19. Risk/Recommendation/History only in Details.
20. Actions both Dashboard and Details.
21. Tutorial retry.
22. Go backend.
23. TDD.

## Что AI сделал хорошо

- заменил cron на simulation loop;
- формализовал `Plane 1:N Portal`;
- придумал hidden instability timestamp;
- сравнил fantasy universes и предложил MTG/Omenpaths;
- помог собрать Plane dataset;
- формализовал Observer lifecycle/flow;
- математически оформил Risk;
- предложил WS broadcast + in-memory state;
- разделил REST commands / WS state;
- предложил SQLite не использовать как realtime engine;
- помог с concurrency/race concerns;
- формализовал Extraction;
- предложил Needs Attention;
- предложил lore-name Leyline Override;
- систематизировал domain errors;
- предложил FakeClock/FakeRandom для TDD.

## Где AI ошибался / что пришлось менять

- frontend-only initial architecture;
- слишком долго развивал Stargate;
- artificial creature deaths;
- несколько плохих Risk formulas;
- fixed 10 sec transit;
- excessive warnings;
- Extraction cost 60;
- automatic return of all observers;
- отдельная Observer page;
- идея automatic termination = COLLAPSED;
- сортировка Portals вместо fixed slots;
- overcomplicated Observer persistence.

## Будущие разделы Worklog

Analysis/architecture описаны выше. Ниже — фактическая запись Stage 0–1.

По мере кода добавить реальные разделы:
- Backend;
- TDD/tests;
- Concurrency bugs;
- Persistence;
- REST/WS;
- Frontend;
- Debugging;
- Deployment;
- Final QA.

Не придумывать задним числом ошибки AI или ручные исправления — записывать реальные.

## Stage 0–1 — реализация (2026-09-03)

### Что сделано

- `docs/requirements.md` — requirement catalog: 18 семейств ID (PLANE..TUTORIAL + API/WS/UI/PERSIST), типы INV/BEH/BAL/UI, ссылки на разделы Final Spec.
- `docs/traceability.md` — матрица `ID | Rule | Type | Test | Implementation | Status | Notes`; статусы PLANNED → RED → GREEN.
- `internal/config` — все balance-значения Final Spec §37 в `Config.Default()`; тест сверяет их со спецификацией (исполняемая часть Stage 0).
- Go-скелет: `cmd/server`, `internal/{domain,engine,clock,random,config}`, `testutil`.
- `clock.Clock` + `RealClock`; `testutil.FakeClock` (thread-safe, `Advance`, детерминированный `BaseTime` 2030-01-01 UTC).
- `random.Random` + `RealRandom` (math/rand/v2 + crypto seed, mutex); `testutil.FakeRandom` (queue, fail-loud panic при исчерпании).
- `testutil.PortalBuilder` — фикстуры (не production-конструктор); Unstable() требует явный hidden timestamp.
- Первый RED-цикл: PORTAL-005, ENERGY-005 (+PORTAL-007), PORTAL-009 — тесты написаны и закоммичены ДО реализации (RED: `ResolveLifecycle undefined`), затем минимальная реализация → GREEN.
- `go vet ./...`, `go test ./...`, `go test -race ./...` — все зелёные; `gofmt` чисто.

### Инструменты

- Реализация: GLM (модель) через pi harness, по планам Stage 0–1.
- Часы/рандом тесты, каталог требований, скелет, минимальный Portal lifecycle.
- Точное потребление токенов из окружения недоступно — не фиксирую (не выдумываю).
- Время: одна сессия 2026-09-03.

### TDD-процесс

- RED зафиксирован отдельным коммитом `test(stage1): RED — first portal lifecycle tests...` — тесты не компилировались (метода не существовало).
- GREEN — следующим коммитом; только natural close + energy depletion + terminal immutability.
- Умышленно НЕ реализовано (Stage 2): instability collapse, Stabilize, Close primitive, creatures, risk, slots, фабрики порталов.

### Решения и наблюдения

- Go-модуль размещён в корне репозитория, а не во вложенном `omenpath-lab/` (дерево пакетов из плана сохранено): проще `go test ./...`, Makefile и README в будущем.
- `ResolveLifecycle(now) (changed bool, err error)` и семантическое время `ClosedAt` (момент события, а не ленивого тика) выбраны сразу по Stage 2 plan §9 design rule — чтобы не переделывать API на следующей стадии.
- Tie «energy depletion == natural close» решён в пользу NATURAL_CLOSE (Stage 2 plan Rule B) уже в минимальной версии; отдельный тест — Stage 2 substage 2.4.
- FakeRandom возвращает значения как есть (без clamp) и паникует при исчерпании очереди — фикстуры fail-loud, скрытой нормализации нет.
- FakeClock и RealRandom покрыты конкурентными тестами под `-race` заранее — это фундамент для LabManager (Stage 11). Уточнение после аудита: у FakeRandom конкурентного теста нет (и не планировался — фикстура однопоточная); покрытие FakeRandom: порядок очереди, passthrough, panic при исчерпании.

### Ошибки AI / ручные правки на этом этапе

- `gofmt` нашёл неверное выравнивание комментариев в `internal/domain/portal.go` после первого коммита скелета — исправлено без изменения кода (единственная правка).
- Других ошибок/переделок не было; новых design-споров с AI на этом этапе не возникло.

## Stage 2 — Portal Core via TDD (2026-09-03)

### Что сделано (чекпоинты A–F из 04_STAGE_02_PORTAL_CORE_TDD.md)

- **A+B**: `ScheduledRemaining`, `CurrentEnergy`, `EnergyDepletionAt`; позднее разрешение сохраняет семантическое время; tie energy/natural → NATURAL_CLOSE (Rule B).
- **C**: hidden instability collapse в `ResolveLifecycle` (Rule C/D); примитив `Stabilize` (UNSTABLE→STABLE, hidden=null, current+15 → новый baseline, decay не трогается; граница 85 включительно).
- **D**: `MaxCreaturesForTTL`, `CreaturesInside`; примитив `Close` (confirmation при существах, MANUAL_CLOSE, terminal rejection).
- **E**: `EnergyLifetime`, `EffectiveLifetime`, `RiskScore`, `RiskLevel` (для terminal — `("", false)`); точные границы 25/50/75; agreed-примеры 14s→HIGH, 10s→CRITICAL; +20 UNSTABLE; cap 100; hidden не влияет на Risk.
- **F**: `FirstFreeSlot` (только OPEN занимают, первый свободный, чистая функция); фабрика `NewNaturalPortal` (TTL/energy/decay/stability/hidden/creatures по cfg+Random; без выбора plane/slot и побочных эффектов).

Каждый чекпоинт: тесты писались до реализации, RED наблюдался (compile-ошибки/фейлы), затем минимальная реализация → GREEN; после каждого — `go test ./...` и `go test -race ./...`; traceability обновлён в том же коммите.

### Реальные ошибки AI на этой стадии (честно)

- Фикстура `TestPortal_StabilizedPortalLosesInstabilityCandidate` изначально с `Energy(100)` — нарушала предусловие Stabilize (current > 85); домен корректно отклонил вызов, тест упал. Исправлена фикстура (Energy 50), не реализация — то самое поведение «нельзя подгонять тест под код наоборот».
- Сломанный doc-комментарий в `risk.go` (формулы без префикса `//` с юникод-минусом) — compile error.
- Условие внутри struct-литерала в `factory.go` (невалидный Go) — вынесено в переменную до литерала.
- Разорванная цепочка вызовов в `factory_test.go` (`QueueInt(0)` на отдельной строке) — цепочка восстановлена.
- Дважды мелкие gofmt-выравнивания после записи файлов.

Все ошибки пойманы компилятором/тестами немедленно, до коммита; в историю не попали.

### Решения и наблюдения

- Tie energy-depletion == hidden-instability (оба раньше natural close) в плане не специфицирован; зафиксирован детерминированно: ENERGY_DEPLETED выигрывает (StrictBefore-семантика кандидатов, NATURAL_CLOSE выигрывает точные тай с обоими по Rules B/C). Исправление после аудита: ранее здесь утверждалось, что из валидной фабрики такой тай «вообще не возникает» — это неверно: момент depletion (`opened_at + energy/decay`) может лежать строго внутри окна hidden `(opened+5s, close−1s)` и совпасть с ним точно. Тай возможен, семантика зафиксирована (ENERGY_DEPLETED выигрывает exact tie) и заперта regression-тестом `TestPortal_EnergyDepletionWinsInstabilityTie`.
- `MaxCreaturesForTTL`: деление Duration усекается к нулю, поэтому TTL<2s даёт 0 и без явного max(0,…) — guard оставлен как документация формулы; кейс TTL=1s покрыт тестом.
- `EnergyLifetime` при decay≤0 возвращает `math.MaxInt64` ns (~292 года) — безопасно внутри `min`; валидная фабрика такой decay не создаёт.
- Порядок draw в фабрике зафиксирован и задокументирован (TTL → energy → decay → roll → [hidden] → creatures) — детерминированные фикстуры FakeRandom.
- Stabilize проверяет именно derived current (тест: baseline 90, current 80 — разрешено; baseline 86 — отказ).
- Граница stage соблюдена: Portal не связан с LabState/Observer/Event; costs/подтверждения транзита — оркестрация позже.

### Верификация

- `gofmt -l` — пусто; `go vet ./...` — чисто; `go build ./...` — ок.
- `go test -count=1 ./...` и `go test -race -count=1 ./...` — все зелёные (46 тестов/подтестов в domain, 67 суммарно).
- В domain-тестах нет `time.Sleep`, реального рандома, БД, HTTP — только FakeClock/FakeRandom/builder.
- Traceability: все требования семей PORTAL/ENERGY/STABILITY/CREATURE/RISK/SLOT в скоупе Stage 2 — GREEN; осознанно PLANNED остались зависящие от будущих стадий (CREATURE-007, RISK-011..013, SLOT-007, PORTAL-001).

## Corrective pass после независимого аудита Stage 0–2 (2026-09-03)

### Что найдено и исправлено

1. **Terminal derived state (PORTAL-009 расширен на derived-значения).** После CLOSED/COLLAPSED портал продолжал «жить» как активный: `ScheduledRemaining` тикал, `CurrentEnergy` продолжала убывать, `CreaturesInside` дорастали до нуля. Исправлено TDD (сначала RED — 5 тестов упали, затем минимальная реализация → GREEN):
   - `ScheduledRemaining` для terminal = 0 (даже если `scheduled_close_at` в будущем);
   - `CurrentEnergy` замораживается на значении в `ClosedAt`;
   - `CreaturesInside` замораживается на значении в `ClosedAt`;
   - `RiskLevel` для terminal по-прежнему отсутствует (без изменений, S2-D2).
   Новые тесты: `TestPortal_ScheduledRemainingIsZeroWhenTerminal`, `TestPortal_EnergyFreezesAtManualClose`, `TestPortal_EnergyFreezesAtCollapse`, `TestPortal_CreaturesFreezeAtManualClose`, `TestPortal_CreaturesFreezeAtCollapse`.
2. **Tie ENERGY_DEPLETED == INSTABILITY.** Утверждение worklog/комментариев «тай невозможен из валидной фабрики» было неверным (depletion-момент может лежать внутри hidden-окна). Семантика не менялась — ENERGY_DEPLETED выигрывает exact tie; добавлен regression-тест `TestPortal_EnergyDepletionWinsInstabilityTie` (сразу GREEN — фиксация семантики, а не фикс бага), решение задокументировано в Stage 2 plan (Rule E note) и комментарии `ResolveLifecycle`.
3. **Недостоверные утверждения этого worklog** (см. правки выше): инструмент — GLM через pi, а не Claude Code; конкурентные тесты есть у FakeClock/RealRandom, но не у FakeRandom; утверждение о невозможности tie — неверно.
4. **Traceability введён статус PARTIAL** — «покрыт чистый helper/подмножество, полная оркестрация позже». Понижены с GREEN: SLOT-004, SLOT-006, PORTAL-002 (helper не доказывает инвариант целиком). PORTAL-004 повышен с PLANNED до GREEN — все три статуса и разные outcomes теперь прямо покрыты freeze-тестами. Вторым docs-проходом (по ревью) дополнительно понижены до PARTIAL: SLOT-001 (Dashboard ещё нет), ENERGY-002 и STABILITY-004 (скрытость от пользователя — API/UI Stage 12/13); example-таблица в Stage 2 plan §26 синхронизирована (SLOT-006 → PARTIAL); Rule D переформулирован однозначно (Natural Close выигрывает любой exact tie с её участием; ENERGY_DEPLETED выигрывает exact tie с INSTABILITY); опечатки в traceability исправлены.
5. **Orchestration invariant для LabManager зафиксирован** в roadmap (Stage 11): перед любой time-sensitive command — сначала `ResolveLifecycle(portal, now)` под тем же lock; стал terminal → action не выполняется. Намеренно НЕ встроено в `Portal.Close`/`Portal.Stabilize`, чтобы не усложнить будущие lifecycle Events (Stage 9).

### Чего НЕ делалось (осознанно)

Observer lifecycle, Lab Energy orchestration, Events, REST, WebSocket, SQLite, Stage 3 — вне скоупа corrective pass.

### Верификация

`gofmt -l .` — пусто; `go vet ./...` — чисто; `go build ./...` — ок; `go test -count=1 ./...` и `go test -race -count=1 ./...` — все зелёные.

## Stage 3 — Observer Lifecycle via TDD (2026-09-04)

### Контекст и граница

- Исполнитель: Codex (GPT-5). Использованы repository inspection, shell/Go toolchain, Git и patch-based editing. Точное число токенов окружение не предоставляет — не фиксирую.
- До реализации перечитаны Final Spec, текущий Worklog, roadmap, планы Stage 0–2/Stage 3, requirements и traceability; просмотрены Go-код, тесты, `git log`, ключевые corrective commits и `git status`.
- Stage 0–2 и corrective/documentation passes подтверждены; baseline `gofmt`/vet/build/test/race был зелёным. `05_STAGE_03_OBSERVER_LIFECYCLE_TDD.md` действительно находился в `8a9dcd7`; Stage 3 кода до этой работы не было.
- Блокирующих противоречий Stage 3 plan с Final Spec не найдено. LOST field clearing сохранён как явно обозначенная Stage 3 canonicalization, а не выдан за прямое требование Final Spec.
- Создан короткий корневой `AGENTS.md` (`b10caa8`) с иерархией source of truth, stage/TDD guards и quality commands.

### Что реализовано

- `NewObserver` и `NewObserverRoster`: AVAILABLE в Laboratory, стабильные one-based IDs; default config даёт roster из 10.
- `Observer.StartOutbound` и `Observer.StartReturning`: строгие state-preconditions, атомарные отказы, отдельный единственный random draw `5..15 sec` на каждый transit.
- `ResolveObserverLifecycle`: OUTBOUND arrival, 20-sec EXPLORING, WAITING_RETURN с сохранённым waiting timestamp, RETURNING success, LOST и multi-phase catch-up за один вызов.
- Все transition timestamps берутся из effective deadlines, не из времени позднего resolver call.
- Только успешный RETURNING→AVAILABLE исследует Plane; повторный return сохраняет первый `ExploredAt`.
- CLOSED/COLLAPSED строго раньше transit deadline делает Observer LOST в `Portal.ClosedAt`; exact tie `ClosedAt == PhaseEndsAt` успешен; более позднее закрытие не действует ретроактивно.
- LOST canonicalization очищает `CurrentPlaneID`, `ActivePortalID`, `PhaseStartedAt`, `PhaseEndsAt`; LOST не реанимируется и не может начать новый transit.
- Два Observers могут одновременно находиться в одном Plane; characterization не вводит global uniqueness или Stage 4 Portal-busy policy.

Добавлено 66 top-level Observer tests в семи checkpoint-файлах плюс construction tests; все обязательные test names из Stage 3 plan присутствуют.

### RED / GREEN history

| Checkpoint | RED evidence | GREEN evidence |
|---|---|---|
| A — construction/invariants | `71562a0` — undefined `NewObserver`, `NewObserverRoster`, `IsTerminal` | `8174a5e` |
| B — Start OUTBOUND | `be0d52f` — undefined `StartOutbound` / errors | `336b9f6` |
| C — OUTBOUND resolution | `6fce895` — undefined `ResolveObserverLifecycle` / invariant error | `bfc1619` |
| D — research | `4393c9e` — EXPLORING returned invariant error; invalid WAITING state was silently accepted | `1dff235` |
| E — return | `2f59bc4` — undefined `StartReturning` / return error | `7632806` |
| F — LOST | `5b9828b` — terminal Portal incorrectly produced EXPLORING/AVAILABLE and explored Plane | `c099eed` |
| G — ordering/catch-up | `fe6ca70` — one call stopped at EXPLORING; repeated resolve still changed state | `5e1391f` |
| H — multiple Observers | already GREEN characterization (no new production behavior required) | `b20cb9e` |

Checkpoint G's exact-tie and non-retroactive-close cases were already compatible with the checkpoint F strict-before implementation; the genuine RED was the missing multi-phase catch-up/idempotence behavior. Checkpoint H was intentionally recorded as a GREEN characterization rather than faking a failure.

### Реальные ошибки/операционные замечания

- Первый новый compile after RED не мог записать Go build cache из sandbox; тот же targeted command был повторён с разрешённым `go test` и показал ожидаемые undefined-symbol compile failures. Это ограничение окружения, не defect проекта.
- Один широкий patch не применился из-за gofmt-выравнивания контекста в `errors.go`; patch был разбит и применён без частичной мутации.
- Первый combined `gofmt && go test` для checkpoint H снова попал в sandbox cache restriction; standalone approved `go test` подтвердил GREEN.
- После GREEN self-review не потребовал изменения product semantics.

### Traceability и verification

- `OBSERVER-002..014` переведены в GREEN; `OBSERVER-001` честно оставлен PARTIAL до permanence/persistence bootstrap.
- `PLANE-006..008` переведены в GREEN; `PLANE-005` оставлен PARTIAL до реального SEND command Stage 4, `PLANE-009` — PARTIAL до progress aggregation.
- `OBSERVER-015..016`, `FLOW-*` и весь Stage 4+ scope остались PLANNED.
- После checkpoints и перед этим worklog update выполнены: `gofmt -l .` (пусто), `go vet ./...`, `go build ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...` — exit 0.
- Scope checks: в `internal/domain/observer*.go` нет `time.Now`/Sleep/ticker/after; lifecycle не зависит от Portal Flow, Risk, creatures, Lab Energy, Extraction или Event system.

Stage 4 не начат.

## Stage 4 — Portal Observer Flow via TDD (2026-09-04)

### Контекст и граница

- Исполнитель: Codex (GPT-5). Реализация выполнена строго по `06_STAGE_04_PORTAL_OBSERVER_FLOW_TDD.md` поверх Stage 2 Portal Core и Stage 3 Observer Lifecycle.
- Блокирующих противоречий с Final Spec §§15–19/21/29 не обнаружено.
- Реализованы только чистые domain-команды. `LabManager`, Laboratory Energy, Leyline Override, Extraction, Events, persistence, REST/WebSocket и simulation не начинались.
- Caller по-прежнему обязан сначала разрешить Portal/Observer lifecycle к `now`; будущий `LabManager` Stage 11 обеспечит resolve-first и сериализацию под lock.

### Что реализовано

- `AvailableObserverIndex`: SEND детерминированно выбирает AVAILABLE Observer с минимальным ID.
- `LongestWaitingObserverIndex`: RECALL фильтрует по destination Plane, выбирает самый ранний `PhaseStartedAt`, при точном tie — минимальный Observer ID.
- `ActiveTransitObserverIndex`: busy scope ограничен точным Portal ID; duplicate или stale transit возвращает invariant error.
- `SendObserver` и `RecallObserver`: общий порядок admission — structural invariants → OPEN → Risk != CRITICAL → direction → creatures == 0 → not busy → eligible Observer → UNSTABLE confirmation → mutation.
- Первый успешный SEND фиксирует `OUTBOUND`, первый успешный RECALL — `INBOUND`; rejected/confirmation-required action flow не меняет, completion/loss flow не сбрасывают.
- SEND разрешён в EXPLORED Plane и Plane с другими Observers; RECALL выбирает только WAITING_RETURN в destination Plane.
- UNSTABLE использует единый `ErrConfirmationRequired`; LOW/MEDIUM/HIGH, low energy, short remaining time и explored destination отдельного warning не создают.
- `ClosePortalWithObservers`: active OUTBOUND/RETURNING требует confirmation; подтверждённый Close переиспользует `Portal.Close` и `ResolveObserverLifecycle`, поэтому Portal становится CLOSED/MANUAL_CLOSE, а Observer — LOST в том же `now`.
- Один `confirm=true` покрывает одновременно creatures и active transit. Lab Energy cost намеренно не реализован.
- Aggregate preflight отклоняет nil/mismatched Portal/Plane, duplicate Observer IDs, неканонические state fields и stale transit до любой мутации/random draw.

Добавлено 98 top-level Stage 4 tests в шести файлах; все обязательные test names из checkpoints A–H присутствуют.

### RED / GREEN history

| Checkpoint | RED evidence | GREEN evidence |
|---|---|---|
| A — selection/busy helpers | `e67667b` — undefined selectors | `6689331` |
| B — successful SEND/first flow | `3dc2f15` — undefined `SendObserver` | `eb22a7c` |
| C — SEND restrictions/warning | `204877a` — missing Stage 4 errors/admission | `d34503d` |
| D — successful RECALL/longest waiting | `0ea2022` — undefined `RecallObserver` | `92ab740` |
| E — RECALL restrictions/warning | `6fdd193` — restrictions returned nil and mutated state | `5bf24dc` |
| F — permanent flow/next transit | already GREEN characterization over A–E behavior | `232611f` |
| G — active-transit manual Close | `71dd620` — undefined `ClosePortalWithObservers` | `bd25981` |
| H — aggregate atomicity/integration | `270ab23` — mismatched/duplicate state mutated; nil Portal panicked | `f883f23` |

Checkpoint F deliberately did not manufacture a RED failure: all permanent-flow and Portal-ID-scope characterizations already passed after checkpoints A–E.

### Реальные ошибки/операционные замечания

- Первый checkpoint A test run не смог писать в sandboxed `/home/asari/.cache/go-build`; повторные Go-команды использовали отдельный `GOCACHE` в `/tmp` и показали настоящий expected RED по отсутствующим symbols.
- Одна комбинированная shell-команда не получила разрешение на `.git/index.lock`; staging и commit были безопасно повторены отдельными разрешёнными `git add`/`git commit` вызовами. Файлы не повреждались.
- Checkpoint H поймал реальный atomicity defect промежуточной реализации: Close успевал мутировать Portal до обнаружения mismatched Plane; aggregate validation перенесён перед business admission и mutation.
- Product semantics завершённых Stage 2/3 не менялись.

### Traceability и verification

- `FLOW-001..007`, `CREATURE-007`, `RISK-012..013`, `OBSERVER-015..016`, `PLANE-005` переведены в GREEN с конкретными tests/symbols.
- `PORTAL-006` и `OBSERVER-012` дополнены active-transit manual-close coverage без изменения прежних semantics.
- `OBSERVER-001` и `PLANE-009` остаются PARTIAL; Stage 5+ requirements остаются PLANNED.
- После каждого meaningful GREEN выполнены `gofmt -l .`, `go vet ./...`, `go build ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...`; все завершились с exit 0.
- Scope scan новых Stage 4 domain-файлов не нашёл wall-clock calls, Lab Energy, Leyline, Extraction, database/HTTP или mutex dependencies.

Stage 5 не начат.

## Stage 5 — Laboratory Energy via TDD (2026-09-04)

### Контекст и граница

- Исполнитель: Codex (GPT-5). Точное число токенов окружение не предоставляет — не фиксирую.
- Реализация выполнена строго по `07_STAGE_05_LABORATORY_ENERGY_TDD.md` поверх завершённых Stage 2 Portal Core и Stage 4 Portal Observer Flow.
- Перед началом рабочее дерево было чистым, `main` указывал на `d538e97`; baseline `go test -count=1 ./...` прошёл полностью.
- Блокирующих противоречий с Final Spec §§14/21/22/37 не найдено: integer Energy, производная регенерация и цены 0/0/5/20/30 совпадают.
- Реализован только Stage 5 domain scope. Collapse/Leyline Override, Extraction Portal, Events, persistence, HTTP/WebSocket, simulation и LabManager не начинались.

### Что реализовано

- `NewLabState` валидирует inclusive baseline `0..cfg.LabEnergyMax`; `NewTutorialLabState` создаёт maximum-energy baseline в переданном `now` без override.
- `LabState.CurrentEnergy` чисто вычисляет целочисленную энергию по завершённым целым секундам, использует `cfg.LabRegenPerSec`, ограничивает результат максимумом и не записывает derived state.
- `LabState.CanAfford` и `SpendEnergy` используют derived balance; положительный debit ставит новый baseline в `now`, нулевая стоимость является настоящим no-op, malformed state/negative cost/backward mutation time отклоняются атомарно.
- Добавлены стабильные `ErrLabEnergyInvariant` и `ErrInsufficientLabEnergy`.
- `SendObserverWithLabEnergy` и `RecallObserverWithLabEnergy` валидируют LabState, но имеют цену 0 и не сдвигают baseline; существующие Stage 4 selection/flow/confirmation/error/random semantics делегируются без изменений.
- `ClosePortalWithLabEnergy` preflight-проверяет `cfg.CloseCost` (default 5), делегирует `ClosePortalWithObservers` и списывает один раз только после успеха.
- `StabilizePortalWithLabEnergy` аналогично применяет `cfg.StabilizeCost` (default 20) вокруг `Portal.Stabilize`; Lab Energy и Portal Energy остаются независимыми ресурсами.
- Цена Extraction 30 покрыта только через общий `SpendEnergy`: ни Extraction command, ни Portal creation/synchronization не добавлены.
- Добавлено 86 top-level Stage 5 tests в восьми специализированных файлах; все обязательные test names checkpoints A–H присутствуют.

### RED / GREEN history

| Checkpoint | RED evidence | GREEN evidence |
|---|---|---|
| A — construction/derived energy | `c79ee53` — undefined constructors, `CurrentEnergy` и invariant error | `b674075` |
| B — spend/affordability | `22549f6` — undefined `SpendEnergy`, `CanAfford`, insufficient error | `46bd5b4` |
| C — zero-cost SEND | `14baa2d` — undefined `SendObserverWithLabEnergy` | `7ee249c` |
| D — zero-cost RECALL | `4e2d9fa` — undefined `RecallObserverWithLabEnergy` | `4e7977d` |
| E — paid Close | `8ee2f66` — undefined `ClosePortalWithLabEnergy` | `a8c86c9` |
| F — paid Stabilize | `4defe33` — undefined `StabilizePortalWithLabEnergy` | `0c9e018` |
| G — Extraction cost contract | already GREEN characterization over config + checkpoint B primitive | `07a2a27` |
| H — transaction regressions | already GREEN characterization over checkpoints A–F | `dffe4d0` |

Checkpoints G/H намеренно не получили искусственный RED. Transaction characterizations не обнаружили нового production gap: insufficient preflight, domain/confirmation failures, zero-cost baseline preservation, error identity и random consumption уже были корректны.

### Решения и наблюдения

- Integer `+1/sec` реализован через floor до завершённых целых секунд: на `T+999ms` прироста нет, на `T+1s` есть.
- Pure read до `EnergyBaseAt` возвращает baseline без регенерации; mutation до baseline отвергается, чтобы время baseline не двигалось назад.
- Порядок платной команды: validate/derive → insufficient check → existing domain command → один synchronous debit. Поэтому отказ не требует rollback и не меняет Lab/Portal/Plane/Observers.
- Cost 0 не ребейзит даже полностью восстановленную энергию; SEND/RECALL сохраняют исходные `EnergyBase` и `EnergyBaseAt`.
- `LeylineOverrideUntil` production-код Stage 5 не читает и не изменяет. Обычные цены Close/Stabilize не содержат Stage 6 exception.
- Product semantics и signatures завершённых Stage 2–4 не менялись; новый слой состоит из тонких wrappers.

### Traceability и verification

- `LAB-001`, `LAB-003..008` переведены в GREEN; `LAB-002` честно PARTIAL до Tutorial bootstrap Stage 14; `LAB-009` PARTIAL до реальной Extraction Stage 7; `LAB-010` PARTIAL до override/Extraction paths Stages 6–7.
- `PORTAL-006`, `STABILITY-005`, `ENERGY-006..008`, `OBSERVER-012/016`, `FLOW-002..006` дополнены evidence energy-aware wrappers и сохраняют GREEN.
- `EMERGENCY-*` и `EXTRACTION-*` остаются PLANNED; `docs/requirements.md` семантически не менялся.
- После каждого meaningful GREEN выполнены `gofmt -l .`, `go vet ./...`, `go build ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...`; все завершились с exit 0.
- Scope scan `internal/domain/lab*.go` не нашёл wall-clock/ticker/sleep, database/HTTP/mutex, `OpenExtraction`, events или lifecycle resolution dependencies.

Stage 6 не начат.

## Stage 6 — Emergency / Leyline Override via TDD (2026-09-04)

### Контекст и граница

- Исполнитель: Codex (GPT-5). Реализация выполнена строго по
  `08_STAGE_06_EMERGENCY_LEYLINE_OVERRIDE_TDD.md` поверх завершённого Stage 5.
- Перед реализацией рабочее дерево было чистым, `main` указывал на `1f45d3d`;
  baseline `gofmt`/vet/build/test/race был зелёным.
- Блокирующих противоречий с Final Spec §§14/20/21/22/29/37 не найдено.
  Override использует полузакрытое окно `[collapseAt, collapseAt+duration)`, а
  позднее разрешение Collapse сохраняет семантический момент `Portal.ClosedAt`.
- Реализован только Stage 6 domain scope. Extraction Portal, Events,
  persistence, HTTP/WebSocket, simulation и LabManager не начинались.

### Что реализовано

- `LabState.ActivateLeylineOverride` атомарно обнуляет Laboratory Energy,
  ребейзит её в момент Collapse и заменяет deadline на настроенную длительность.
- `LabState.LeylineOverrideActive` является чистым derived query; начало окна
  выводится из deadline и `cfg.EmergencyDuration`, поэтому последующий обычный
  energy debit не сдвигает emergency window.
- `ResolvePortalLifecycleWithLabEmergency` использует copy-then-commit:
  запускает Override только при новом переходе в COLLAPSED, сохраняет причину и
  `ClosedAt` Portal, не реагирует на CLOSED и идемпотентен для terminal replay.
- Каждый новый хронологически допустимый Collapse снова обнуляет накопленную
  энергию и заменяет deadline; out-of-order/invalid состояние отклоняется без
  частичной мутации Lab или Portal.
- `ClosePortalWithLabEnergy` и `StabilizePortalWithLabEnergy` выбирают цену 0
  только внутри активного Override. Все creature/transit confirmations,
  stability/overcharge/terminal restrictions и atomic failure semantics
  сохранены. На точном deadline снова действуют обычные цены 5/20.
- Регенерация `+1/sec` продолжается во время Override и не прерывается
  бесплатными действиями. Extraction cost остаётся 30, но покрыт только как
  generic `SpendEnergy` contract; Stage 7 command/Portal не добавлены.
- Добавлено 88 top-level Stage 6 tests в восьми checkpoint-файлах.

### RED / GREEN history

| Checkpoint | RED evidence | GREEN evidence |
|---|---|---|
| A — activation/window | `d7340f8` — undefined `ActivateLeylineOverride` / `LeylineOverrideActive` | `b37883b` |
| B — energy Collapse orchestration | `0696e80` — undefined lifecycle wrapper | `b3bde94` |
| C — instability Collapse | already GREEN characterization over checkpoint B wrapper | `660a2b9` |
| D — repeated Collapse reset | already GREEN characterization over checkpoints A–C | `8c8787b` |
| E — free Close | `45e1f1d` — Stage 5 wrapper charged/rebased energy and rejected zero balance | `eb15243` |
| F — free Stabilize | `8b2112c` — Stage 5 wrapper rejected zero balance | `cb4a3ca` |
| G — regeneration/Extraction invariant | already GREEN characterization over implemented primitives | `c5140ae` |
| H — emergency regressions | already GREEN after correction of a test fixture; no production defect | `a15231e` |

Checkpoints C/D/G/H намеренно не получили искусственный RED. Checkpoint H
сначала использовал одно существо и `now=+5s`, когда corridor уже был пуст по
правилу `CreatureTransit=2s`; тест ошибочно ожидал confirmation. Минимальная
коррекция фикстуры на `now=+1s` воспроизвела требуемый failure path, после чего
targeted и полный suite прошли без изменения production-кода.

### Traceability и verification

- `EMERGENCY-001..003` и `EMERGENCY-005..006` переведены в GREEN;
  `EMERGENCY-004` честно оставлен PARTIAL до настоящего Extraction Stage 7.
- `PORTAL-007/008`, `LAB-003/007/008`, `STABILITY-005`, `ENERGY-006..008` и
  `OBSERVER-012` дополнены emergency evidence и остаются GREEN.
- `LAB-009` остаётся PARTIAL. `LAB-010` остаётся PARTIAL, но теперь явно
  фиксирует, что Override Close/Stabilize paths завершены и отсутствует только
  реальный Extraction insufficient-energy path Stage 7.
- `EXTRACTION-*` остаются PLANNED; `docs/requirements.md` семантически не
  менялся.
- После Checkpoint H выполнены `gofmt -l .` (пусто), `go vet ./...`,
  `go build ./...`, `go test -count=1 ./...` и
  `go test -race -count=1 ./...`; все завершились с exit 0.
- Scope scan production-файлов Stage 6 не нашёл wall-clock/ticker/sleep,
  database/HTTP/mutex, Extraction behavior symbols, Observer lifecycle,
  slot-selection или natural-portal generation dependencies.

Stage 7 не начат.

## Stage 7 — Extraction Portal via TDD (2026-09-04)

### Контекст и граница

- Исполнитель: Codex (GPT-5). Реализация выполнена строго по
  `09_STAGE_07_EXTRACTION_PORTAL_TDD.md` поверх завершённого Stage 6.
- Перед реализацией рабочее дерево было чистым, `main` указывал на `1838076`;
  baseline `gofmt`/vet/build/test/race был зелёным.
- Блокирующих противоречий с Final Spec §§14/20/21/22/29/37 не найдено.
  Уточнённая до реализации семантика зафиксирована в approved design
  `6965d26`: opening проверяет WAITING_RETURN в выбранном Plane, а sync заново
  выбирает текущего longest-waiting Observer без reservation.
- Реализован только Stage 7 domain scope. Events, persistence, HTTP/WebSocket,
  simulation tick, natural spawning и LabManager не начинались.

### Что реализовано

- `NewExtractionPortal` создаёт контролируемый Portal со статусом OPEN,
  kind=EXTRACTION, stability=STABLE, flow=INBOUND, creatures=0, Portal Energy
  60..100, decay 0.1..1.0 и TTL 30..60 sec.
- `ExtractionPlaneEligible` проверяет канонический Observer roster и требует
  хотя бы одного WAITING_RETURN Observer именно в выбранном Plane.
- `OpenExtractionPortal` атомарно валидирует eligibility/free regular Slot/Lab
  Energy, списывает 30 и добавляет Portal в первый свободный Slot. Отказы не
  мутируют агрегат и не расходуют randomness; Leyline Override не отменяет
  Extraction cost.
- `Portal.ExtractionSynchronizedAt` хранит один completion marker. Пятисекундное
  окно синхронизации полузакрыто: до `OpenedAt+5s` resolver является no-op, а
  при позднем вызове использует точный semantic deadline.
- `ResolveExtractionSynchronization` в момент sync заново выбирает текущего
  longest-waiting Observer (earliest waiting timestamp, затем lowest ID) и
  атомарно переводит его в RETURNING. Если первоначальный кандидат уже ушёл,
  выбирается следующий; если никто не ждёт, marker всё равно завершается без
  random draw, refund или автоматического возврата.
- После marker все resolver replays являются no-op, включая вызовы после
  transit deadline. Все дальнейшие возвраты выполняются обычным manual RECALL;
  до завершения sync он отвергается с `ErrExtractionSynchronizing`.
- Close до sync оставляет Observer в WAITING_RETURN. Close во время RETURNING
  наследует Stage 4 confirmation и при подтверждении делает Observer LOST;
  Collapse во время transit также даёт LOST. Marker не очищается.
- Добавлено 143 top-level Extraction tests в восьми checkpoint-файлах; все
  обязательные test names из Stage 7 plan присутствуют.

### RED / GREEN history

| Checkpoint | RED evidence | GREEN evidence |
|---|---|---|
| A — factory/model | `a566111` — отсутствовали factory и completion marker | `1a2f1f2` |
| B — Plane eligibility | `8c152a7` — отсутствовал eligibility helper | `f86049a` |
| C — atomic opening | `f1478c7` — отсутствовали open command/errors | `f87a79d` |
| D — synchronization marker | `3fda61a` — отсутствовал resolver | `e9945e8` |
| E — automatic return | `2c62cfa` — marker завершался без старта RETURNING | `d551a49` |
| F — manual RECALL gate | `a2a0b34` — pre-sync RECALL проходил и расходовал random | `c7fda1e` |
| G — Close/Collapse/loss | already GREEN characterization поверх Stage 3/4 lifecycle | `bde92b0` |
| H — integration/regressions | `2bff17e` — late replay ошибочно валидировал завершённый transit и возвращал invariant error | `c427b3d` |

Checkpoint G намеренно не получил искусственный RED: approved Close/Lost
semantics уже полностью следовали из завершённых Stage 3/4 primitives.
Checkpoint H corrective RED закрепил строгую one-shot idempotence: marker и
terminal/pre-deadline no-op теперь проверяются до roster validation, которая
нужна только при фактической незавершённой синхронизации.

### Реальные ошибки/операционные замечания

- Первый RED-тест automatic return разыменовывал отсутствующий `PhaseEndsAt` и
  паниковал вместо чистого assertion failure; fixture assertion был исправлен
  без production-изменений, после чего RED воспроизведён корректно.
- Первый RED-тест manual gate использовал пустой FakeRandom и паниковал, потому
  что отсутствующий guard дошёл до random draw. Recording random позволил
  чисто показать одновременно nil error и ошибочный расход randomness.
- При добавлении manual gate широкий текстовый patch сначала попал в SEND; это
  было замечено немедленной инспекцией diff и перенесено в RECALL до запуска
  тестов и до commit. Ошибочная семантика в history не попадала.
- Финальный self-review обнаружил late-replay defect, не покрытый исходным
  checkpoint-набором; он исправлен отдельной честной RED/GREEN парой.

### Traceability и verification

- `EXTRACTION-001..010`, `LAB-009..010`, `EMERGENCY-004` и `SLOT-007`
  переведены в GREEN с конкретными tests/symbols.
- `PORTAL-003/006`, `OBSERVER-013/016` и `FLOW-003` дополнены Extraction
  evidence без изменения semantics завершённых stages.
- `docs/requirements.md` семантически не менялся.
- После каждого meaningful GREEN выполнены `gofmt -l .`, `go vet ./...`,
  `go build ./...`, `go test -count=1 ./...` и
  `go test -race -count=1 ./...`; все завершились с exit 0.
- Scope scan production-файлов Stage 7 не нашёл wall-clock/ticker/sleep,
  Events, persistence, HTTP/WebSocket, simulation, natural spawn или
  LabManager dependencies.

Stage 8 не начат.

## Stage 8 — Simulation Engine via TDD (2026-09-04)

### Контекст и граница

- Исполнитель: Codex (GPT-5). Реализация выполнена строго по
  `10_STAGE_08_SIMULATION_TDD.md` поверх завершённого Stage 7.
- Рабочая ветка начата от чистого `main` на `d161442`. Baseline
  `gofmt`/vet/test/race прошёл; `go build ./...` в sandbox внешнего worktree
  не смог прочитать Git VCS metadata, а тот же build с разрешённым доступом
  завершился с exit 0. Это была особенность sandbox, не ошибка проекта.
- Блокирующих противоречий между Stage 8 plan, approved design и Final Spec
  §§3, 5–13, 22, 31–34, 37 не найдено. Сохранены утверждённые правила:
  inclusive delay 0..20 sec, максимум один Natural spawn за тик, hard cap
  семь OPEN Portals и fresh delay после освобождения Slot.
- Реализован только Stage 8 pure domain scope. Stage 9 Events, persistence,
  реальный ticker, LabManager/mutex, transport, snapshots и frontend не
  начинались.

### Что реализовано

- Добавлены `SimulationState`, `NaturalSpawnState`, `NaturalSpawnResult`,
  `SimulationTickResult`, `ErrSimulationInvariant`, `NewNaturalSpawnState`,
  `ResolveNaturalSpawn`, `NeedsAttentionPortalIndex` и
  `SimulationState.ResolveTick`.
- Natural Generator использует inclusive целосекундный delay 0..20, считает
  только OPEN Portals, при 7/7 очищает timer и ставится на паузу. Первый тик
  после освобождения Slot только рисует fresh delay; даже delay 0 не создаёт
  Portal до более позднего тика. Один тик создаёт максимум один Portal.
- Due spawn повторно проверяет cap, берёт первый свободный regular Slot,
  выбирает destination index из всех 85 Planes, допускает повторный Plane,
  использует `NextPortalID` и существующий `NewNaturalPortal` без изменения
  factory ranges/draw order. Исторические terminal записи и порядок слайсов
  сохраняются.
- Tick работает copy-then-commit и выполняет стадии строго в порядке:
  Portal lifecycle → Observer lifecycle по ID → Extraction synchronization
  по Portal ID → Natural spawn → Needs Attention по финальному состоянию.
  Cross-Portal Collapse применяется по semantic `ClosedAt`, затем Portal ID,
  поэтому slice order не влияет на последний Leyline reset.
- Due Observer phases разрешены structural preflight и догоняются существующим
  `ResolveObserverLifecycle`; Portal termination раньше transit корректно даёт
  LOST, а успешный return исследует Plane в semantic deadline.
- Extraction сохраняет Stage 7 one-shot marker и на sync заново выбирает
  текущего longest-waiting Observer. Research completion в этот же timestamp
  может сразу попасть в auto-return; первоначально ожидавшийся, но уже ушедший
  Observer не резервируется.
- Same-timestamp replay является неизменяющим и draw-free; reverse time,
  malformed aggregate или обязательная конфигурация отклоняются до мутаций и
  randomness. Derived Lab/Portal Energy, creatures и risk не сохраняются.
- Needs Attention выбирает только OPEN Portal по цепочке: максимальный current
  risk score → UNSTABLE → меньший effective lifetime → более старый opened_at
  → меньший Portal ID. Возвращается исходный slice index; сортировки/мутации
  нет, numeric risk в tick result не публикуется.
- Добавлено 162 top-level Stage 8 теста в восьми `simulation*_test.go` файлах.
  Автоматический audit обязательных test names checkpoints A–H не нашёл
  пропусков.

### RED / GREEN history

| Checkpoint | RED evidence | GREEN evidence |
|---|---|---|
| A — state/scheduler invariants | `6c6f494` — отсутствовали simulation types/API/error | `bd9b40a` |
| B — Needs Attention | `2dd00dc` — отсутствовал selector | `e4c0f23` |
| C — pause/resume scheduler | `ea4a8ff` — отсутствовал Natural scheduler resolver | `4037a09` |
| D — due Natural spawn | `bb15047` — due branch не создавал Portal | `403ab4e` |
| E — Portal tick | `0326d80` — tick не разрешал Portal lifecycle/Collapse | `a27caf3` |
| F — Observer tick | `8891861` — tick не продвигал Observer phases | `695bc11` |
| G — Extraction tick | `461a01e` — sync оставался pending | `ea6e8c8` |
| H — full regressions | все 22 обязательных теста сразу GREEN characterization | `6d4df54` |
| Corrective — config preflight | `7be2f0e` — malformed required config проходил и мог расходовать random | `1e58efe` |

Checkpoint H намеренно не получил искусственный RED: полный порядок стадий,
cap, replay, large jumps, atomic failure и post-spawn Attention уже следовали
из checkpoints A–G. Финальный self-review затем нашёл отдельный реальный
config-preflight defect; он зафиксирован собственной RED/GREEN парой.

### Решения, ошибки и corrective passes

- Для checkpoint A `ResolveTick` был введён как validation/no-op shell, чтобы
  state invariants имели вызываемую public boundary; реальное поведение
  добавлялось checkpoints E–G.
- До RED-коммитов были исправлены только дефекты тестовых fixtures: конфликт
  helper names с тестами Stage 3 и два nil-pointer assertion path. В Git
  сохранены чистые behavioral RED, а не compile/panic noise.
- Финальный review обнаружил, что tick проверял spawn config, но не все
  Stage 2–7 диапазоны и подавлял ошибку Attention derivation. Добавлен
  `validSimulationConfig`; malformed Portal/Observer/Extraction/Emergency/Risk
  config теперь отклоняется атомарно до draw.
- Production randomness order закреплён тестами: Extraction transit draw
  предшествует Natural destination; затем без изменений идут Natural factory
  draws; fresh spawn delay рисуется последним. При 7-м OPEN Portal следующий
  delay не рисуется.

### Traceability и verification

- В requirements catalog добавлены `SPAWN-001..006`,
  `SIMULATION-001..004`, `ATTENTION-001..007` как прямые отображения Final
  Spec без изменения product semantics.
- `SPAWN-001..006`, `SIMULATION-002..004`, `ATTENTION-001..007`, `SLOT-006`
  и `PLANE-003` имеют GREEN. `SIMULATION-001` остаётся PARTIAL: domain step
  завершён, настоящий one-second ticker принадлежит Stage 11.
- `PLANE-004`, `PORTAL-001/002`, `SLOT-004` и `UI-007` остаются PARTIAL на
  явно описанных будущих границах. `WS-002`, `EVENT-001..005` и `PERSIST-*`
  остаются PLANNED.
- Финальный scope scan production `simulation*.go` не нашёл wall clock,
  ticker/sleep, HTTP/database/mutex/WebSocket, Event/ACTION_REJECTED или
  persistence dependencies. `internal/engine/manager.go` и
  `internal/domain/event.go` не менялись.
- После checkpoint H и corrective GREEN выполнены `gofmt -l .` (пусто),
  `go vet ./...`, `go build ./...`, `go test -count=1 ./...` и
  `go test -race -count=1 ./...`; все завершились с exit 0.

Stage 9 не начат.

## Corrective pass после сверки Blocks A/B (2026-09-04)

### Причина и граница

- После общего аудита Final Spec, реализованных Stages 0–8 и traceability
  подтверждены три интеграционных дефекта Stage 8: зависимость первого
  `Plane.ExploredAt` от Observer ID при позднем тике, отказ второй due
  Extraction-синхронизации из-за transit первой и неполный aggregate preflight.
- Исправления выполнены без Events, persistence, LabManager, ticker,
  transport, frontend и Recommendation algorithm. Публичные command-path
  semantics Stages 4/7 и порядок стадий Stage 8 сохранены.

### Что исправлено

- `resolveObserverStage` собирает для каждого изначально UNEXPLORED Plane
  самый ранний semantic deadline среди успешных возвратов текущего тика.
  Observer ID остаётся только детерминированным порядком обхода. Для уже
  EXPLORED Plane исходный `ExploredAt` не меняется.
- Simulation Extraction stage использует внутренний prepared resolver после
  единого aggregate preflight. Несколько due Portals последовательно заново
  выбирают текущего longest-waiting Observer и не считают transit,
  созданный предыдущим Portal этой же стадии, stale входом. Публичные
  `RecallObserver` и `ResolveExtractionSynchronization` сохраняют строгую
  прямую валидацию.
- `validateSimulationState` теперь проверяет согласованность Plane exploration,
  origin Natural scheduler, канонические Portal Slot/kind/status/stability/
  flow/termination, numeric baselines, lifecycle timestamps, Extraction marker,
  terminal semantic outcome, а также Observer timestamps, status fields,
  references и transit flow. Due Observer phase остаётся допустимой для
  catch-up.
- Hidden instability timestamp валиден только внутри полного окна:
  `[OpenedAt + InstabilityMinLifetime, ScheduledCloseAt - InstabilityCloseMargin]`.
- Старые synthetic fixtures приведены к уже существующим domain-инвариантам;
  invalid Extraction aggregate теперь отклоняется общим preflight как
  `ErrSimulationInvariant` до частичной обработки.

### RED / GREEN evidence

| Исправление | RED | GREEN |
|---|---|---|
| Самый ранний exploration timestamp | `4a192a4` | `5cb75b6` |
| Несколько late Extraction sync | `ac042db` | `048e00e` |
| Plane и scheduler invariants | `38d7561` | `e0cb9bb` |
| Portal aggregate invariants | `ab3a3b9` | `5519ea5` |
| Observer aggregate invariants | `0d27feb` | `68e8879` |
| Верхняя граница instability window | `702d8df` | `bb4f443` |

### Requirements, traceability и product gate

- `EXTRACTION-002` уточнён как глобальное наличие WAITING_RETURN Observer;
  новый `EXTRACTION-011` отдельно фиксирует обязательную eligibility выбранного
  Plane и имеет GREEN evidence.
- `PLANE-002` переведён из PLANNED в PARTIAL с domain evidence и честно
  отложенной Stage 10 persistence-границей.
- `PLANE-009`, `EXTRACTION-008`, `SIMULATION-002` и `SIMULATION-003` дополнены
  corrective regression mappings.
- Добавлены `RECOMMENDATION-001..004` со статусом PLANNED. Final Spec задаёт
  deterministic/no-LLM, закрытый enum, Portal Details visibility и
  informational-only semantics, но не задаёт decision table выбора. Реализация
  Recommendation в Stage 12/17 заблокирована до явного product amendment;
  алгоритм в corrective pass не придумывался.

### Итоговая verification

- `gofmt -l .` — пустой вывод.
- `go vet ./...` — exit 0.
- `go build ./...` — exit 0.
- `go test -count=1 ./...` — exit 0.
- `go test -race -count=1 ./...` — exit 0.
- Автоматическая проверка traceability: 377 уникальных test references,
  отсутствующих — 0.
- Diff относительно `9cd6db6` не меняет `internal/domain/event.go` и
  `internal/engine/manager.go`; scope scan не нашёл Stage 9, persistence,
  transport или ticker symbols в изменённых production-файлах.

Stage 9 не начат.

## Переход на блоковый delivery-режим (2026-09-04)

- Пользователь подтвердил ускоренный режим для оставшихся Stages 9–27.
- Единицей planning/execution становится roadmap block: C (9–14), D (15–21),
  E (22–27). Внутренние stage scope и RED/GREEN-коммиты сохраняются.
- Подтверждение между стадиями блока не требуется; после каждого блока агент
  выполняет полный verification/audit, останавливается и сверяет результат с
  пользователем.
- Разрешена параллельная работа независимых задач через изолированные
  worktrees с последовательной review/integration.
- Product semantics Final Spec, traceability и обязательные проверки не
  ослабляются. Recommendation decision table и deployment credentials остаются
  явными будущими gates.

Stage 9 всё ещё не начат: эта запись фиксирует только изменение процесса.

## Block C — product amendments перед planning (2026-09-04)

- Пользователь подтвердил точные event triggers для Stage 9: отдельное
  Extraction opening event; пара `OBSERVER_ARRIVED` + `RESEARCH_STARTED` при
  прибытии Observer именно в Plane назначения; start/restart/end семантика
  Leyline Override; одно Risk event на итоговую смену level; domain boundary
  для `ACTION_REJECTED`; единая хронологическая история без пагинации.
- Пользователь подтвердил Tutorial semantics для Stage 14: явные UI signals,
  Step 0 до первого тика, ожидание corridor на Step 2, автоматическое
  пересоздание timed scenario, retry после LOST, полный reset, idempotent start
  и continuity/gate при переходе в Live.
- Решения внесены в Final Spec §§26.1, 28.1 и §35 до составления execution-plan.
  Добавлены requirement IDs `EVENT-006..011`, `TUTORIAL-009..015`, `API-012`;
  они остаются PLANNED до соответствующей реализации Block C.
- Recommendation decision table не определена и не входит в эти amendments;
  её product gate для Block D остаётся без изменений.

Stage 9 implementation по-прежнему не начат.

### Block C execution-plan

- Создан `11_BLOCK_C_STAGE_09_14_TDD.md`: единый checkpoint-by-checkpoint план
  для Stages 9–14 с отдельными RED/GREEN commit boundaries.
- Зафиксированы package boundaries Events → SQLite → LabManager → REST →
  WebSocket → Tutorial, shared DTO, resolve-first transaction protocol,
  SQLite schema/restart contract и block-level verification.
- План явно не реализует Recommendation decision table и не начинает Stage 15.

До пользовательского утверждения execution-plan production implementation
Stage 9 не начинается.

## Recommendation Engine product decision (2026-09-04)

- Пользователь напомнил, что Recommendation algorithm был отложен до конкретной
  стадии, а не до frontend автоматически. Исправлено: backend decision table
  принадлежит Stage 12 Block C; Stage 17 только отображает результат.
- Согласован приоритет: безопасное перемещение Observer → предотвращение
  Collapse/Lab Energy loss → исследование UNEXPLORED Plane → исключение
  бесполезных действий с EXPLORED Plane.
- Алгоритм учитывает WAITING_RETURN/EXPLORING/active transit Observers, Portal
  flow, creatures, Risk, current Laboratory Energy и conservative time horizon.
  Hidden collapse timestamp и random не используются.
- Добавлены Final Spec §23.1 и requirements `RECOMMENDATION-005..012`.
  Все Recommendation rows остаются PLANNED до Stage 12 implementation.

Block C plan требует отдельной корректировки после утверждения written design;
Stage 9 production implementation не начат.

## Tutorial guidance/system design amendment (2026-09-04)

- Пользователь уточнил, что Step 0 не должен автоматически исчезать на первом
  tick и не должен перегружаться ценами или lifecycle rules. Он содержит лор,
  цель и карту интерфейса; explicit `TUTORIAL_INTRO_COMPLETED` начинает практику.
- Для каждого Step зафиксированы четыре независимые части: contextual guidance,
  действие игрока, system transition и completion condition.
- Stabilize exercise расширен с HIGH→MEDIUM до допустимого
  HIGH/CRITICAL→MEDIUM/LOW при обязательном реальном снижении Risk band.
- LOST retry использует normal SEND → research → RECALL другого Observer через
  сохраняемую Step 6 subphase; teleport состояния запрещён.
- Natural generator отключён во всём Tutorial, prepared Portals создаются по
  детерминированным свойствам, а не через случайный generator.
- Добавлены Final Spec §28.2 и requirements `TUTORIAL-016..019`; изменены
  `TUTORIAL-005`, `009`, `013`.

Production implementation Stage 9 всё ещё не начат.

### Block C plan synchronized with approved product designs

- `11_BLOCK_C_STAGE_09_14_TDD.md` обновлён после письменного утверждения
  Recommendation и Tutorial designs.
- Stage 12 теперь включает отдельный RED/GREEN checkpoint для полной
  Recommendation decision table; `API-002` больше не планируется PARTIAL из-за
  отсутствующего алгоритма. Только UI rendering row остаётся до Stage 17.
- Stage 14 разделён на persisted/prepared Steps 0–5, normal Step 6/7 retry и
  Tutorial API/reset/Live checkpoints. Добавлены exact signals, phases,
  prepared properties, player/system transitions и test names.
- Устаревший Recommendation gate удалён из block delivery design и roadmap.

Stage 9 production implementation не начат; изменение касается только
execution documentation.

## Stage 9 — Event system via TDD (2026-09-04)

### Реализованный scope

- `EventDraft`, closed `EventType` validation и persisted Event validation
  требуют JSON object payload, непустой message, positive entity/Event IDs и
  nonzero UTC timestamp.
- `PortalHistory` фильтрует общий Event source по точному `portal_id`,
  сортирует `created_at ASC, id ASC` и возвращает глубокие копии pointer fields.
- `EventsForStateTransition` создаёт deterministic drafts для Portal,
  Observer, Plane, Extraction, Leyline Override и Risk transitions. Late tick
  восстанавливает пересечённые observer phases по semantic deadlines; tie order
  соответствует Final Spec §26.1.
- `SimulationState.ResolveTick` возвращает `[]EventDraft` без persistence/ID.
  Same-`now` replay возвращает пустой stream; первый tick использует
  `Lab.EnergyBaseAt` как предыдущую boundary при отсутствии `LastTickAt`.
- `NewActionRejectedEvent` кодирует action и domain cause, но его обязательный
  вызов для всех manager-команд остаётся Stage 11.
- SQLite, `database/sql`, engine orchestration, HTTP и WebSocket не начаты.

### RED / GREEN evidence

| Checkpoint | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 9A model/validation/history | `d9ec172` | `go test -count=1 ./internal/domain -run 'Test(Event\|PortalHistory\|NewActionRejected)'`: отсутствуют `EventDraft`, `IsEventType` и новые APIs | `cf5371f` |
| 9B deterministic transition stream | `72dc1da` | focused suite: отсутствуют `EventsForStateTransition` и `SimulationTickResult.Events` | `c0a480a` |

Дополнительные pre-GREEN regressions подтвердили RED для portal-linked Override,
destination-linked transit loss и first-tick baseline; исправления вошли в
`c0a480a`.

### Requirements и verification

- `EVENT-002`, `EVENT-004..009`, `EVENT-011` — GREEN на Stage 9 domain boundary.
- `EVENT-001`, `EVENT-003`, `EVENT-010` — PARTIAL до Stage 11 command
  orchestration и Stage 12 transport boundary.
- `gofmt -l .` — пустой вывод.
- `go vet ./...` — exit 0.
- `go build ./...` — exit 0.
- `go test -count=1 ./...` — exit 0.

На момент этой записи Stage 9 был завершён, а реализация Stage 10 ещё не была
начата.

### Stage 9 corrective review pass

- Quality review выявил cadence-dependent `RESEARCH_COMPLETED`: late
  `OUTBOUND → WAITING_RETURN` сохранял исходный `portal_id`, а два обычных
  tick перехода `OUTBOUND → EXPLORING → WAITING_RETURN` создавали completion с
  `portal_id = nil`. Canonical результат теперь одинаков в обоих режимах:
  `RESEARCH_COMPLETED` относится к Observer/Plane и не содержит Portal ID.
- `eventsForPlanes` одинаково отклоняет duplicate и non-positive Plane IDs в
  обоих snapshots до генерации событий, поэтому malformed input не может
  создать duplicate `PLANE_EXPLORED`.
- `NewActionRejectedEvent` читает `cause.Error()` ровно один раз; message и
  payload гарантированно содержат один и тот же текст даже для stateful error.
- Прямыми tests закреплены semantic timestamp и Portal/Observer/Plane IDs для
  `PORTAL_STABILIZED` и `OBSERVER_DISPATCHED`.

Corrective TDD evidence:

| RED | Наблюдаемый RED | GREEN |
|---|---|---|
| `5e4f822` | late completion имел `portal_id=1`, split completion — `nil`; stateful cause читался дважды; malformed Plane snapshots принимались | `d2fe178` |

- Corrective focused/domain suites — exit 0.
- `go vet ./...`, `go build ./...`, `go test -count=1 ./...` — exit 0.
- `go test -race -count=1 ./...` — exit 0.

На момент corrective review Stage 9 реализация Stage 10 ещё не была начата.

## Stage 10 — SQLite persistence via TDD (2026-09-04)

### Реализованный scope

- Добавлены idempotent SQLite migration и canonical bootstrap: шесть требуемых
  product tables, служебная `schema_migrations`, constraints/indexes, embedded
  catalog из 85 Planes, 10 AVAILABLE Observers, Tutorial Step 0, Lab Energy 100
  и paused natural scheduler. Bootstrap не зависит от working directory и не
  перезаписывает уже существующее состояние.
- `Store.Commit` атомарно сохраняет полный Simulation/App snapshot и Event
  drafts в одной transaction, назначает Event IDs в хронологическом порядке и
  полностью откатывает state при ошибке Event insert.
- `Store.Load` восстанавливает сохранённые baselines/deadlines без скрытого
  lifecycle resolution; `Store.ListEvents` использует те же Event rows для
  global log и Portal History.
- Persistence codecs строго отклоняют corrupt enum/JSON/timestamps,
  round-trip сохраняет Unix epoch, optional extraction/override timestamps,
  scheduler и next Portal ID. Event entity IDs являются soft references, чтобы
  сохранять `ACTION_REJECTED` для отсутствующей запрошенной сущности.
- Review passes дополнительно закрепили структурную валидацию snapshot,
  non-finite Portal Energy rejection, consistent transactional reads,
  configurable structurally-valid timings, корректные shared-memory DSN и
  обязательные SQLite foreign-key/busy-timeout pragmas на каждом соединении.
- Stage 10 не запускает ticker, HTTP или WebSocket; manager ownership и
  meaningful-write orchestration остаются Stage 11.

### RED / GREEN evidence

| Checkpoint / corrective pass | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 10A migrations/bootstrap | `573716c` | отсутствовали `Store`, migration schema и canonical bootstrap | `5b17f10` |
| 10B atomic round-trip/restart | `27a6909` | отсутствовали `Commit`, `Load`, `ListEvents`, codecs и atomic rollback | `d8b1e6b` |
| soft Event references / invalid persisted data | `41adfd4` | FK не позволял missing requested entity IDs; invalid timestamps не отклонялись строго | `5220578` |
| Unix epoch timestamp | `a392a95` | zero Unix-nanos ошибочно трактовался как отсутствующее время | `c9430f6` |
| persistence invariants / consistent reads | `ac8249c` | malformed snapshots и non-finite Energy проходили; Load не гарантировал единый read snapshot | `f69829c` |
| configurable snapshots / memory DSN | `c885358` | валидные custom timings отвергались; memory URI semantics терялись | `85debf7` |
| adversarial SQLite pragma override | `89c6fb4` | URI мог переопределить обязательные connection pragmas | `bf6d132` |

Оба обязательных checkpoint прошли spec review. Последующие code-quality и
adversarial review passes дали пять отдельных RED/GREEN corrective pairs выше;
финальный review не оставил блокирующих замечаний для Stage 10 boundary.

### Requirements и verification

- `PERSIST-001`, `PERSIST-002` — GREEN.
- `PERSIST-003`, `PERSIST-004` — PARTIAL: Store-level atomic commit/recovery
  реализованы, но manager startup и правило meaningful writes относятся к
  Stage 11.
- `PLANE-004` — GREEN: canonical embedded bootstrap создаёт ровно 85
  неизученных Planes. `LAB-002` остаётся PARTIAL до полного Tutorial bootstrap
  Stage 14; Stage 10 лишь подтверждает persisted initial row Step 0/Energy 100.
- Focused persistence suites и review verification — exit 0.
- `gofmt -l .` — пустой вывод.
- `go vet ./...` — exit 0.
- `go build ./...` — exit 0.
- `go test -count=1 ./...` — exit 0.

Stage 10 завершён. Stage 11 на тот момент ещё не был начат.

## Stage 11 — LabManager / concurrency via TDD (2026-09-05)

### Реализованный scope

- Добавлен `LabManager` как единственный владелец активного snapshot:
  constructor загружает persisted state, все reads/ticks/commands сериализованы,
  а наружу возвращаются глубокие копии.
- Общий resolve-first transaction template выполняет полный catch-up до команды,
  атомарно сохраняет snapshot и Event drafts, публикует in-memory state только
  после успешного commit и сохраняет catch-up вместе с `ACTION_REJECTED` при
  domain rejection.
- Реализованы manager-команды `STABILIZE`, `CLOSE`, `SEND`, `RECALL` и
  `OPEN_EXTRACTION`, а также global Events и Portal History через единый
  repository source.
- `Run` потребляет supplied tick timestamps, `Updates` выдаёт неблокирующий
  coalescing signal для tick/action. Сериализация закрепляет exactly-one outcome,
  уникальные Portal IDs/slots и согласованные snapshots при конкурентном доступе.
- Persistence failure не меняет память и восстанавливает checkpoint random
  source, поэтому повтор команды воспроизводит те же state/events. Ожидающий
  ownership caller может отмениться через context.
- Stage 11 не добавляет HTTP/JSON/WebSocket transport: фактический WebSocket
  broadcast и проверка malformed HTTP остаются Stages 12–13.

### RED / GREEN evidence

| Checkpoint / corrective pass | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 11A ownership / resolve-first / atomic manager commit | `0707336` | отсутствовали manager repository boundary, constructor и command orchestration APIs | `3279a34` |
| 11B supplied ticks / signals / concurrency | `5aa4211` | `Tick` брал clock до manager lock и не обеспечивал требуемый serialized signal boundary | `5ff4388` |
| stale ticks / cancellation | `0305cbe` | stale tick завершал `Run` ошибкой, а отмена не прерывала уже выбранный tick, ожидающий ownership | `bdba6bf` |
| cancellable ownership / random rollback | `693320c` | ожидающие reads/commands не учитывали context; failed commit необратимо потреблял random draw | `cdc0b16` |
| random restore failure visibility | `647538b` | cancellation path мог скрыть ошибку восстановления random checkpoint | `38cf7f8` |

Checkpoint 11A и 11B прошли spec review. Code-quality review выявил три группы
ошибок: stale tick/cancellation semantics, неотменяемое ожидание manager lock и
неатомарное потребление random, затем маскировку random restore failure. Каждая
группа закреплена отдельной RED/GREEN парой; итоговый review одобрил Stage 11
без оставшихся замечаний.

### Requirements и verification

- `EVENT-001`, `EVENT-003`, `PERSIST-003`, `PERSIST-004`, `SIMULATION-001`,
  `PORTAL-001`, `PORTAL-002` — GREEN на реализованной manager boundary.
- `EVENT-010` — PARTIAL: manager отделяет и сохраняет domain rejection, но
  HTTP parsing/method/route boundary будет доказана Stage 12.
- `WS-002` — PARTIAL: coalescing update edge от supplied ticks/actions готов,
  authoritative DTO и WebSocket broadcast относятся к Stage 13.
- Focused engine suite, focused engine race suite и полный обычный suite —
  exit 0.
- `gofmt -l .` — пустой вывод.
- `go vet ./...` — exit 0.
- `go build ./...` — exit 0.
- `go test -count=1 ./...` — exit 0.

Stage 11 завершён. Stage 12 на тот момент ещё не был начат.

## Stage 12 — REST API and Recommendation via TDD (2026-09-05)

### Реализованный scope

- Добавлен общий public DTO layer для REST и будущего WebSocket: семь
  фиксированных slots, derived values на authoritative snapshot timestamp,
  Portal Details с history/risk/recommendation и без hidden domain fields.
- Реализованы `GET /api/state`, `GET /api/portals/{id}` и `GET /api/events`.
  Portal Details получает snapshot и history через единый manager boundary,
  поэтому не смешивает состояния разных моментов времени.
- Реализована полная deterministic Recommendation decision table Final Spec
  §23.1: безопасность Observer имеет приоритет, horizon comparisons строгие,
  hypothetical Stabilize работает на копиях, random/hidden collapse time не
  используются, а recommendation не ограничивает domain commands.
- Реализованы пять Stage 12 command routes, строгий JSON object decoder,
  confirmation retry, стабильные 404/409 domain responses и opaque 500 для
  infrastructure/invariant/composite failures.
- Malformed body, wrong method, unknown route и syntactically invalid Portal ID
  отсекаются до manager, поэтому не могут создать `ACTION_REJECTED`.
- Tutorial endpoints/signals не реализовывались; WebSocket не начинался.

### RED / GREEN evidence

| Checkpoint / corrective pass | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 12A REST reads / public DTO | `3574dc8` | отсутствовали `transport` DTO builders и HTTP router/read endpoints | `67ae898` |
| 12B Recommendation decision table | `1b19e91` | отсутствовали Recommendation type/function и Portal Details field | `a7fc674` |
| 12C commands / strict JSON / domain errors | `fabfb12` | отсутствовали command handlers/routes и HTTP error mapping | `f8e2c75` |
| coherent reads / exact command JSON | `2b3e231` | Details собирался из раздельных reads, DTO использовал отдельный clock, decoder принимал неканоничные bodies | `1afe935` |
| opaque composite errors / router validation | `8c3f5ef` | joined infrastructure errors ошибочно раскрывали domain mapping; router принимал nil manager/invalid config | `e389819` |
| single-cause joined domain mapping | `a8be2ff` | `errors.Join` с одной чистой domain cause ошибочно превращался в opaque 500 | `e4d5144` |

Дополнительный coverage-only commit `9931d1e` закрепил отсутствовавшее прямое
доказательство transport boundary: unknown route и invalid/overflow Portal IDs
не вызывают ни command, ни read manager APIs.

Все три плановых checkpoint прошли spec review. Последующий code-quality review
выявил incoherent read/time boundary, слишком мягкий JSON decoder, небезопасную
классификацию composite errors и отсутствие constructor validation. Исправления
прошли отдельными RED/GREEN-парами; финальный review не оставил замечаний для
Stage 12 boundary.

### Requirements и verification

- `API-001..008`, `API-010`, `API-011`, `EVENT-010`,
  `RECOMMENDATION-001`, `RECOMMENDATION-002`, `RECOMMENDATION-004..012` —
  GREEN.
- `API-009`, `API-012` — PLANNED до Tutorial Stage 14.
- `RECOMMENDATION-003` — PARTIAL: backend включает поле только в Portal Details,
  но фактическое frontend rendering относится к Stage 17.
- Focused domain/transport/httpapi suites и полный обычный suite — exit 0.
- `gofmt -l .` — пустой вывод.
- `go vet ./...` — exit 0.
- `go build ./...` — exit 0.
- `go test -count=1 ./...` — exit 0.

Stage 12 завершён. Stage 13 на тот момент ещё не был начат.

## Stage 13 — WebSocket realtime via TDD (2026-09-05)

### Реализованный scope

- Добавлен `/ws/lab` на `github.com/coder/websocket`: каждый новый клиент
  немедленно получает authoritative snapshot того же `transport.StateSnapshot`,
  который использует REST.
- Один Hub bridge потребляет `LabManager.Updates()` и после tick, успешной или
  отклонённой domain-команды публикует свежий snapshot всем клиентам. Reconnect
  всегда начинает с latest manager state.
- У каждого клиента capacity-one queue: новая версия заменяет непрочитанную,
  медленный клиент не блокирует manager/bridge/других клиентов. Sequence guard
  не допускает возврата к более старому snapshot.
- Concurrent connect/broadcast/disconnect, disconnect cleanup и отсутствие
  hidden realtime fields проверены под race detector.
- Router владеет WebSocket lifecycle через idempotent `Close`. Единственный
  bridge переживает zero-client handoff, а shutdown ждёт завершения bridge и
  всех уже допущенных handlers, включая заблокированный initial snapshot read.
- Frontend/UI не создавался; Stage 14 Tutorial не начинался.

### RED / GREEN evidence

| Checkpoint / corrective pass | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 13A initial snapshot / tick broadcast | `3a2b07f` | отсутствовали realtime Hub/handler, WebSocket dependency и initial/tick delivery | `7294706` |
| 13B actions / reconnect / router integration | `8166256` | `/ws/lab` в общем router отвечал 404 вместо WebSocket upgrade 101 | `a3f7167` |
| router lifecycle ownership | `452f711` | router не владел Hub shutdown: clients/bridge продолжали жить после HTTP lifecycle | `cd8b217` |
| bridge handoff / shutdown wait | `9472ce9` | остановка bridge при последнем disconnect могла потерять coalesced update; `Close` не ждал bridge exit | `1fec23f` |
| handler shutdown wait | `9d91fa2` | `Close` мог вернуться при ещё работающем handler, заблокированном на initial `State` | `99aa484` |

### Зафиксированное отклонение процесса

Capacity-one client queue, replacement/coalescing и неблокирующий publish были
реализованы уже в GREEN 13A (`7294706`), то есть до RED-коммита 13B. Поэтому
`8166256` непосредственно доказал RED router integration (`404 → 101`), а не
отсутствие уже существовавшего queue algorithm. История намеренно не
переписывалась: добавленные в 13B behavior tests и последующие race runs
проверяют slow-client isolation, coalescing, reconnect и concurrent lifecycle.

Оба checkpoint прошли spec review. Code-quality review выявил отсутствие явного
router ownership для Hub, race/lost-edge риск при остановке bridge на последнем
disconnect и неполное ожидание handlers при shutdown. Каждое замечание закрыто
отдельной RED/GREEN-парой; итоговый review не оставил замечаний для Stage 13.
Финальная root verification независимо повторила focused, full и race suites.

### Requirements и verification

- `WS-001`, `WS-002`, `WS-003` — GREEN.
- REST/WS используют общий DTO; `TestWebSocket_DoesNotExposeHiddenFields`
  подтверждает, что WebSocket не раскрывает domain-only timers/rates/scores.
- Focused realtime/httpapi suites и их race runs — exit 0.
- `gofmt -l .` — пустой вывод.
- `go vet ./...` — exit 0.
- `go build ./...` — exit 0.
- `go test -count=1 ./...` — exit 0.
- `go test -race -count=1 ./...` — exit 0.

Stage 13 завершён. Stage 14 не начат.

## Stage 14 — Tutorial engine и завершение Block C via TDD (2026-09-05)

### Реализованный scope

- Добавлен сохраняемый Tutorial context: step, phase, target Portal/Plane/
  Observer IDs и вычисляемый `expected_action`. SQLite migration 002,
  Store round-trip и restart сохраняют точный контекст без duplicate targets
  или Events.
- Реализована deterministic state machine Steps 0–9. Step 0 не меняется от
  tick; explicit intro/details/event-log signals продвигают только ожидаемый
  step и target. Natural generator полностью paused в Tutorial.
- Prepared Portals создаются по semantic profiles: безопасный Step 1 corridor,
  гарантированный HIGH/CRITICAL → MEDIUM/LOW Stabilize, безопасный CRITICAL
  natural-close target и clear return Portal с достаточным horizon.
- Step 6/7 использует обычные SEND → OUTBOUND → research → WAITING_RETURN →
  RECALL → RETURNING transitions. LOST запускает честный replacement flow с
  другим AVAILABLE Observer без teleport; terminal и direction-breaking
  scenarios получают fresh target ID с обычными Events.
- Реализованы строгие Tutorial REST endpoints, атомарный reset и gated Live
  handoff. Reset возвращает Energy 100, AVAILABLE roster и unexplored Planes,
  удаляя прежние Portals/Events. Live бесплатно завершает OPEN Tutorial
  Portals, атомарно разрешает активные transits, сохраняет Energy/history/
  exploration/Observer outcomes и запускает fresh Natural schedule с 0 OPEN.
- REST и WebSocket возвращают один `transport.StateSnapshot` с публичным
  Tutorial context и без hidden prepared values. Critical expected rejection
  одной transaction сохраняет progress, `ACTION_REJECTED`, replacement target
  и немедленный WebSocket update.
- Composition root теперь владеет Store, Manager ticker, REST/WebSocket Router
  и единым graceful shutdown; smoke test проходит SQLite bootstrap → REST →
  WebSocket. Независимые и joined service failures сохраняются без маскировки
  ожидаемым cancellation noise.
- Frontend и Stage 15 не начинались. Player-facing Tutorial copy/rendering
  остаётся Stage 20.

### RED / GREEN evidence

| Checkpoint / corrective pass | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 14A persisted context / Steps 0–5 | `5a95e37` | отсутствовали Tutorial types, migration context, prepared profiles и state-machine transitions | `3b3fc4f` |
| 14B research / return / LOST retry | `3d54ace` | отсутствовали Step 6 phases, safe same-Plane return target и normal replacement lifecycle | `e9e9c66` |
| 14C API / reset / Live continuity | `59325ab` | отсутствовали Tutorial endpoints, atomic reset, public context DTO и Live handoff | `54fd75a` |
| read-only progress / direct guided actions | `62761c6` | GET catch-up продвигал Tutorial state и смешивал lifecycle resolution с explicit progression | `ff381c1` |
| full restart journey / unified shutdown | `fbf4e69` | отсутствовали cold-start→restart→Live E2E и единое завершение HTTP/Manager/Hub/Store | `407889b` |
| irreversible retry / atomic Live transit resolution | `3238cb4` | wrong SEND мог soft-lock RECALL_READY; replacement event терялся; Live сохранял активный transit | `f086d0d` |
| lifecycle catch-up / independent service failures | `7901d39` | rejected lifecycle commands теряли due transitions; shutdown мог скрыть один из независимых failures | `9c5a3eb` |
| joined service failure filtering | `2d1c703` | expected cancellation внутри joined error маскировал неожиданный вложенный failure | `bfe68b2` |
| symmetric direction-breaking retry | `026769c` | успешный wrong RECALL во время `SEND_REPLACEMENT` фиксировал Portal как INBOUND и оставлял Tutorial без пригодного SEND target | `a569eb6` |

Все три плановых checkpoint прошли spec review. Первое итоговое review выявило
три Important расхождения: soft-lock после wrong SEND в `RECALL_READY`,
отсутствующий `PORTAL_OPENED` при раннем LOST и неатомарный Live handoff с
активным transit. Они закрыты парой `3238cb4` / `f086d0d`; следующее независимое
spec review одобрило именно этот corrective boundary. Последующие quality passes
закрепили lifecycle catch-up и сохранение joined infrastructure failures.
Финальный whole-Block-C review затем выявил ещё одно Important симметричное
расхождение: wrong RECALL в фазе `SEND_REPLACEMENT` мог навсегда зафиксировать
текущий target как INBOUND и оставить ожидаемый SEND без пригодного Portal.
Пара `026769c` / `a569eb6` добавила общий symmetric retry helper, fresh target,
restart evidence и normal Events. Повторное итоговое code-quality review после
этой пары дало APPROVED без оставшихся Critical/Important findings для Stage 14
и Block C boundary.

### Requirements и verification

- `TUTORIAL-001..015`, `TUTORIAL-017`, `TUTORIAL-018`, `API-009`, `API-012`,
  `LAB-002` — GREEN.
- `TUTORIAL-016` — PARTIAL: backend полностью задаёт context/action/system/
  completion contract; фактические Tutorial copy и rendering остаются Stage 20.
- `TUTORIAL-019` — PLANNED до Stage 20 UI.
- Focused domain/persistence/engine/httpapi/realtime/transport/server suites —
  exit 0.
- `gofmt -l .` — пустой вывод.
- `go vet ./...` — exit 0.
- `go build ./...` — exit 0.
- `go test -count=1 ./...` — exit 0.
- `go test -race -count=1 ./...` — exit 0.
- Автоматическая проверка test names из traceability — все перечисленные
  символы существуют.

Stage 14 и Block C завершены. Stage 15 не начат; выполнение остановлено перед
обязательной пользовательской сверкой блоков.

## Roadmap reconciliation после Block C (2026-09-05)

- Пользователь выбрал точечное обновление roadmap вместо минимальной правки или
  полного переписывания: сохранить документ кратким, но показать фактические
  статусы блоков, текущую planning boundary и актуальный block-delivery protocol.
- `02_IMPLEMENTATION_ROADMAP_TDD.md` теперь отмечает Blocks A, B и C как GREEN,
  Block D как следующий PLANNED scope, а Block E — как запрещённый до завершения
  и пользовательской сверки Block D.
- Устаревшее rolling-wave правило про ближайшие 1–2 стадии заменено на один
  detailed plan для ближайшего блока. Per-stage verification использует
  gofmt/vet/build/full ordinary tests; race detector обязателен на block boundary
  согласно текущему `AGENTS.md`.
- Дополнительно проверено ранее обсуждавшееся значение Natural spawn delay.
  Canonical Final Spec §7/§37, `docs/requirements.md`, Stage 8 plan, config и
  tests согласованно фиксируют inclusive `0..20 sec`. Поэтому roadmap уже был
  корректен в этой строке; прежнее предположение AI о необходимости заменить
  её на `1..20` было ошибочным и не применялось к документам или коду.
- Архитектурная граница Block D уточнена: frontend потребляет существующие
  публичные REST/WebSocket/Tutorial contracts и не изобретает gameplay
  semantics. Реальный обнаруженный backend defect допускает только отдельный
  доказанный corrective TDD pass.
- Устаревший roadmap-пункт `sync.RWMutex` для Stage 11 заменён фактической
  context-cancellable exclusive ownership boundary (`contextMutex`): manager
  сериализует resolve, random transaction, persistence и publication, а
  ожидающий caller может завершиться по своему context.

Roadmap-аудит не меняет product semantics, requirement statuses или код.
Block D implementation и Stage 15 всё ещё не начаты; следующий шаг — отдельный
detailed plan Stages 15–21 после пользовательской проверки roadmap.

## Block D — frontend design и detailed TDD plan (2026-09-05)

### Утверждённая UX-граница

- Подготовлен и пользовательски принят дизайн
  `docs/superpowers/specs/2026-09-05-block-d-frontend-design.md`: English UI в
  стилистике Mission Control, маршруты Dashboard / Portal Details / Event Log /
  AI Worklog и единый authoritative client store поверх REST + WebSocket.
- Dashboard всегда показывает все семь фиксированных Portal Slots без
  горизонтального или вертикального scrolling самих slots: desktop layout
  `4 + 3`, phone command-board layout `2 x 4` с центрированным Slot 7.
- В каждом slot постоянно видны четыре gameplay-команды: STABILIZE, CLOSE,
  SEND OBSERVER и RECALL OBSERVER. DETAILS остаётся навигацией. Недоступная
  команда сохраняет кликабельное объяснение причины, но не отправляет POST.
- Tutorial Step 5 является намеренным исключением: ожидаемый
  `ATTEMPT_CRITICAL_SEND` даёт кнопку `TRY SEND`, которая выполняет реальный
  отклоняемый запрос, создаёт `ACTION_REJECTED` и позволяет backend state
  machine продвинуть обучение без UI soft-lock.
- Portal visual использует локальный Plane artwork и Canvas 2D sparks по
  окружности: чистое изображение внутри, без внутренних колец и полос. Один
  общий animation scheduler, лимиты частиц, pause вне viewport и статичный
  `prefers-reduced-motion` сохраняют семь одновременно видимых Portals
  управляемыми по производительности.
- Extraction chooser использует локальные оптимизированные WebP assets,
  поиск и фильтры ALL / UNEXPLORED / EXPLORED / OBSERVER PRESENT. Runtime
  hotlink внешних изображений запрещён; manifest хранит attribution и fallback.
- Tutorial copy распределён по действиям и system effects соответствующих
  шагов: Step 0 даёт только lore/interface/lab orientation, а цены, risk,
  direction, collapse и Observer outcomes объясняются тогда, когда становятся
  practically relevant.

### Доказанные integration additions

- Для честного disabled-state UX Stage 15 добавляет в Portal DTO только
  вычисляемые `can_*` и `*_reason_code`; они переиспользуют существующие
  backend validators и не меняют gameplay semantics.
- Для фильтра OBSERVER PRESENT Stage 18 добавляет в Plane DTO агрегаты
  `observers_in_plane` и `observers_waiting_return`. Они вычисляются из
  authoritative Observer state и не раскрывают hidden simulation data.
- Portal recommendation остаётся backend-owned. Frontend отображает уже
  определённый Stage 12 contract и не вводит собственный decision algorithm.
- Изменения transport contract выполняются как отдельные RED/GREEN TDD
  checkpoints внутри соответствующей стадии и сопровождаются traceability.

### Planning evidence и текущий статус

- Design baseline зафиксирован commits `e0688e5`, `72fc16e`, `6076561`;
  два последних commits закрывают соответственно observer-presence transport
  gap и critical-send Tutorial progression.
- Создан корневой execution plan `12_BLOCK_D_STAGE_15_21_TDD.md`. Он описывает
  Stages 15–21 checkpoint-by-checkpoint: точные файлы, тестовые границы,
  наблюдаемый RED, minimal GREEN, stage verification, traceability/worklog и
  stage-level commits.
- Локальные визуальные прототипы `.superpowers/` добавлены в `.gitignore` и не
  являются production assets или частью Stage 15.
- Stage 15 implementation не начат. Block E / Stage 22 не начаты и остаются
  запрещены до полного Block D boundary verification и пользовательской
  сверки часов.

### Disk-conscious frontend tooling correction (2026-09-05)

- Пользователь запретил глобальные установки и попросил не расходовать без
  необходимости ограниченное место Windows-диска. Проверка окружения показала:
  Node `20.19.3`, npm `11.17.0`, Go `1.26.4`, Docker/Compose уже доступны;
  frontend `web/` и project-local npm dependencies ещё отсутствуют.
- В существующем Playwright cache уже находятся Chromium и headless shell
  revision `1208` плюс FFmpeg `1011`; они принадлежат установленному локально
  Playwright `1.58.2`. Поэтому Block D plan закрепляет `@playwright/test@1.58.2`
  вместо `1.63.0` и запрещает `playwright install`.
- npm install/ci выполняются только project-local, с временным cache в
  `/tmp/omenpath-npm-cache` и `PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1`. Docker не
  используется по умолчанию: перенос тех же dependencies в image/volume не
  уменьшает disk usage.
- После Block D нужно показать фактический размер `web/node_modules` и
  временного cache и предложить пользователю их удалить. Автоматическая
  очистка без подтверждения запрещена.
- При первом Stage 15 install npm доказал, что ошибочно выбранный в плане
  `@testing-library/jest-dom@6.10.0` требует Node 22 и сам помечен deprecated.
  До production implementation baseline исправлен на совместимый
  `@testing-library/jest-dom@6.9.1`; глобальные пакеты и browser cache не
  изменялись.

## Stage 15 — Frontend foundation via TDD (2026-09-05)

### Реализованный scope

- Создан воспроизводимый React 19 / TypeScript 6 / Vite 8 workspace с router,
  shared semantic shell, Vitest/RTL, ESLint/Prettier и production build.
- Реализованы typed REST client для всех MVP endpoints, bounded WebSocket
  reconnect `1/2/5/10 sec`, единый external snapshot store и React Provider.
  REST/WS используют один `acceptSnapshot`; старый или invalid `generated_at`
  не перезаписывает authoritative state. Command in-flight хранится по
  `portalID/action` key, без optimistic domain mutation.
- Backend quick actions расширены nullable unavailable reason fields. Pure
  `PortalActionAvailability` запускает ordinary confirmed command preflight на
  изолированной копии aggregate и локальном deterministic Random, поэтому не
  меняет state и не потребляет manager Random. HTTP conflicts и DTO reasons
  используют единый `transport.DescribeDomainError`.
- One-time Scryfall discovery создал 85 локальных уникальных WebP плюс generic
  fallback. 64 Plane получили matching art, 21 — deterministic generated
  fallback; весь runtime asset set занимает 3.4 MiB, крупнейший файл около
  92 KiB при лимите 180 KiB. Manifest содержит source/credit/policy/hash.
- Реализован Canvas 2D Portal effect: изображение остаётся чистым внутри,
  particles рождаются на окружности и движутся наружу по касательной; high/low
  caps `170/42`, один shared RAF scheduler, IntersectionObserver/document pause
  и reduced-motion static behavior. Добавлен Art Credits dialog.

### RED / GREEN evidence

| Checkpoint | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 15A workspace/routes | `7ed451f` | Vitest не мог разрешить отсутствующий `App`/router/pages | `fd647f7` |
| 15B REST/WS/store | `c07be33` | отсутствовали client, snapshot store и realtime modules | `e145ae2` |
| 15C action reasons | `c55f272` | отсутствовали availability API, DTO reason fields, shared descriptor и frontend copy | `1daf24a` |
| 15D local art/Canvas | `2950530` | отсутствовали manifest, resolver, particle/scheduler/Canvas и credits modules | `f2dfb3e` |

### Corrections и verification

- Baseline Go test сначала использовал пустой `/tmp` GOMODCACHE и закономерно
  упёрся в запрещённый network download; повтор с существующим module cache
  подтвердил code baseline. Local `httptest` ports требуют sandbox escalation.
- Typecheck Stage 15A доказал, что Vitest `test` config должен использовать
  `vitest/config`, а не plain `vite` type; исправлено одной import boundary.
- Два первых client GREEN failures оказались test-fixture defect: один
  `Response` нельзя читать несколько раз. Fixture стал возвращать новый
  Response на каждый запрос; production client не менялся.
- Availability preflight выявил два structurally invalid старых/new fixtures
  (OUTBOUND Observer при flow NONE и collapse позже close); исправлены только
  fixtures, не semantics.
- `npm --prefix web run format:check` — PASS.
- `npm --prefix web run lint` — PASS.
- `npm --prefix web run typecheck` — PASS.
- `npm --prefix web run test` — 10 files, 36 tests PASS.
- `npm --prefix web run build` — PASS.
- `gofmt -l .` — пустой output.
- `go vet ./...` — PASS.
- `go build ./...` — PASS.
- `go test -count=1 ./...` — PASS.
- UI-001..006 остаются aggregate PARTIAL: foundation доказан, но player-facing
  screens реализуются только Stages 16–21. Stage 16 ещё не начат; Block E не
  начат.

## Stage 16 — Dashboard via TDD (2026-09-05)

### Реализованный scope

- Dashboard строится только из authoritative snapshot: полный Lab summary,
  ровно семь Slots в `slot_index` order, Empty State, Leyline Override deadline
  и один Needs Attention link без перестановки карточек.
- Occupied и Empty Slot всегда показывают четыре gameplay-команды в одинаковом
  порядке. Недоступные команды остаются focusable, показывают безопасную
  причину и не выполняют POST; только реально выполняемая команда получает
  native `disabled`. Ответ команды принимается через общий snapshot store без
  optimistic domain mutation.
- Portal Slot показывает Energy с одним десятичным и оставшееся время `mm:ss`,
  но не показывает Risk, Recommendation, History и скрытые расчётные поля.
- Responsive board подтверждён настоящим Chromium: desktop использует 4+3,
  phone `390x760` — 2x4 с центрированным Slot 7; все семь Slots и 28 команд
  видны без прокрутки доски или страницы.

### RED / GREEN evidence

| Checkpoint | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 16A summary/Slots | `f7ded73` | отсутствовали Dashboard summary и фиксированные Portal/Empty Slot components | `2888deb` |
| 16B actions/layout | `f0cf92f` | отсутствовали shared four-command control и responsive browser contract | `8825d8b` |

Дополнительный contract-coverage commit `ae7e872` закрепил malformed Slot
rejection, snapshot-driven availability/attention update, occupied-to-empty
геометрию и явное разделение Vitest unit files от Playwright specs.

### Corrections и verification

- Первый phone E2E доказал, что сама Portal board помещалась, но vertical
  padding shell делал всю страницу выше viewport. Мобильная shell-компоновка
  уплотнена без скрытия Slots, summary или команд; повторный E2E прошёл.
- Первый full Vitest run обнаружил, что default discovery захватывал
  `web/e2e`; unit config теперь явно включает только `src/**/*.test.{ts,tsx}`.
- `npm --prefix web run format:check` — PASS.
- `npm --prefix web run lint` — PASS.
- `npm --prefix web run typecheck` — PASS.
- `npm --prefix web run build` — PASS.
- `npm --prefix web run test` — PASS.
- `npm --prefix web run test:e2e -- dashboard.spec.ts` — 2 PASS, 2
  intentionally skipped across mutually exclusive desktop/phone projects.
- `gofmt -l .` — пустой output.
- `go vet ./...` — PASS.
- `go build ./...` — PASS.
- `go test -count=1 ./...` — PASS.
- `UI-001`, `UI-002`, `UI-006`, `UI-007` и `SLOT-001` теперь GREEN;
  `UI-003..005` остаются PLANNED для Stages 17, 19 и 21. Stage 17 ещё не
  начат; Block E не начат.

## Stage 17 — Portal Details via TDD (2026-09-05)

### Реализованный scope

- `/portals/:id` валидирует positive integer до API call и различает loading,
  Portal Not Found, transient error с Retry и успешный Details view.
- Coalesced live resource допускает один GET in-flight и максимум один trailing
  refresh независимо от числа snapshot edges; dispose aborts запрос и удаляет
  trailing work. React Strict Mode не ломает resource lifecycle.
- Portal Details показывает все требуемые секции: Portal, Destination,
  Diagnostics с Risk/Recommendation и `How Risk Works`, History и те же четыре
  Actions, что Dashboard. Terminal diagnostics используют `Not applicable`;
  hidden score/decay/lifetime/timestamps не раскрываются.
- History сохраняет полученный от API chronological order; structured payload
  находится в закрытом disclosure и pretty-print выполняется только по запросу.
- Переход по matching Dashboard Details link несёт one-shot in-memory intent.
  Только он отправляет `PORTAL_DETAILS_OPENED`; direct route/GET, wrong target и
  wrong step signal не отправляют. Ответ signal принимается общим store.

### RED / GREEN evidence

| Checkpoint | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 17A live Details | `393e9c7` | отсутствовали live resource, required Details sections и route validation | `2af3dfe` |
| 17B History/Actions/signal | `3f09c06` | отсутствовали Event primitives, shared Details actions и navigation intent matcher | `9f75dfc` |

### Corrections и verification

- TypeScript не выводил mutation nullable-переменной внутри mock callback;
  signal fixture хранит захваченные AbortSignal в typed array, production code
  не менялся.
- Нативный закрытый `<details>` держит payload в DOM, но скрывает его визуально;
  тест исправлен с проверки отсутствия DOM-node на фактический closed/visibility
  contract.
- `npm --prefix web run format:check` — PASS.
- `npm --prefix web run lint` — PASS.
- `npm --prefix web run typecheck` — PASS.
- `npm --prefix web run build` — PASS.
- `npm --prefix web run test` — 16 files, 64 tests PASS.
- `npm --prefix web run test:e2e -- portal-details.spec.ts` — 4 PASS
  (desktop и phone, matching click и direct load).
- `gofmt -l .` — пустой output.
- `go vet ./...` — PASS.
- `go build ./...` — PASS.
- `go test -count=1 ./...` — PASS.
- `UI-003` и `RECOMMENDATION-003` теперь GREEN; frontend evidence добавлен к
  уже GREEN `EVENT-004` и `TUTORIAL-010`. Stage 18 ещё не начат; Block E не
  начат.

## Stage 18 — Extraction, confirmations and errors via TDD (2026-09-05)

### Реализованный scope

- State PlaneDTO additive расширен derived-полями `observers_in_plane` и
  `observers_waiting_return`. Один observer pass считает EXPLORING,
  WAITING_RETURN и RETURNING как присутствующих; waiting — точное подмножество.
  OUTBOUND/AVAILABLE/LOST и другой Plane не учитываются; неизвестный Plane ID —
  invariant error. Persistence schema не менялась.
- Shared shell открывает Extraction chooser независимо от предсказанной
  eligibility. Он показывает все 85 Plane с локальными изображениями,
  name/alias search, четыре mutually-exclusive фильтра, explored/in-plane/
  waiting badges, текущую Lab Energy и точную цену 30.
- Submit отправляет один positive `plane_id`; success принимает authoritative
  snapshot и закрывает dialog, а 404/409/network feedback сохраняет выбор.
  Focus удерживается в dialog, Escape отменяет без POST; длинный список
  прокручивается отдельно от всегда доступного footer.
- Общий FeedbackProvider гарантирует один confirm modal и ordered toast queue.
  Confirmable 409 повторяет точный command/id с `confirm=true`; cancel не
  отправляет второй запрос и возвращает focus. Non-confirmable failure очищает
  busy state. ConnectionState объявляет offline/reconnecting/connected и
  позволяет retry без удаления последнего snapshot.

### RED / GREEN evidence

| Checkpoint | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 18A Plane presence | `ca3ddb6` | PlaneDTO не содержал derived observer presence fields | `3543d54` |
| 18B Extraction chooser | `a27bc24` | отсутствовали filters, dialog и local-art chooser flow | `a2f9eb4` |
| 18C confirmations/errors | `40812bf` | отсутствовали confirm modal, feedback provider, connection retry и confirm orchestration | `3b1b72b` |

### Corrections и verification

- Focused backend run без escalation дошёл до WebSocket tests и упал только на
  sandbox запрете loopback listener; идентичный run с локальными sockets прошёл.
- Первый local-art assertion ожидал `/assets/planes/`, но канонический manifest
  использует `/planes/`; исправлен тест, не resolver.
- Первый Extraction E2E жёстко ожидал отсутствующее в seed имя `Agyrem`; тест
  стал выбирать первую authoritative карточку. Следующий E2E выявил реальный
  Grid overflow: список вытеснял footer. Явная `minmax(0,1fr)` строка оставляет
  прокрутку только списку, повторный desktop/phone run прошёл.
- `npm --prefix web run format:check` — PASS.
- `npm --prefix web run lint` — PASS.
- `npm --prefix web run typecheck` — PASS.
- `npm --prefix web run build` — PASS.
- `npm --prefix web run test` — 21 files, 78 tests PASS.
- Stage 18 browser suite — 8 PASS, 2 intentional project skips.
- `gofmt -l .` — пустой output.
- `go vet ./...` — PASS.
- `go build ./...` — PASS.
- `go test -count=1 ./...` — PASS.
- Extraction UI evidence добавлен к `EXTRACTION-*`/`API-008`, confirm/error — к
  `API-010/011`; `UI-001..003` остаются GREEN. Stage 19 ещё не начат; Block E
  не начат.

## Stage 19 — Global Event Log via TDD (2026-09-05)

### Реализованный scope

- `/events` использует общий EventList/EventRow и coalesced live resource для
  полного `GET /api/events`; полученный chronological order не меняется и
  pagination не добавляется. Payload закрыт по умолчанию, pretty-printed как
  text и не интерпретируется как HTML.
- Все 18 canonical `event_type` получили стабильные human labels. Можно выбрать
  один или несколько типов и точные positive Portal/Observer/Plane IDs;
  комбинация — logical AND, clear восстанавливает исходный порядок. Empty
  source отличается от empty filtered result.
- Accepted snapshot даёт не более одного trailing refresh. Route unmount aborts
  текущий GET; появившиеся events отображаются без сброса активных фильтров.
- Header navigation записывает one-shot Event Log intent. Только matching
  `OPEN_EVENT_LOG` отправляет `EVENT_LOG_OPENED`; direct load и wrong step не
  меняют Tutorial. Signal response принимается общим store.

### RED / GREEN evidence

| Checkpoint | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 19A log/filters | `78229d1` | отсутствовали filter model/components и live Global Event Log | `569e89f` |
| 19B realtime/signal | `1a57a6c` | Event Log не потреблял navigation intent и не отправлял matching signal | `ea87a2c` |

### Corrections и verification

- Abort test сначала завершал второй GET до unmount и поэтому не мог доказать
  abort; fixture оставляет trailing request in-flight и проверяет его signal.
- Browser locator `Action rejected` совпал с filter label, event type и message;
  assertion ограничен exact type label внутри одной `event-row`.
- `npm --prefix web run format:check` — PASS.
- `npm --prefix web run lint` — PASS.
- `npm --prefix web run typecheck` — PASS.
- `npm --prefix web run build` — PASS.
- `npm --prefix web run test` — 24 files, 88 tests PASS.
- `npm --prefix web run test:e2e -- events.spec.ts` — 2 PASS (desktop/phone).
- `gofmt -l .` — пустой output.
- `go vet ./...` — PASS.
- `go build ./...` — PASS.
- `go test -count=1 ./...` — PASS.
- `UI-004` теперь GREEN; frontend evidence добавлен к `EVENT-005`, `EVENT-011`
  и `TUTORIAL-010`. Stage 20 ещё не начат; Block E не начат.

## Stage 20 — Tutorial UI via TDD (2026-09-05)

### Реализованный scope

- Persistent non-blocking Tutorial panel показывает authoritative step/phase,
  target IDs, одно текущее действие и порционную English copy для всех шагов
  0–9. Step 0 ограничен lore/interface/lab overview; цены, риски, направления
  и lifecycle объясняются только на соответствующих шагах.
- Точный target Slot и требуемая команда подсвечиваются без сортировки карточек.
  SEND на CRITICAL force-enabled только для matching
  `ATTEMPT_CRITICAL_SEND` target и только при причине
  `PORTAL_CRITICAL_RISK`; другие step/portal/reason не получают исключения.
- BEGIN PRACTICE, TRY SEND, Reset и START LIVE используют обычные backend
  endpoints/signals и shared command locks. Frontend не двигает Tutorial
  локально. Ожидаемый critical 409 не превращается в общий error toast, а UI
  ждёт последующий authoritative WebSocket snapshot.
- Reset требует явного подтверждения последствий. Tutorial panel исчезает
  только после принятия snapshot с mode `LIVE`.
- Реальный Playwright journey прошёл Step 0→9→Live за 54.8 s без
  `waitForTimeout`: в конце 0/7 OPEN Portals, а Energy, Events, Observers,
  exploration и Plane state сохраняют continuity.

### RED / GREEN evidence

| Checkpoint | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 20A guidance/panel | `ff48bbd` | отсутствовали таблица contextual copy, persistent panel и target outline | `a5d1493` |
| 20B actions/journey | `81eed80` | отсутствовали CTA orchestration, exact critical override, reset/live handoff и browser journey | `e0e9e9a` |

### Corrections и verification

- Первый guidance run выявил несколько слишком неточных English формулировок
  и скрытый первый paragraph в collapsed disclosure; copy и видимая структура
  приведены к проверяемому смыслу плана.
- Double-click fixture сначала возвращал нереалистичный Step 0 snapshot после
  успешного intro. Он исправлен на реальный Step 1 response; command lock и
  authoritative expected action предотвращают повторный POST.
- Первый Go suite без специального cache упёрся в read-only системный cache, а
  sandbox-run — в запрет loopback `httptest`; повтор с cache в `/tmp` и
  разрешёнными local sockets прошёл без изменения кода.
- `npm --prefix web run format:check` — PASS.
- `npm --prefix web run lint` — PASS.
- `npm --prefix web run typecheck` — PASS.
- `npm --prefix web run test` — 27 files, 111 tests PASS.
- `npm --prefix web run build` — PASS.
- `npm --prefix web run test:e2e -- tutorial.spec.ts` — 1 desktop PASS in
  54.8 s, 1 intentional phone skip.
- `gofmt -l .` — пустой output.
- `go vet ./...` — PASS.
- `go build ./...` — PASS.
- `go test -count=1 ./...` — PASS.
- `TUTORIAL-016` и `TUTORIAL-019` теперь GREEN; frontend evidence добавлен к
  lifecycle/signal/reset/live rows. Stage 21 ещё не начат; Block E не начат.

## Stage 21 — AI Worklog UI и frontend integration via TDD (2026-09-05)

### Реализованный scope

- `/ai-worklog` импортирует корневой `01_AI_WORKLOG_CURRENT.md?raw` как
  единственный источник, рендерит Markdown/GFM через ReactMarkdown без
  `rehype-raw` и строит table of contents из исходных headings. Таблицы имеют
  локальный horizontal scroll; code blocks и длинный документ не ломают page.
- Vite plugin разрешает только этот точный root Worklog import и читает его во
  время build/dev transform; произвольный доступ frontend к filesystem не
  открыт. Уже закреплённые project-local `react-markdown` и `remark-gfm`
  использованы без новых установок.
- Shared shell приведён к согласованному Mission Control layout: стабильный
  desktop sidebar и phone bottom navigation. Browser back/forward сохраняет
  корректный route, а ConnectionState не меняет ширину content column.
- Мобильные Slot contents уплотнены без скрытия порталов или команд. Новый
  geometry test доказал, что последняя команда Slot 7 находится выше bottom
  navigation, а не просто скрыта `overflow`.
- Browser integration подтверждает локальные изображения и наличие artist,
  Source, Policy и fan-content notice. Runtime remote image requests нет.

### Участие разработчика и AI

- Разработчик выбрал English UI, Mission Control направление, семь видимых
  Slots на любом экране, локальный Plane artwork, Canvas sparks и обязательные
  объяснения причин недоступных backend-команд.
- AI формализовал frontend boundaries, подготовил TDD checkpoints, реализовал
  typed client/store/components и исправлял только доказанные тестами layout и
  orchestration defects. Gameplay semantics оставались backend-owned.
- Ключевые prompts/уточнения разработчика: отказаться от прокрутки семи
  порталов, сохранить четыре gameplay actions, показывать Plane image внутри
  портала, приблизить внешний вид к искрящемуся круглому portal reference и
  давать Tutorial информацию порционно.
- Общее число токенов и точное суммарное время разработки интерфейсом не
  предоставлены, поэтому числа не выдумывались. Измеряемые длительности
  отдельных verification runs зафиксированы рядом с соответствующими stages.

### RED / GREEN evidence

| Checkpoint | RED | Наблюдаемый RED | GREEN |
|---|---|---|---|
| 21A Worklog source/render | `8f92825` | route оставался placeholder без root Markdown, semantic GFM, TOC и HTML safety | `09ea924` |
| 21B shell/integration | `216addb` | navigation была top row: не desktop sidebar и перекрывала phone Slot 7 controls | `997b00b` |

### Corrections и verification

- Первое точечное `server.fs.allow` не разрешило raw root import в Vitest и
  одновременно было менее узким контрактом. Его заменил exact-source Vite
  plugin; focused test и production build прошли.
- Старый phone smoke-test проверял отсутствие scroll, но `overflow:hidden`
  маскировал команды ниже viewport. Новый coordinate assertion выявил реальный
  дефект примерно в 75 px; compact mobile Portal/Empty Slot content исправил
  видимость без удаления controls.
- Production build проходит; Vite сообщает advisory warning о единственном
  652 kB JS chunk. Это не ошибка Stage 21 contract и не исправлялось
  несогласованным Stage 22 performance scope.
- `npm --prefix web run format:check` — PASS.
- `npm --prefix web run lint` — PASS.
- `npm --prefix web run typecheck` — PASS.
- `npm --prefix web run test` — 28 files, 115 tests PASS.
- `npm --prefix web run build` — PASS.
- `npm --prefix web run test:e2e` — 15 PASS, 5 intentional desktop/phone
  project skips; full journey занимал около 1 minute, весь suite — 1.4 min.
- `gofmt -l .` — пустой output.
- `go vet ./...` — PASS.
- `go build ./...` — PASS.
- `go test -count=1 ./...` — PASS.
- `UI-005` теперь GREEN; `UI-001..007` подтверждены GREEN. Stage 21 завершён;
  Block E / Stage 22 не начат.
