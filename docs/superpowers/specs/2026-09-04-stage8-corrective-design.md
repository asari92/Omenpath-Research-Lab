# Corrective pass Stage 8 — дизайн

## Цель

Исправить три подтверждённых интеграционных дефекта Stage 8 до начала Stage 9,
не расширяя работу на Events, persistence, LabManager, transport или frontend.
Final Spec остаётся единственным источником продуктовой семантики.

## Граница работы

Corrective pass включает:

- корректный хронологический `Plane.ExploredAt`, когда один поздний тик
  разрешает несколько возвратов Observer;
- согласованную обработку нескольких просроченных Extraction-синхронизаций;
- полную обещанную Stage 8 валидацию canonical полей Portal и Observer;
- синхронизацию Stage 8 plan, requirements, traceability и worklog;
- явный учёт известного контракта Recommendation Engine и блокирующего
  product-definition gate.

В corrective pass не входят Stage 9, restart recovery, реальный секундный
ticker, concurrency, persistence, API, WebSocket, UI и алгоритм выбора
Recommendation.

## Выбранный подход

Используется точечная коррекция с сохранением публичных domain API и порядка
стадий тика из Final Spec. Общая хронологическая очередь событий не вводится:
это пересеклось бы со Stage 9 и изменило бы существенно больше завершённого
поведения, чем требуют найденные дефекты.

### Самый ранний timestamp исследования Plane

Обход Observers остаётся детерминированным и не меняет порядок хранимого slice.
Для каждого Plane, который был UNEXPLORED в начале Observer stage, собираются
все успешные завершения RETURNING, разрешённые текущим тиком. После обхода Plane
получает самый ранний семантический deadline возврата.

Если Plane уже был EXPLORED до начала тика, его исходный `ExploredAt`
сохраняется даже тогда, когда очередной разрешённый return содержит более
ранний timestamp. Это сохраняет действующий контракт идемпотентного повторного
возврата и одновременно убирает зависимость первого исследования от Observer
ID.

### Несколько поздних Extraction-синхронизаций

Строгая проверка stale transit остаётся частью публичного допуска SEND/RECALL.
Simulation tick уже выполняет полную aggregate preflight-проверку, поэтому
использует внутренний prepared-path Extraction synchronization. Этот путь не
повторяет command-time freshness validation после каждой предыдущей
синхронизации, изменившей рабочую копию roster.

Extraction Portals по-прежнему обходятся по возрастанию Portal ID. Каждый due
Portal заново выбирает текущего longest-waiting Observer. Observer, которого
уже выбрал предыдущий Portal, имеет статус RETURNING и не может быть выбран
повторно. Поэтому несколько due Portals получают разных кандидатов, пока они
существуют.

Observer stage остаётся перед Extraction stage. Следовательно, возврат,
запущенный поздней Extraction-синхронизацией, остаётся RETURNING до следующего
тика даже тогда, когда его семантический transit deadline уже не позже
текущего `now`. Следующий тик догоняет этот переход. Так сохраняется
утверждённый порядок стадий без второго Observer pass.

Публичный `ResolveExtractionSynchronization` сохраняет строгую атомарность и
валидацию прямых вызовов. Общая внутренняя подготовленная логика может быть
выделена, но command-path обязан сохранить все Stage 7 error identities и
порядок расходования random.

### Aggregate validation

`SimulationState.ResolveTick` отклоняет malformed state до мутации и
расходования random.

Plane и scheduler validation проверяют:

- положительные уникальные Plane ID;
- согласованность `Explored` и `ExploredAt`;
- `ExploredAt` не находится в будущем;
- активный scheduler имеет оба timestamp, `ScheduledAt <= DueAt` и
  `ScheduledAt <= now`;
- paused scheduler не содержит timestamp.

Portal validation проверяет:

- положительный ID, существующий destination и Slot в диапазоне `1..7` для
  OPEN и исторических записей;
- известные kind, status, stability, flow и termination enums;
- положительную длительность lifecycle и упорядоченные timestamps создания,
  открытия, energy baseline, update и terminal transition;
- energy baseline в `0..100`, положительный decay и creatures baseline в
  `0..cfg.CreatureMax`;
- STABLE не имеет hidden collapse timestamp, включая состояние после
  стабилизации; UNSTABLE имеет hidden timestamp внутри допустимого окна;
- NATURAL не имеет Extraction marker;
- EXTRACTION всегда STABLE, INBOUND и без creatures;
- завершённый Extraction marker равен `OpenedAt + ExtractionSync`;
- OPEN не имеет terminal fields, а terminal status, reason и `ClosedAt`
  согласованы с соответствующим lifecycle outcome.

Observer validation сохраняет due phase как допустимый вход и проверяет:

- положительный уникальный ID и разрешимые ссылки Plane/Portal;
- canonical набор optional fields для каждого статуса;
- `CreatedAt <= UpdatedAt <= now`;
- `PhaseStartedAt <= now`, а `PhaseEndsAt` строго позже старта, но может
  быть due;
- OUTBOUND и RETURNING ссылаются на Portal совместимого постоянного flow;
- Plane RETURNING Observer совпадает с destination активного Portal.

Terminal Portal может оставаться ссылкой незавершённого transit: ordered tick
должен принять такое состояние и перевести Observer в LOST.

## Согласованность документации

Stage 8 execution plan исправляется так, чтобы Observer-ID traversal больше не
определял exploration timestamp. В разделы invariants и tests добавляются новые
regression cases.

`docs/requirements.md` разделяет глобальную доступность Extraction и
eligibility выбранного Plane. Затронутые статусы и примечания traceability
приводятся к доказанному состоянию. `01_AI_WORKLOG_CURRENT.md` фиксирует
аудит, RED/GREEN commits, принятые решения и итоговую верификацию.

Известные контракты Recommendation добавляются со статусом PLANNED:

- deterministic, без runtime LLM;
- результат принадлежит закрытому enum из Final Spec;
- рекомендация информационная и не блокирует действия;
- показывается только в Portal Details.

Final Spec не содержит decision table выбора Recommendation. До составления
детального плана Stage 12 или Stage 17 продуктовая семантика должна быть
согласована и добавлена в Final Spec. Corrective pass не изобретает этот
алгоритм.

## TDD и структура commits

Отдельная RED/GREEN history сохраняется для:

1. самого раннего exploration timestamp;
2. нескольких поздних Extraction synchronization;
3. canonical aggregate validation.

Каждый RED commit содержит только компилирующиеся тесты и необходимое уточнение
ожидаемого правила. Каждый GREEN commit содержит минимальное production
изменение. После трёх циклов выполняются focused tests, полный test suite, race
detector, formatting, vet и build. Затем traceability/worklog обновляются
отдельным docs commit. Stage 9 не начинается.

## Критерии приёмки

- Поздний тик с двумя успешными возвратами в один ранее UNEXPLORED Plane
  записывает более ранний return deadline независимо от Observer ID и slice
  order.
- Уже EXPLORED Plane никогда не теряет первоначальный exploration timestamp
  из-за повторного возврата.
- Несколько due Extraction Portals не приводят к ошибке только потому, что
  предыдущий Portal в том же Extraction stage создал уже due transit.
- Каждый due Extraction Portal остаётся one-shot, а каждый автоматический
  return не более одного раза выбирает текущего waiting Observer.
- Каждый canonical invariant, обещанный Stage 8 plan, имеет rejection test;
  malformed input отклоняется атомарно и без random draw.
- Существующие Stage 0–8 tests и публичные domain error contracts остаются
  GREEN.
- Recommendation явно заблокирован на product definition вместо придуманной
  реализации.
- Stage 9 implementation не появляется.
