# AI Worklog — Current
## Stage: analysis & architecture

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
