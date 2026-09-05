# Omenpath Research Lab — Final Spec v5

> **Canonical source of truth.** Все stage plans, тесты и реализация должны соответствовать этому документу. Core-rule нельзя менять молча: изменение сначала фиксируется здесь, затем отражается в тестах/планах/Worklog.
>
> Revision 2026-09-06: утверждены 20 Observers, anonymous multi-lab sessions и
> Block D UI/UX corrective contract. Filename сохраняется для стабильности
> существующих ссылок.

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
- `Laboratory`
- `Session`
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
Active Observer Transit (Observer, direction, remaining time), when present
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

20 permanent Observer entities.

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

Multiple Observers may exist in same Plane, including all 20.

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

If Portal becomes terminal exactly at the transit deadline, transit succeeds:
only a strictly earlier Portal close causes LOST.

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

Сравнение строгое: точное совпадение deadline имеет нулевой запас безопасности
и поэтому намеренно считается unsafe только для Recommendation. Это
консервативное правило подсказки: по §16 сам lifecycle при равенстве завершает
transit успешно, потому что LOST возникает лишь при более раннем закрытии
Portal.

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
Observers in Lab      X / 20
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

### 24.1 Dashboard presentation

Dashboard должен полностью помещаться в viewport без page/board scrolling.
Все 7 Slots имеют одинаковые размеры независимо от наличия Portal.

Desktop layout:
```text
4 Slots в верхнем ряду
3 Slots строго по центру нижнего ряда
```

Compact/mobile layout:
```text
2 + 2 + 2 + 1
```

Все 7 Slots и все 4 Quick Actions каждого occupied Slot остаются видимыми.
Controls пустого Slot присутствуют для сохранения геометрии, но являются
настоящими disabled controls и не обрабатывают pointer/keyboard activation.

UNSTABLE выделяет фон всей карточки заметным red danger treatment. Активный
Leyline Override меняет оформление всего Dashboard, а не только локальный
indicator.

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

### 25.1 Portal Details presentation

Portal Details полностью помещается в viewport без page scrolling:

- крупный Portal по центру;
- Actions непосредственно под Portal без отдельной framed section;
- Portal state/Energy/time/active Observer transit слева;
- Destination/research/creatures/Risk/Recommendation справа;
- компактная History-полоса снизу.

History обязательна. При раскрытии прокручивается только её внутренний список,
не вся страница.

Для CLOSED/COLLAPSED Portal destination image, Portal visual, glow и decoration
полностью обесцвечиваются. Background refresh не показывает постоянную
`Refreshing...` надпись.

Risk presentation:
```text
LOW       green
MEDIUM    yellow
HIGH      orange
CRITICAL  red
```

Recommendation дополнительно получает различимые цвет, icon и explanation для
safe / suggested / urgent / unavailable state.

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

UI показывает Event в порядке:
```text
HH:mm:ss-dd-MM-yyyy
Event title
Event details
```

Event type дополнительно различается цветом и icon. Event Log может
прокручивать внутренний список, так как история не ограничена высотой viewport.

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
4. STABILIZE a prepared UNSTABLE Portal; must guarantee Risk
   HIGH/CRITICAL → MEDIUM/LOW.
5. Prepared CRITICAL Portal; SEND must be rejected. Expected rejection completes step.
6. Observer completes research; new Portal to same Plane; RECALL.
7. Successful return → Plane EXPLORED.
8. Open Event Log.
9. Training Complete → Start Live.

### 28.1 Семантика исполнения Tutorial

- Step 0 остаётся активным сколько угодно и не завершается simulation tick.
  `TUTORIAL_INTRO_COMPLETED` после явного действия пользователя создаёт
  подготовленный Tutorial Portal и переводит state machine на Step 1.
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
- Natural Generator полностью отключён в Tutorial. Tutorial tick разрешает
  lifecycle существующих prepared entities, но не создаёт Natural Portals.
- Prepared Portals создаются детерминированно по требуемым свойствам сценария,
  а не через случайный Natural generator.
- LOST во время Step 7 не телепортирует replacement Observer в Plane. Step 6
  повторяет обычный путь через подфазы `SEND_REPLACEMENT`, `WAIT_RESEARCH` и
  `RECALL_READY`, сохраняя настоящие domain transitions и Events.

### 28.2 Tutorial guidance и системные переходы

Каждый step отдельно задаёт объяснение, действие игрока, system transition и
completion condition:

| Step | Объяснение | Действие игрока | Действие системы | Completion |
|---|---|---|---|---|
| 0 | Лор лаборатории, цель 85/85, Dashboard/Lab Summary/Observers/7 Slots/Needs Attention/Details/Event Log | Нажать «Начать практику» | До сигнала 0 OPEN и paused Natural Generator; после сигнала создать безопасный target Portal | `TUTORIAL_INTRO_COMPLETED` |
| 1 | Portal Energy отличается от Lab Energy; индивидуальный расход; Time не гарантирует запас Energy; Stability/Risk/Recommendation/History | Открыть target Portal Details | GET не мутирует state; explicit signal проверяет target ID | matching `PORTAL_DETAILS_OPENED` |
| 2 | Creatures блокируют corridor и выходят по одному каждые 2 sec | Ждать | Обычные ticks уменьшают derived Creatures; broken target пересоздаётся | `CreaturesInside == 0` |
| 3 | SEND стоит 0; нужен AVAILABLE; transit 5–15 sec; первое использование фиксирует OUTBOUND | SEND через target | Обычная command выбирает Observer; система сохраняет Observer/Plane и создаёт отдельный Step 4 target | Observer стал OUTBOUND |
| 4 | STABILIZE стоит 20; Lab Energy 0–100 и +1/sec; Portal Energy ≤85%; +15 Portal Energy; UNSTABLE → STABLE | STABILIZE target | Обычная debit/command/events; затем создать Step 5 CRITICAL target | STABLE и Risk HIGH/CRITICAL → MEDIUM/LOW |
| 5 | CRITICAL блокирует SEND/RECALL; CLOSE стоит 5; CLOSED ≠ COLLAPSED; Collapse обнуляет Lab Energy и запускает Override | Попытаться SEND | Обычный domain rejection + `ACTION_REJECTED`; Step 5 target настроен на безопасный NATURAL_CLOSE | `ErrPortalCriticalRisk` |
| 6 | Research 20 sec; RECALL стоит 0; longest-waiting; новый Portal фиксирует INBOUND | Ждать research, затем RECALL; при retry сначала SEND replacement | После research создать безопасный Portal к тому же Plane; retry использует normal outbound/research/inbound flow | Observer стал RETURNING |
| 7 | EXPLORED только после return; terminal Portal во время transit делает Observer LOST | Ждать | Обычный lifecycle делает AVAILABLE+EXPLORED либо LOST и возврат к Step 6 | AVAILABLE и Plane EXPLORED |
| 8 | Event Log и Portal History используют общий источник | Открыть Event Log | GET не мутирует; explicit signal проверяет текущий step | `EVENT_LOG_OPENED` |
| 9 | Natural Portals; Extraction стоит 30; sync 5 sec; первый return автоматический; переход в Live | Нажать «Начать Live» | Бесплатно закрыть оставшиеся OPEN Tutorial Portals, сохранить continuity, запустить Natural schedule | Mode LIVE |

Step 0 не содержит action prices или подробных Portal/Observer rules: правила
даются непосредственно перед соответствующим действием. UI copy реализуется в
Stage 20, но Stage 14 API возвращает authoritative `step`, `phase`, target IDs
и `expected_action` для правильного отображения.

Prepared scenario properties:

- Step 1 Portal: STABLE, creatures > 0, безопасный запас для clearance и SEND;
- Step 4 Portal: UNSTABLE, current Energy ≤85%, Risk HIGH/CRITICAL; обычный
  Stabilize гарантирует итоговый Risk MEDIUM/LOW;
- Step 5 Portal: STABLE CRITICAL из-за короткого scheduled lifetime, но Energy
  не заканчивается раньше NATURAL_CLOSE;
- Step 6 return Portal: STABLE, corridor clear, lifetime строго больше
  `ObserverTransitMax`.

Completed Portals продолжают обычный lifecycle. Бесплатная принудительная
очистка всех оставшихся OPEN Tutorial Portals выполняется только при переходе
в Live.

### 28.3 Tutorial presentation

Tutorial отображается как floating overlay поверх Dashboard и не меняет layout
остальной страницы. `More context` отсутствует: весь contextual text текущего
step виден сразу.

Выполнение ожидаемого action немедленно показывает следующий authoritative
step. Если между UI renders сервер успел завершить промежуточный step, UI
помещает его в presentation queue, показывает как completed ровно 7 sec и затем
автоматически догоняет текущий authoritative step. Это не откатывает gameplay
state.

`Back` открывает уже показанные steps. `Forward` возвращает к текущему step;
будущие невыполненные задания пропустить нельзя.

При пересоздании terminal Tutorial Portal backend semantics остаются
немедленными, но UI показывает exit animation старого Portal и entrance нового
с общей visual delay 2 sec.

## 29. UI actions / errors

Dashboard is one React page containing 7 Slot components, not seven pages.

Actions available both on Dashboard and Portal Details.

Backend returns domain errors.

Normal errors → toast/inline.

Любой command немедленно показывает pressed state, затем pending state и
блокирует повторную отправку до response. Success и failure имеют различимые
visual feedback и содержательное сообщение.

При потере realtime connection последний accepted snapshot остаётся видимым, а
commands временно блокируются. Connection labels:
```text
Planar link stable
Planar paths unstable
Disconnected from the planes
```

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

### 30.1 Frontend visual contract

Все routes используют одну цельную dark-fantasy magical laboratory surface.
Левая information/navigation zone не имеет отдельного background, тяжёлого
vertical separator или самостоятельной application frame. Одна decorative
frame окружает весь viewport; ley-line, stone, bronze, leather и glass motifs
продолжаются через всю страницу. Branding и proprietary UI assets других игр
не копируются.

Portal entrance/terminal transitions, UNSTABLE и Override имеют motion. При
`prefers-reduced-motion` смысл состояния сохраняется через opacity, icon, text
и color без длительного движения.

Все 85 Planes имеют локальное изображение без card text/frame. Сначала
используется проверенный landscape/art crop; при его отсутствии создаётся
оригинальное изображение по каноническому описанию Plane. Runtime не зависит от
remote artwork URLs; source/artist/policy metadata поддерживается для каждого
asset.

Navigation order:
```text
Dashboard
Event Log
Help
Open Extraction
Artwork Credits
AI Worklog
```

## 31. Backend architecture

```text
React
  ├─ REST commands ─────────────┐
  └─ WebSocket realtime ────────┤
                                ▼
                              Go API
                                │
                       LabManager Registry
                    ┌───────────┼───────────┐
                    │           │           │
                   REST     Simulation    WS Hub
                    │           │           │
                    └───────────┼───────────┘
                                │
                      Active State by lab_id
                                │
                              SQLite
```

Каждая laboratory имеет отдельный `LabManager` serialization boundary и update
stream, keyed by `lab_id`. REST и WebSocket одного клиента разрешают один и тот
же `lab_id`; cross-lab reads, mutations и broadcasts запрещены.

## 32. Concurrency

Active state каждой laboratory защищён atomically собственным LabManager.
Manager registry отдельно синхронизирует создание, lookup и cleanup managers.

Simulation and REST action can occur concurrently; exactly one valid transition wins.

No duplicated events.

Must pass:
```bash
go test -race ./...
```

## 33. Simulation Tick

1 sec tick.

Per active laboratory tick:
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
labs
sessions
planes
portals
observers
events
lab_state
app_state
```

Все gameplay tables содержат обязательный `lab_id`. Singleton state становится
singleton-per-lab. Entity/foreign keys включают `lab_id`; каждый persistence
read/write явно scoped по resolved laboratory.

Session rows хранят только hash opaque token, `lab_id`, expiry и last activity.
Raw session token не сохраняется. Expired laboratory удаляется каскадно после
проверки, что session не была продлена конкурентным request.

Migration переносит существующий singleton state в специальную legacy
laboratory. Первый request без cookie атомарно присоединяет новую session к
ещё не занятой legacy laboratory; только последующие посетители получают новый
canonical bootstrap. Так upgrade не делает существующий progress недоступным.

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
{"signal": "PORTAL_DETAILS_OPENED", "portal_id": 42}
```

Допустимые значения: `TUTORIAL_INTRO_COMPLETED`, `PORTAL_DETAILS_OPENED` и
`EVENT_LOG_OPENED`. Сигнал продвигает Tutorial только при совпадении с текущим
ожидаемым step и target. `portal_id` обязателен только для
`PORTAL_DETAILS_OPENED` и должен совпадать с текущим Tutorial target; для двух
остальных сигналов он отсутствует.

WebSocket:
```text
/ws/lab
```

Backend broadcasts authoritative snapshot about once/sec and immediately after important actions.

### 35.1 Anonymous laboratory session

Первый request без действительной session cookie атомарно создаёт новую
laboratory, canonical bootstrap state и анонимную session.

Session contract:
```text
sliding expiry             30 days since last activity
cookie                     HttpOnly; SameSite=Lax
production cookie          Secure; HTTPS required
token entropy              at least 256 bits
database                   token hash only
```

Закрытие browser не завершает session. Один browser profile и его вкладки
используют одну laboratory. Другой browser/profile/device получает отдельную
laboratory. Expired/cleared cookie создаёт новую игру; перенос progress между
devices без общей session не поддерживается.

Activity означает успешно разрешённый REST request либо WebSocket handshake.
Уже connected WebSocket не позволяет cleanup удалить laboratory до disconnect,
но не выполняет database write на каждый tick.

Все REST routes и `/ws/lab` проходят одну session resolution boundary. В
WebSocket client регистрируется только в Hub соответствующего `lab_id`.

Authoritative state/Details DTO показывают active Observer transit как минимум
через `observer_id`, `portal_id`, direction/phase, `started_at` и
`completes_at`. Remaining time вычисляется из этих timestamps и resolved
snapshot time.

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

Navigation label `AI Worklog` является последним item и ведёт на `/ai-worklog`.

### 36.1 Help

App route:
```text
/help
```

Help содержит:
- Multiverse lore и цель Laboratory;
- карту интерфейса;
- Lab Energy и Portal Energy;
- Portal lifecycle, Stability, Risk и terminal outcomes;
- Observers, transit, research, return и LOST;
- Creatures и corridor;
- SEND / RECALL / CLOSE / STABILIZE;
- Extraction;
- Leyline Override;
- Recommendations;
- Tutorial recap и glossary.

Обязательный UI остаётся English. Переключение English/Russian относится к
optional post-MVP scope и реализуется только при оставшемся времени.

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

OBSERVER_COUNT = 20
OBSERVER_TRANSIT_MIN = 5
OBSERVER_TRANSIT_MAX = 15
RESEARCH_DURATION = 20

SESSION_TTL = 30 days
SESSION_REFRESH_INTERVAL = 12 hours
SESSION_CLEANUP_INTERVAL = 1 hour
SESSION_TOKEN_BYTES = 32
LAB_ID_BYTES = 16
SESSION_COOKIE_NAME = omenpath_session

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
- anonymous session creation/renewal/expiry;
- cross-lab REST/WebSocket/persistence isolation;
- expired-lab cleanup races;
- fixed no-scroll Dashboard and Portal Details layouts;
- Tutorial presentation queue and action feedback;
- local artwork coverage for all 85 Planes;
- race/concurrency.

## 39. Security

No secrets/tokens in repo.

Anonymous session token:
- генерируется cryptographically secure RNG;
- передаётся только cookie;
- хранится в SQLite только как hash;
- не принимается через path/query/body;
- не логируется;
- production cookie требует HTTPS и `Secure`.

Каждый tenant-owned persistence query обязан включать `lab_id`. Cross-lab
object lookup возвращает обычный not found и не раскрывает существование entity
в другой laboratory.

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
- open Help and understand lore/mechanics;
- see all 7 equal Slots without Dashboard scrolling;
- use Portal Details without page scrolling;
- see Observer transit direction and remaining time;
- return after browser restart and continue the same unexpired laboratory;
- open a different browser profile and receive an isolated laboratory;
- run repository/tests from README;
- not hit a broken primary flow.

## 41. Scope guard

This is not a large game.

Out of scope:
- registered username/password/social auth;
- roles;
- shared/cooperative multiplayer laboratory;
- progress transfer between devices;
- horizontal multi-instance scaling;
- mandatory English/Russian localization;
- microservices;
- Redis/Kafka/Kubernetes;
- runtime LLM recommendations;
- Creature entities;
- combat/inventory;
- Observer management page;
- full MTG encyclopedia;
- complex 3D.

Priority: **small, coherent, explainable, tested, working system.**
