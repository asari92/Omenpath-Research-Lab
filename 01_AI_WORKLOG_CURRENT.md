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

- Реализация: Claude Code (агент в pi harness) по планам Stage 0–1.
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
- FakeClock/FakeRandom покрыты конкурентными тестами под `-race` заранее — это фундамент для LabManager (Stage 11).

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

- Tie energy-depletion == hidden-instability (оба раньше natural close) в плане не специфицирован; зафиксирован детерминированно: ENERGY_DEPLETED выигрывает (StrictBefore-семантика кандидатов, NATURAL_CLOSE выигрывает точные тай с обоими по Rules B/C). Из валидной фабрики такой тай вообще не возникает (hidden строго внутри окна).
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
