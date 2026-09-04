# Omenpath Research Lab — Final Spec v5

> **Canonical source of truth.** Все stage plans, тесты и реализация должны соответствовать этому документу. Core-rule нельзя менять молча: изменение сначала фиксируется здесь, затем отражается в тестах/планах/Worklog.

## 1. Product goal

Магическая лаборатория отслеживает и контролирует межмировые Omenpaths, ведущие в 85 Planes Magic: The Gathering.

Пользователь:
- наблюдает за Portal Slots;
- управляет открытыми Portals;
- стабилизирует / закрывает их;
- отправляет и возвращает Observers;
- управляет Laboratory Energy;
- избегает аварийных Collapses;
- исследует все 85 Planes.

Victory condition: `85 / 85 Planes EXPLORED`.

## 2. Core entities

- `Plane`
- `Portal`
- `Observer`
- `LabState`
- `AppState`
- `Event`

## 3. Plane

Permanent entity.

Fields:
```text
id
name
aliases
catalog_tier
explored
explored_at
```

Rules:
- один Plane имеет 0..N Portal instances;
- одновременно разрешено несколько OPEN Portals в один Plane;
- exploration status принадлежит Plane, не Portal;
- Plane становится EXPLORED только после успешного возвращения Observer, который завершил research;
- повторный return в уже EXPLORED Plane не увеличивает progress повторно.

## 4. Portal

Portal — уникальный instance одного открытия.

Example:
```text
Omenpath #0042 → Innistrad
```

New connection to Innistrad later = new Portal.

Fields:
```text
id
name
slot_index
kind
destination_plane_id

energy_base
energy_base_at
energy_decay_rate

stability

opened_at
scheduled_close_at
instability_collapse_at

creatures_initial

observer_flow

status
termination_reason
closed_at

created_at
updated_at
```

Kinds:
```text
NATURAL
EXTRACTION
```

Statuses:
```text
OPEN
CLOSED
COLLAPSED
```

Termination:
```text
CLOSED:
- NATURAL_CLOSE
- MANUAL_CLOSE

COLLAPSED:
- ENERGY_DEPLETED
- INSTABILITY
```

CLOSED и COLLAPSED — разные outcomes.

## 5. Seven fixed Portal Slots

Dashboard всегда содержит 7 фиксированных Slots.

Rules:
- максимум 7 OPEN Portals;
- новый Portal занимает первый свободный Slot;
- Portal не меняет Slot в течение lifecycle;
- CLOSED/COLLAPSED освобождает Slot;
- Extraction использует обычный Slot;
- при 7/7 Natural Generator ждёт свободный Slot.

Dashboard slot показывает:
```text
Name
Destination (+ EXPLORED/UNEXPLORED)
Portal Energy
Stability
Time Remaining
Creatures Inside
Status
Quick Actions
```

Dashboard slot НЕ показывает:
```text
Risk
Recommendation
History
```

Эти данные находятся в Portal Details.

## 6. Needs Attention

Slots не сортируются.

Backend выбирает один priority Portal:
1. highest internal `risk_score`;
2. tie → UNSTABLE first;
3. tie → lower `effective_lifetime`;
4. tie → older `opened_at`.

Dashboard показывает:
```text
Needs Attention
Slot N
Omenpath #XXXX
Destination
```

Risk number не показывается.

## 7. Natural Portal Generator

Live starts with:
```text
0 OPEN Portals
```

Spawn:
```text
random delay 0..20 sec
```

После каждого spawn — новый `random(0..20 sec)`.

При 7/7 генерация приостанавливается. После освобождения Slot создаётся новый delay.

Destination:
```text
random among 85 Planes
```

Multiple active Portals to same Plane allowed.

## 8. Natural TTL / Natural Close

Initial balance:
```text
TTL 10..300 sec
```

Store:
```text
scheduled_close_at
```

At expiry:
```text
OPEN → CLOSED
termination_reason = NATURAL_CLOSE
```

UI label: `Time Remaining`.

В README объяснить, что исходное «время до схлопывания» разделено на штатное CLOSED и аварийное COLLAPSED.

## 9. Portal Energy

NATURAL:
```text
initial energy: 10.0..100.0%
decay rate: 0.1..1.0 %/sec
```

Decay rate hidden from user.

Realtime UI shows Energy with one decimal:
```text
73.8%
73.2%
...
```

Derived:
```text
current_energy =
max(0, energy_base - elapsed_seconds * energy_decay_rate)
```

Do not update SQLite every second.

If current energy reaches 0 before normal close:
```text
OPEN → COLLAPSED
ENERGY_DEPLETED
```

## 10. Stability

```text
STABLE
UNSTABLE
```

Initial balance:
```text
UNSTABLE_PROBABILITY = 35%
```

STABLE has no instability collapse timestamp.

UNSTABLE gets hidden:
```text
instability_collapse_at
```

Generation rule:
```text
random(
    opened_at + 5 sec,
    scheduled_close_at - 1 sec
)
```

So instability cannot collapse the Portal during its first 5 seconds, and hidden collapse is always scheduled at least 1 second before Natural Close.

User sees only `UNSTABLE`, never hidden timestamp.

If hidden time arrives while still OPEN+UNSTABLE:
```text
OPEN → COLLAPSED
INSTABILITY
```

## 11. Stabilize

Cost:
```text
20 Lab Energy
```

Effect:
```text
UNSTABLE → STABLE
instability_collapse_at = null
Portal Energy += 15
```

New current Energy becomes new baseline; decay rate stays unchanged.

Restrictions:
- Portal must be OPEN;
- must be UNSTABLE;
- current Portal Energy must be `<=85%`;
- Lab Energy >=20 unless Leyline Override active.

At exactly 85%, Stabilize allowed → 100%.

## 12. Creatures Inside

Required Portal field.

Natural initial count:
```text
0..10
```

Creature passage:
```text
2 sec each
```

Clearance safety margin:
```text
2 sec
```

Generation:
```text
max_creatures =
max(
  0,
  min(
    10,
    floor((TTL - 2 sec) / 2 sec)
  )
)

creatures_initial = random(0..max_creatures)
```

Example:
```text
TTL=10 sec → max creatures=4
```

Derived current count:
```text
creatures_inside =
max(
  0,
  creatures_initial - floor(seconds_since_open / 2)
)
```

Rules:
- `creatures_inside > 0` blocks SEND and RECALL;
- CLOSE remains possible only through confirmation;
- no creature death statistics.

## 13. Risk

Risk is derived, not persisted.

Backend has numeric `risk_score`; frontend gets only level.

Energy lifetime:
```text
energy_lifetime = current_energy / energy_decay_rate
```

Effective lifetime:
```text
effective_lifetime =
min(
  scheduled_remaining,
  energy_lifetime
)
```

Base risk:
```text
base_risk =
max(
  0,
  (45 - effective_lifetime) / 45 * 100
)
```

Instability:
```text
UNSTABLE +20
STABLE +0
```

Final:
```text
risk_score =
min(100, base_risk + instability_penalty)
```

Levels:
```text
0..25       LOW
>25..50     MEDIUM
>50..75     HIGH
>75..100    CRITICAL
```

Hidden instability timestamp NEVER affects risk.

CRITICAL blocks SEND and RECALL.

User does not see:
- numeric score;
- decay rate;
- energy_lifetime;
- hidden collapse timestamp.

Risk is defined only for OPEN Portals. CLOSED/COLLAPSED Portals have no current Risk Level; their Details show final status, termination reason and history instead.

Portal Details has `How Risk Works`; README contains full formula.

## 14. Laboratory Energy

Integer only:
```text
0..100
```

Tutorial starts at:
```text
100
```

Regeneration:
```text
+1/sec
```

Derived from baseline timestamp; no per-second DB update.

Costs:
```text
SEND OBSERVER     0
RECALL OBSERVER   0
CLOSE             5
STABILIZE        20
EXTRACTION       30
```

## 15. Observer

10 permanent Observer entities.

Current-state fields only:
```text
id
status
current_plane_id
active_portal_id
phase_started_at
phase_ends_at
created_at
updated_at
```

Past travel history lives in Event Log.

Statuses:
```text
AVAILABLE
OUTBOUND
EXPLORING
WAITING_RETURN
RETURNING
LOST
```

AVAILABLE = Laboratory (`current_plane_id=null`).

Lifecycle:
```text
AVAILABLE
→ OUTBOUND
→ EXPLORING
→ WAITING_RETURN
→ RETURNING
→ AVAILABLE
```

LOST is terminal.

Multiple Observers may exist in same Plane, including all 10.

Sending into already EXPLORED Plane is allowed without warning.

## 16. Observer Transit

Each transit duration generated once:
```text
random 5..15 sec
```

Only one Observer may be in transit through a Portal at a time.

If Portal becomes CLOSED or COLLAPSED before transit ends:
```text
Observer → LOST
```

This applies to OUTBOUND and RETURNING.

After successful OUTBOUND:
```text
OUTBOUND → EXPLORING
current_plane_id = destination
active_portal_id = null
```

Research:
```text
20 sec
```

Then:
```text
EXPLORING → WAITING_RETURN
```

Plane still UNEXPLORED.

After successful RETURNING:
```text
RETURNING → AVAILABLE
current_plane_id = null
active_portal_id = null
```

If research completed, Plane becomes EXPLORED at this moment.

## 17. Observer Flow per Portal

```text
NONE
OUTBOUND
INBOUND
```

NATURAL starts NONE.

First SEND:
```text
NONE → OUTBOUND
```

Then Portal permanently allows only `Lab → Plane`.

First RECALL:
```text
NONE → INBOUND
```

Then Portal permanently allows only `Plane → Lab`.

Direction does not reset after a transit.

## 18. SEND OBSERVER

Cost 0.

Allowed even:
- to EXPLORED Plane;
- to Plane already containing other Observers.

Restrictions:
- Portal OPEN;
- Risk != CRITICAL;
- Flow != INBOUND;
- creatures_inside == 0;
- no Observer currently in transit through Portal;
- at least one AVAILABLE Observer in Lab.

Warning only if Portal UNSTABLE:
```text
This Omenpath is unstable.
It may collapse unexpectedly during transit.
[Cancel] [Send Anyway]
```

No warning for low Energy / low remaining time / explored destination.

## 19. RECALL OBSERVER

Cost 0.

If multiple WAITING_RETURN Observers exist, choose longest-waiting one.

Restrictions:
- Portal OPEN;
- Risk != CRITICAL;
- Flow != OUTBOUND;
- creatures_inside == 0;
- Portal not busy with another transit;
- destination has WAITING_RETURN Observer.

Warning only if UNSTABLE.

## 20. Extraction Portal

Purpose: controlled return when random natural Portal is unavailable.

Availability:
- at least one WAITING_RETURN Observer somewhere;
- free Slot;
- Lab Energy >=30.

User selects Plane from grid:
- no waiting Observer → disabled/dimmed;
- waiting Observer exists → enabled.

Parameters:
```text
kind = EXTRACTION
stability = STABLE
observer_flow = INBOUND
energy = random 60..100
decay = random 0.1..1.0
TTL = random 30..60 sec
creatures = 0
```

After opening:
```text
5 sec synchronization
```

Then longest-waiting Observer automatically starts RETURNING.

Only first is automatic.

Further Observers require manual RECALL.

If Portal closes/collapses during actual RETURNING → that Observer LOST.
Observers still WAITING_RETURN remain in Plane.

## 21. Manual Close

Cost:
```text
5
```

During Leyline Override:
```text
0
```

Effect:
```text
OPEN → CLOSED
MANUAL_CLOSE
```

If creatures exist → confirmation.

If Observer is actively transiting → confirmation:
```text
Closing now will make Observer LOST.
```

After confirmation:
```text
Observer → LOST
Portal → CLOSED
```

## 22. Leyline Override

Every COLLAPSED:
```text
Lab Energy → 0
Leyline Override active for 20 sec
```

During Override:
```text
Close cost = 0
Stabilize cost = 0
```

All other restrictions remain.

Extraction stays 30.

Lab Energy continues +1/sec.

Another Collapse:
```text
Energy → 0
Override deadline reset to now+20 sec
```

## 23. Recommendation Engine

Deterministic; no runtime LLM.

Possible recommendations:
```text
LEAVE OPEN
WAIT FOR CORRIDOR
STABILIZE
CLOSE
SEND OBSERVER
RECALL OBSERVER
```

Recommendation shown only in Portal Details.

Recommendation is not a hard restriction.

### 23.1 Recommendation decision table

Цели в порядке приоритета:

1. безопасное перемещение и сохранение Observer;
2. предотвращение Collapse и обнуления Laboratory Energy;
3. исследование UNEXPLORED Plane;
4. отсутствие бесполезных действий с уже EXPLORED Plane.

Recommendation вычисляется только для OPEN Portal. Terminal Portal возвращает
`null`. Engine не использует hidden instability timestamp и не расходует
random. Recommendation не меняет доступность commands.

Безопасный новый SEND/RECALL требует, чтобы STABLE Portal имел
`effective_lifetime > ObserverTransitMax`. Для ожидания corridor требуется
`creature_clearance_remaining + ObserverTransitMax`. Для активного transit
используется известный `observer.phase_ends_at`; для будущего возврата
EXPLORING Observer — `research_remaining + ObserverTransitMax`.

Сравнение строгое: при точном совпадении deadline Portal lifecycle разрешается
раньше Observer lifecycle, поэтому Observer был бы LOST.

Stabilize считается подходящим только когда command допустим с текущей
Laboratory Energy (или бесплатен при Leyline Override), а состояние Portal
после детерминированного hypothetical Stabilize обеспечивает требуемый запас.
Hypothetical calculation не мутирует state.

Decision table применяется сверху вниз:

1. Extraction до завершения synchronization → `LEAVE OPEN`.
2. Observer уже в transit через этот Portal:
   - текущий путь безопасен → `LEAVE OPEN`;
   - Stabilize сделает его безопасным → `STABILIZE`;
   - иначе → `LEAVE OPEN`, потому что CLOSE гарантирует LOST.
3. В destination Plane есть WAITING_RETURN Observer и flow допускает INBOUND:
   - безопасный свободный corridor → `RECALL OBSERVER`;
   - creatures блокируют corridor, но после очистки запаса хватит →
     `WAIT FOR CORRIDOR`;
   - Stabilize сделает return безопасным → `STABILIZE`.
4. В destination Plane есть EXPLORING Observer и flow допускает будущий
   INBOUND:
   - Portal безопасно доживёт до research completion и return → `LEAVE OPEN`;
   - Stabilize создаст этот запас → `STABILIZE`.
5. Plane UNEXPLORED и в нём нет Observer и к нему не движется OUTBOUND
   Observer:
   - безопасный свободный corridor и есть AVAILABLE Observer →
     `SEND OBSERVER`;
   - creatures блокируют SEND, но после очистки запаса хватит →
     `WAIT FOR CORRIDOR`;
   - Stabilize сделает SEND безопасным → `STABILIZE`.
6. Plane EXPLORED и в нём нет Observer → `CLOSE`, если Close допустим, иначе
   `LEAVE OPEN`.
7. Если безопасная mission action невозможна и Risk HIGH/CRITICAL → `CLOSE`,
   если Close допустим, иначе `LEAVE OPEN`.
8. Во всех остальных случаях → `LEAVE OPEN`.

При `ObserverFlow = NONE` возврат WAITING_RETURN Observer имеет приоритет над
отправкой нового. Portal с OUTBOUND flow не сохраняется ради будущего RECALL.
Recommendation `CLOSE` может потребовать обычное UI confirmation; это не делает
её hard restriction или автоматическим действием.

## 24. Dashboard

Summary:
```text
Planes Explored       X / 85
Laboratory Energy     X / 100
Observers in Lab      X / 10
Observers in Worlds   X
Observers in Transit  X
Observers Lost        X
Active Omenpaths      X / 7
Critical Portals      X
Closed Portals        X
Collapsed Portals     X
```

Seven fixed Portal Slots.

Quick actions in Slot.

Risk/Recommendation/History only in Details.

Needs Attention points to one priority Portal.

## 25. Portal Details

Route:
```text
/portals/:id
```

Sections:

Portal:
```text
Name
Status
Energy
Stability
Time Remaining
Creatures
Observer Flow
```

Destination:
```text
Plane
EXPLORED/UNEXPLORED
Observers Exploring
Observers Waiting Return
Previous Connections
```

Diagnostics:
```text
Risk
Recommendation
How Risk Works
```

History:
Events filtered by `portal_id`.

Actions:
Same backend commands as Dashboard.

## 26. Event Log

Global page:
```text
/events
```

Event source shared with Portal History.

Events:
```text
PORTAL_OPENED
PORTAL_STABILIZED
PORTAL_CLOSED
PORTAL_COLLAPSED
RISK_LEVEL_CHANGED
OBSERVER_DISPATCHED
OBSERVER_ARRIVED
RESEARCH_STARTED
RESEARCH_COMPLETED
OBSERVER_RETURN_STARTED
OBSERVER_RETURNED
OBSERVER_LOST
PLANE_EXPLORED
EXTRACTION_PORTAL_OPENED
EXTRACTION_SYNCHRONIZED
LEYLINE_OVERRIDE_STARTED
LEYLINE_OVERRIDE_ENDED
ACTION_REJECTED
```

Event fields:
```text
id
event_type
portal_id
observer_id
plane_id
message
payload_json
created_at
```

Risk event only when level changes.

### 26.1 Семантика событий

- Открытие Extraction Portal создаёт только `EXTRACTION_PORTAL_OPENED`, без
  дополнительного `PORTAL_OPENED`.
- Когда Observer завершает исходящий переход через обычный Portal и прибывает
  в Plane назначения, один переход создаёт оба события: `OBSERVER_ARRIVED` и
  `RESEARCH_STARTED`. Возврат в лабораторию создаёт отдельное
  `OBSERVER_RETURNED`.
- Каждый успешный Collapse запускает Leyline Override и создаёт новый
  `LEYLINE_OVERRIDE_STARTED`, в том числе при перезапуске уже активного окна.
- `LEYLINE_OVERRIDE_ENDED` создаётся ровно один раз при первом разрешении
  состояния после текущего deadline.
- Если одно разрешение состояния перескочило несколько числовых границ Risk,
  создаётся одно `RISK_LEVEL_CHANGED` от предыдущего уровня к итоговому.
  Открытие и terminal close/collapse сами по себе это событие не создают.
- Каждый отклонённый domain command создаёт `ACTION_REJECTED`, включая ответ с
  требованием подтверждения. Некорректный transport input и неизвестный route
  domain-событий не создают.
- Global Event Log и Portal History возвращают общий источник событий в
  хронологическом порядке. В MVP возвращается вся подходящая история без
  пагинации.

## 27. Empty State

When 0 OPEN Portals:
- all 7 Slots remain visible;
- message indicates waiting for next anomaly.

Can happen:
- start Live Mode;
- after all Portals end;
- while waiting for next automatic Natural Portal.

## 28. Tutorial

Deterministic state machine.

Rule:
- Step advances only after expected completion condition.
- Wrong reversible action → same step.
- Wrong irreversible action destroying scenario → recreate equivalent Tutorial Portal and repeat step.

Tutorial Energy starts 100 and regenerates normally; not reset entering Live.

Tutorial state, Event Log, Observer state and Plane exploration persist into Live.

Before Live, Tutorial Portals are completed; Live starts 0/7.

Steps:

0. Empty State.
1. Open Portal Details; show Risk, Recommendation, History.
2. Creatures block SEND; wait corridor.
3. SEND Observer.
4. STABILIZE a prepared UNSTABLE Portal; must guarantee Risk HIGH→MEDIUM.
5. Prepared CRITICAL Portal; SEND must be rejected. Expected rejection completes step.
6. Observer completes research; new Portal to same Plane; RECALL.
7. Successful return → Plane EXPLORED.
8. Open Event Log.
9. Training Complete → Start Live.

### 28.1 Семантика исполнения Tutorial

- Step 0 виден до первого simulation tick. Первый тик создаёт подготовленный
  Tutorial Portal и переводит state machine на Step 1.
- Открытие Portal Details и Event Log передаётся явными UI-сигналами в
  `POST /api/tutorial/signal`; read-only GET никогда не меняют Tutorial state.
- Step 2 завершается ожиданием очистки corridor. Намеренно вызывать
  отклонённый SEND не требуется.
- Если подготовленный timed Tutorial Portal истёк или иначе разрушил активный
  сценарий, Tutorial автоматически создаёт эквивалентный Portal с новым ID и
  повторяет текущий step.
- Если Observer стал LOST во время упражнения на возврат, Tutorial возвращается
  к Step 6 и повторяет маршрут с другим доступным Observer.
- `POST /api/tutorial/start` идемпотентно запускает или продолжает Tutorial.
- `POST /api/tutorial/reset` полностью начинает Tutorial заново: Energy = 100,
  все Observers = AVAILABLE, все Planes = UNEXPLORED, прежние Tutorial Portals
  и Events удаляются.
- `POST /api/live/start` разрешён только после достижения Step 9. Оставшиеся
  Tutorial Portals закрываются системой без списания Energy. Текущие Energy,
  Event Log, Observer state и Plane exploration сохраняются; Live начинается
  без OPEN Portals.
- Истёкшие подготовленные timed Portals пересоздаются, поэтому корректность не
  зависит от выполнения шага в короткое wall-clock окно.

## 29. UI actions / errors

Dashboard is one React page containing 7 Slot components, not seven pages.

Actions available both on Dashboard and Portal Details.

Backend returns domain errors.

Normal errors → toast/inline.

Confirmation-required → modal, then retry same endpoint with:
```json
{"confirm": true}
```

Typical conflict status:
```text
409
```

## 30. Technical stack

Backend:
```text
Go
chi
WebSocket
SQLite
database/sql
```

Frontend:
```text
React
TypeScript
Vite
```

## 31. Backend architecture

```text
React
  ├─ REST commands ─────────────┐
  └─ WebSocket realtime ────────┤
                                ▼
                              Go API
                                │
                            LabManager
                    ┌───────────┼───────────┐
                    │           │           │
                   REST     Simulation    WS Hub
                    │           │           │
                    └───────────┼───────────┘
                                │
                           Active State
                                │
                              SQLite
```

## 32. Concurrency

Shared active state protected atomically (initially `sync.RWMutex`, potentially channels where useful).

Simulation and REST action can occur concurrently; exactly one valid transition wins.

No duplicated events.

Must pass:
```bash
go test -race ./...
```

## 33. Simulation Tick

1 sec tick.

Per tick:
1. current Lab Energy;
2. Portal Energy;
3. Creatures;
4. Natural Close;
5. Energy Collapse;
6. Instability Collapse;
7. Observer transit;
8. Research;
9. Extraction sync;
10. Risk level;
11. Event changes;
12. Natural Portal spawn timer;
13. Needs Attention;
14. state snapshot;
15. WebSocket broadcast.

## 34. Persistence

SQLite tables:
```text
planes
portals
observers
events
lab_state
app_state
```

Do not update derived realtime fields each second.

Persist meaningful transitions and baselines.

Observer table:
```text
id
status
current_plane_id
active_portal_id
phase_started_at
phase_ends_at
created_at
updated_at
```

Portal table:
```text
id
name
slot_index
kind
destination_plane_id
energy_base
energy_base_at
energy_decay_rate
stability
opened_at
scheduled_close_at
instability_collapse_at
creatures_initial
observer_flow
status
termination_reason
closed_at
created_at
updated_at
```

## 35. Transport

REST:
```text
GET  /api/state
GET  /api/portals/{id}
GET  /api/events

POST /api/portals/{id}/stabilize
POST /api/portals/{id}/close
POST /api/portals/{id}/send-observer
POST /api/portals/{id}/recall-observer
POST /api/extraction/open
POST /api/tutorial/start
POST /api/tutorial/reset
POST /api/tutorial/signal
POST /api/live/start
```

Тело Tutorial signal:
```json
{"signal": "PORTAL_DETAILS_OPENED"}
```

Допустимые значения: `PORTAL_DETAILS_OPENED` и `EVENT_LOG_OPENED`. Сигнал
продвигает Tutorial только при совпадении с текущим ожидаемым step и target.

WebSocket:
```text
/ws/lab
```

Backend broadcasts authoritative snapshot about once/sec and immediately after important actions.

## 36. AI Worklog

App route:
```text
/ai-worklog
```

Must contain:
- tools used;
- development time;
- token usage if available;
- stages;
- developer vs AI contribution;
- key prompts;
- developer decisions;
- AI mistakes/bad suggestions;
- manual rewrites;
- verification;
- future improvements.

## 37. Balance config

Centralized:

```text
MAX_ACTIVE_PORTALS = 7
SPAWN_DELAY_MIN = 0
SPAWN_DELAY_MAX = 20

NATURAL_TTL_MIN = 10
NATURAL_TTL_MAX = 300

PORTAL_ENERGY_MIN = 10
PORTAL_ENERGY_MAX = 100
PORTAL_DECAY_MIN = 0.1
PORTAL_DECAY_MAX = 1.0

UNSTABLE_PROBABILITY = 0.35
INSTABILITY_MIN_LIFETIME = 5
INSTABILITY_CLOSE_MARGIN = 1

CREATURE_MAX = 10
CREATURE_TRANSIT_SEC = 2
CREATURE_CLEARANCE_MARGIN = 2

OBSERVER_COUNT = 10
OBSERVER_TRANSIT_MIN = 5
OBSERVER_TRANSIT_MAX = 15
RESEARCH_DURATION = 20

LAB_ENERGY_MAX = 100
LAB_REGEN_PER_SEC = 1

CLOSE_COST = 5
STABILIZE_COST = 20
STABILIZE_BOOST = 15
STABILIZE_MAX_START_ENERGY = 85

EXTRACTION_COST = 30
EXTRACTION_ENERGY_MIN = 60
EXTRACTION_ENERGY_MAX = 100
EXTRACTION_TTL_MIN = 30
EXTRACTION_TTL_MAX = 60
EXTRACTION_SYNC_SEC = 5

EMERGENCY_DURATION = 20

RISK_SAFE_HORIZON = 45
RISK_INSTABILITY_PENALTY = 20
```

Balance values may be tuned after simulation; core domain semantics must not silently change.

## 38. Testing / QA

Backend/domain uses TDD.

Required major coverage:
- Portal lifecycle;
- Energy;
- Stability;
- Stabilize boundaries;
- creature timing;
- Risk boundaries;
- Slots;
- Observer lifecycle;
- Flow;
- transit failure;
- exploration timing;
- Laboratory Energy;
- Extraction;
- Leyline Override;
- Tutorial retry;
- persistence/restart;
- REST domain errors;
- WS snapshots;
- race/concurrency.

## 39. Security

No secrets/tokens in repo.

Provide:
```text
.gitignore
.env.example
```

## 40. Definition of Done

Evaluator can:
- understand app in ~1 minute;
- see Empty State and 7 fixed Slots;
- open Portal Details;
- see required portal fields;
- see Risk/Recommendation/History;
- act from Dashboard and Details;
- Stabilize and see Risk change;
- Close;
- Send;
- observe critical restriction;
- complete Tutorial;
- Recall;
- see LOST;
- see Natural Close and Collapse;
- use Extraction;
- see Leyline Override;
- see exploration progress;
- use Needs Attention;
- open Event Log and AI Worklog;
- run repository/tests from README;
- not hit a broken primary flow.

## 41. Scope guard

This is not a large game.

Out of scope:
- auth;
- roles;
- multiplayer;
- microservices;
- Redis/Kafka/Kubernetes;
- runtime LLM recommendations;
- Creature entities;
- combat/inventory;
- Observer management page;
- full MTG encyclopedia;
- complex 3D.

Priority: **small, coherent, explainable, tested, working system.**
