# Omenpath Research Lab — Requirement Catalog

> **Stage 0 deliverable.** Source of truth: `00_FINAL_SPEC_v5.md`.
> Правило: нельзя менять тест под код, если requirement не изменился. Изменение требования — сначала здесь, потом в тестах/планах/Worklog.

## Legend

| Type | Meaning |
|---|---|
| **INV** | Invariant — не может нарушаться |
| **BEH** | Behavior — переход/действие/поведение во времени |
| **BAL** | Balance Config — тюнимый параметр (живёт в `internal/config`, Final Spec §37) |
| **UI** | UI Contract — shape/visibility authoritative state для фронтенда |

Statuses отслеживаются в [`docs/traceability.md`](traceability.md): `PLANNED → RED → GREEN → REFACTORED`; `PARTIAL` — правило покрыто частично
(чистый helper / подмножество поведения), полная оркестрация — на будущей стадии.

Балансовые значения (BAL) централизованы в `internal/config/config.go` (`Config.Default()`); тест
`internal/config/config_test.go` сверяет их с Final Spec §37 (исполняемая часть Stage 0).

---

## PLANE

| ID | Type | Rule | Spec |
|---|---|---|---|
| PLANE-001 | INV | Plane — постоянная сущность, независимая от Portal | §3 |
| PLANE-002 | INV | Один Plane имеет 0..N Portal instances | §3 |
| PLANE-003 | BEH | Одновременно разрешено несколько OPEN Portals в один Plane | §3, §7 |
| PLANE-004 | BEH | Seed 85 Planes; все по умолчанию UNEXPLORED (`explored=false`) | §3, data seed |
| PLANE-005 | INV | SEND не делает Plane EXPLORED | §15, §16 |
| PLANE-006 | INV | Завершение OUTBOUND (arrival) не делает Plane EXPLORED | §16 |
| PLANE-007 | INV | Завершение research не делает Plane EXPLORED | §16 |
| PLANE-008 | BEH | Plane становится EXPLORED только после успешного возвращения Observer с завершённым research | §3, §16 |
| PLANE-009 | INV | Повторный return в уже EXPLORED Plane не увеличивает progress повторно | §3 |

## PORTAL

| ID | Type | Rule | Spec |
|---|---|---|---|
| PORTAL-001 | INV | Portal — уникальный instance одного открытия; новое соединение с Plane = новый Portal | §4 |
| PORTAL-002 | BEH | Имена последовательные: `Omenpath #XXXX` | §4 |
| PORTAL-003 | BEH | Kinds: NATURAL / EXTRACTION | §4 |
| PORTAL-004 | INV | Statuses: OPEN / CLOSED / COLLAPSED; CLOSED и COLLAPSED — разные outcomes | §4 |
| PORTAL-005 | BEH | At TTL expiry: OPEN → CLOSED, `termination_reason = NATURAL_CLOSE` | §8 |
| PORTAL-006 | BEH | Manual Close: OPEN → CLOSED, `MANUAL_CLOSE` (cost 5 — см. LAB-007) | §21 |
| PORTAL-007 | BEH | Energy достигает 0 до normal close: OPEN → COLLAPSED, `ENERGY_DEPLETED` | §9 |
| PORTAL-008 | BEH | Hidden instability time при OPEN+UNSTABLE: → COLLAPSED, `INSTABILITY` | §10 |
| PORTAL-009 | INV | Terminal (CLOSED/COLLAPSED) Portal не может снова стать OPEN | §4 |

## SLOT

| ID | Type | Rule | Spec |
|---|---|---|---|
| SLOT-001 | INV | Dashboard всегда содержит ровно 7 фиксированных Slots (MAX_ACTIVE_PORTALS=7) | §5, §37 |
| SLOT-002 | BEH | OPEN Portal занимает один Slot | §5 |
| SLOT-003 | BEH | Новый Portal занимает первый свободный Slot | §5 |
| SLOT-004 | INV | Portal не меняет Slot в течение lifecycle | §5 |
| SLOT-005 | BEH | CLOSED/COLLAPSED освобождает Slot | §5 |
| SLOT-006 | BEH | При 7/7 OPEN Natural Generator ждёт свободный Slot | §5, §7 |
| SLOT-007 | BEH | Extraction использует обычный Slot | §5 |

## ENERGY

| ID | Type | Rule | Spec |
|---|---|---|---|
| ENERGY-001 | BAL | Initial natural energy: 10.0..100.0% | §9, §37 |
| ENERGY-002 | BAL | Decay rate: 0.1..1.0 %/sec, скрыт от пользователя | §9, §37 |
| ENERGY-003 | BEH | `current_energy = max(0, energy_base − elapsed × decay)`; derived, без записи в SQLite каждую секунду | §9, §34 |
| ENERGY-004 | INV | Current energy clamp ≥ 0 | §9 |
| ENERGY-005 | BEH | Energy достигает 0 до normal close → COLLAPSED / ENERGY_DEPLETED | §9 |
| ENERGY-006 | BEH | Stabilize: Portal Energy += 15 (к current) | §11 |
| ENERGY-007 | BEH | Stabilize: новый current Energy = новый baseline (`energy_base_at = now`); decay не меняется | §11 |
| ENERGY-008 | INV | Decay rate не меняется после Stabilize | §11 |
| ENERGY-009 | BEH | Current energy > 85% → Stabilize запрещён | §11 |
| ENERGY-010 | BEH | Ровно 85% → Stabilize разрешён → 100% | §11 |

## STABILITY

| ID | Type | Rule | Spec |
|---|---|---|---|
| STABILITY-001 | INV | Только STABLE / UNSTABLE | §10 |
| STABILITY-002 | INV | STABLE не имеет `instability_collapse_at` | §10 |
| STABILITY-003 | BEH | UNSTABLE получает hidden `instability_collapse_at` при создании: `random(opened_at+5s, scheduled_close_at−1s)` | §10, §37 |
| STABILITY-004 | UI | Hidden timestamp никогда не показывается пользователю и не влияет на Risk | §10, §13 |
| STABILITY-005 | BEH | Stabilize: UNSTABLE → STABLE | §11 |
| STABILITY-006 | BEH | Stabilize: `instability_collapse_at = null` | §11 |
| STABILITY-007 | BEH | STABLE нельзя стабилизировать повторно | §11 |

## CREATURE

| ID | Type | Rule | Spec |
|---|---|---|---|
| CREATURE-001 | BAL | Natural initial count: 0..10 | §12, §37 |
| CREATURE-002 | BAL | Проход одного существа: 2 sec | §12, §37 |
| CREATURE-003 | BAL | Clearance safety margin: 2 sec | §12, §37 |
| CREATURE-004 | BEH | `max_creatures = max(0, min(10, floor((TTL−2)/2)))` | §12 |
| CREATURE-005 | BEH | TTL=10 sec → max creatures = 4 | §12 |
| CREATURE-006 | BEH | `creatures_inside = max(0, creatures_initial − floor(seconds_since_open/2))` | §12 |
| CREATURE-007 | BEH | `creatures_inside > 0` блокирует SEND и RECALL | §12, §18, §19 |
| CREATURE-008 | UI | Close при creatures > 0 возможен только через confirmation | §12, §21 |

## RISK

| ID | Type | Rule | Spec |
|---|---|---|---|
| RISK-001 | BEH | `energy_lifetime = current_energy / decay` | §13 |
| RISK-002 | BEH | `effective_lifetime = min(scheduled_remaining, energy_lifetime)` | §13 |
| RISK-003 | BEH | `base_risk = max(0, (45 − effective_lifetime)/45 × 100)` (RISK_SAFE_HORIZON=45) | §13, §37 |
| RISK-004 | BEH | UNSTABLE +20 (RISK_INSTABILITY_PENALTY) | §13, §37 |
| RISK-005 | INV | `risk_score = min(100, base + penalty)` | §13 |
| RISK-006 | BEH | 0..25 → LOW | §13 |
| RISK-007 | BEH | >25..50 → MEDIUM | §13 |
| RISK-008 | BEH | >50..75 → HIGH | §13 |
| RISK-009 | BEH | >75..100 → CRITICAL | §13 |
| RISK-010 | INV | Hidden instability timestamp НЕ влияет на Risk | §13 |
| RISK-011 | UI | Фронтенд получает только level, не numeric score; score/decay/hidden timestamp не показываются | §13 |
| RISK-012 | BEH | CRITICAL блокирует SEND | §13, §18 |
| RISK-013 | BEH | CRITICAL блокирует RECALL | §13, §19 |

Risk определён только для OPEN Portals; для CLOSED/COLLAPSED текущего Risk Level нет (см. Stage 2 plan §18).

## LAB

| ID | Type | Rule | Spec |
|---|---|---|---|
| LAB-001 | INV | Laboratory Energy: integer 0..100 | §14 |
| LAB-002 | BEH | Tutorial стартует с 100 | §14, §28 |
| LAB-003 | BEH | Регенерация +1/sec, derived от baseline timestamp (без per-second DB update) | §14, §34 |
| LAB-004 | BEH | Cap 100 | §14 |
| LAB-005 | BAL | SEND OBSERVER = 0 | §14, §37 |
| LAB-006 | BAL | RECALL OBSERVER = 0 | §14, §37 |
| LAB-007 | BAL | CLOSE = 5 | §14, §37 |
| LAB-008 | BAL | STABILIZE = 20 | §14, §37 |
| LAB-009 | BAL | EXTRACTION = 30 | §14, §37 |
| LAB-010 | BEH | Недостаточно Lab Energy → отказ paid action (кроме случаев Leyline Override для Close/Stabilize — EMERGENCY-003) | §14 |

## OBSERVER

| ID | Type | Rule | Spec |
|---|---|---|---|
| OBSERVER-001 | INV | Ровно 20 permanent Observer entities | §15, §37 |
| OBSERVER-002 | BEH | Initial status AVAILABLE | §15 |
| OBSERVER-003 | INV | AVAILABLE = Laboratory (`current_plane_id = null`) | §15 |
| OBSERVER-004 | BEH | Lifecycle: AVAILABLE → OUTBOUND → EXPLORING → WAITING_RETURN → RETURNING → AVAILABLE | §15 |
| OBSERVER-005 | INV | LOST — terminal | §15 |
| OBSERVER-006 | BAL | Transit duration: random 5..15 sec | §16, §37 |
| OBSERVER-007 | BEH | Длительность транзита генерируется один раз при старте | §16 |
| OBSERVER-008 | BEH | Успешный OUTBOUND → EXPLORING; `current_plane_id = destination`, `active_portal_id = null` | §16 |
| OBSERVER-009 | BAL | Research: 20 sec | §16, §37 |
| OBSERVER-010 | BEH | Research completion → WAITING_RETURN (Plane ещё UNEXPLORED) | §16 |
| OBSERVER-011 | BEH | Успешный RETURNING → AVAILABLE, поля обнуляются | §16 |
| OBSERVER-012 | BEH | Portal стал CLOSED во время transit (OUTBOUND/RETURNING) → Observer LOST | §16 |
| OBSERVER-013 | BEH | Portal стал COLLAPSED во время transit → Observer LOST | §16 |
| OBSERVER-014 | BEH | Несколько Observers в одном Plane разрешены (включая все 20) | §15 |
| OBSERVER-015 | BEH | SEND в уже EXPLORED Plane разрешён без warning | §15, §18 |
| OBSERVER-016 | BEH | RECALL при нескольких WAITING_RETURN выбирает longest-waiting | §19 |

## FLOW

| ID | Type | Rule | Spec |
|---|---|---|---|
| FLOW-001 | BEH | NATURAL портал стартует с observer_flow = NONE | §17 |
| FLOW-002 | BEH | Первый SEND: NONE → OUTBOUND; далее портал навсегда только Lab → Plane | §17 |
| FLOW-003 | BEH | Первый RECALL: NONE → INBOUND; далее навсегда только Plane → Lab | §17 |
| FLOW-004 | BEH | OUTBOUND отклоняет RECALL | §18, §19 |
| FLOW-005 | BEH | INBOUND отклоняет SEND | §18 |
| FLOW-006 | INV | Только один Observer в transit через Portal одновременно | §16, §17 |
| FLOW-007 | BEH | Следующий transit разрешён после окончания предыдущего; направление не сбрасывается | §17 |

## EXTRACTION

| ID | Type | Rule | Spec |
|---|---|---|---|
| EXTRACTION-001 | BAL | Cost 30 | §20, §37 |
| EXTRACTION-002 | BEH | Требуется хотя бы один WAITING_RETURN Observer где-либо | §20 |
| EXTRACTION-003 | BEH | Требуется свободный Slot | §20 |
| EXTRACTION-004 | BEH | Параметры: kind=EXTRACTION, stability=STABLE, flow=INBOUND, creatures=0 | §20 |
| EXTRACTION-005 | BAL | Energy: random 60..100 | §20, §37 |
| EXTRACTION-006 | BAL | TTL: random 30..60 sec | §20, §37 |
| EXTRACTION-007 | BEH | После открытия: 5 sec synchronization | §20, §37 |
| EXTRACTION-008 | BEH | После sync longest-waiting Observer автоматически начинает RETURNING | §20 |
| EXTRACTION-009 | BEH | Только первый возврат автоматический | §20 |
| EXTRACTION-010 | BEH | Дальнейшие Observers — только manual RECALL | §20 |
| EXTRACTION-011 | INV | Выбранный Plane должен содержать хотя бы одного WAITING_RETURN Observer; иначе Plane недоступен для открытия Extraction Portal | §20 |

## EMERGENCY

| ID | Type | Rule | Spec |
|---|---|---|---|
| EMERGENCY-001 | BEH | Каждый COLLAPSED: Lab Energy → 0 | §22 |
| EMERGENCY-002 | BEH | Leyline Override активен 20 sec | §22, §37 |
| EMERGENCY-003 | BEH | Во время Override Close cost = 0 и Stabilize cost = 0; остальные ограничения сохраняются | §22 |
| EMERGENCY-004 | INV | Extraction cost во время Override остаётся 30 | §22 |
| EMERGENCY-005 | BEH | Lab Energy продолжает +1/sec во время Override | §22 |
| EMERGENCY-006 | BEH | Новый Collapse: Energy → 0 и deadline сбрасывается на now+20 sec | §22 |

## SPAWN

| ID | Type | Rule | Spec |
|---|---|---|---|
| SPAWN-001 | BAL | Natural spawn delay is inclusive random 0..20 sec | §7, §37 |
| SPAWN-002 | BEH | Every successful spawn creates a fresh next delay unless capacity becomes full | §7 |
| SPAWN-003 | BEH | At 7/7 generation pauses; after a Slot frees, a fresh delay starts | §5, §7 |
| SPAWN-004 | BEH | Natural destination is random among the canonical 85 Planes | §3, §7 |
| SPAWN-005 | BEH | Multiple OPEN Portals may target the same Plane | §3, §7 |
| SPAWN-006 | BEH | At most one Natural spawn per tick; delay 0 becomes eligible on a later tick | §7, §33 |

## SIMULATION

| ID | Type | Rule | Spec |
|---|---|---|---|
| SIMULATION-001 | BEH | One-second simulation has a supplied-time deterministic domain step | §31, §33 |
| SIMULATION-002 | BEH | Tick transition order follows the Stage 8 subset of §33 | §33 |
| SIMULATION-003 | INV | Tick is atomic, monotonic and idempotent at the same timestamp | §32, §33 |
| SIMULATION-004 | INV | Realtime Lab/Portal Energy, creatures and risk remain derived | §9, §12, §13, §34 |

## ATTENTION

| ID | Type | Rule | Spec |
|---|---|---|---|
| ATTENTION-001 | BEH | Highest internal risk_score wins | §6 |
| ATTENTION-002 | BEH | Exact risk tie prefers UNSTABLE | §6 |
| ATTENTION-003 | BEH | Remaining tie prefers lower effective_lifetime | §6 |
| ATTENTION-004 | BEH | Remaining tie prefers older opened_at | §6 |
| ATTENTION-005 | INV | Only OPEN Portals participate | §6, §13 |
| ATTENTION-006 | INV | Complete technical tie uses lower Portal ID | §6, Stage 8 design S8-D8 |
| ATTENTION-007 | INV | Selection does not sort or mutate stored Portals | §5, §6 |

## RECOMMENDATION

| ID | Type | Rule | Spec |
|---|---|---|---|
| RECOMMENDATION-001 | INV | Recommendation Engine детерминирован; runtime LLM не используется | §23 |
| RECOMMENDATION-002 | INV | Допустим только закрытый enum: LEAVE OPEN, WAIT FOR CORRIDOR, STABILIZE, CLOSE, SEND OBSERVER, RECALL OBSERVER | §23 |
| RECOMMENDATION-003 | UI | Recommendation показывается только в Portal Details | §23–25 |
| RECOMMENDATION-004 | INV | Recommendation является информационной подсказкой и не вводит hard restriction | §23 |
| RECOMMENDATION-005 | INV | Terminal Portal не имеет Recommendation; engine не использует hidden collapse timestamp и random | §23.1 |
| RECOMMENDATION-006 | BEH | Безопасность движения определяется консервативным временным горизонтом transit/corridor/research | §23.1 |
| RECOMMENDATION-007 | BEH | Активный transit нельзя рекомендовать прервать CLOSE; Stabilize предлагается только если делает путь безопасным | §23.1 |
| RECOMMENDATION-008 | BEH | WAITING_RETURN имеет приоритет: RECALL/WAIT/STABILIZE по безопасности и direction | §23.1 |
| RECOMMENDATION-009 | BEH | EXPLORING Observer учитывается для сохранения пригодного INBOUND Portal до возврата | §23.1 |
| RECOMMENDATION-010 | BEH | Новый SEND предлагается только для UNEXPLORED Plane без Observer и при безопасном horizon | §23.1 |
| RECOMMENDATION-011 | BEH | EXPLORED Plane без Observer рекомендует CLOSE; опасный Portal без безопасной mission action также закрывается | §23.1 |
| RECOMMENDATION-012 | INV | Hypothetical Stabilize не мутирует state и учитывает command eligibility/Laboratory Energy | §23.1 |

## EVENT

| ID | Type | Rule | Spec |
|---|---|---|---|
| EVENT-001 | BEH | Каждое meaningful transition создаёт Event | §26 |
| EVENT-002 | BEH | Risk-событие только при смене level | §26 |
| EVENT-003 | BEH | Отклонённое действие создаёт ACTION_REJECTED | §26 |
| EVENT-004 | BEH | Portal History = Events, отфильтрованные по `portal_id` | §25 |
| EVENT-005 | INV | Global Event Log использует тот же источник событий | §26 |
| EVENT-006 | BEH | Extraction opening создаёт только `EXTRACTION_PORTAL_OPENED`, без дублирующего `PORTAL_OPENED` | §26.1 |
| EVENT-007 | BEH | При завершении outbound transit в Plane создаются `OBSERVER_ARRIVED` и `RESEARCH_STARTED`; возврат создаёт `OBSERVER_RETURNED` | §26.1 |
| EVENT-008 | BEH | Каждый Collapse создаёт `LEYLINE_OVERRIDE_STARTED`; окончание текущего окна создаётся ровно один раз | §26.1 |
| EVENT-009 | BEH | Перескок нескольких Risk-границ создаёт одно изменение previous→current; open/terminal transition сами его не создают | §26.1 |
| EVENT-010 | BEH | Все domain-отказы, включая confirmation-required, создают `ACTION_REJECTED`; transport parsing/route errors — нет | §26.1 |
| EVENT-011 | BEH | Event Log и Portal History хронологические и без пагинации в MVP | §26.1 |

## TUTORIAL

| ID | Type | Rule | Spec |
|---|---|---|---|
| TUTORIAL-001 | BEH | Deterministic state machine | §28 |
| TUTORIAL-002 | BEH | Step продвигается только после expected completion condition | §28 |
| TUTORIAL-003 | BEH | Wrong reversible action → тот же step | §28 |
| TUTORIAL-004 | BEH | Wrong irreversible action → recreate equivalent Tutorial Portal, повторить step | §28 |
| TUTORIAL-005 | BEH | Stabilize step гарантирует Risk HIGH/CRITICAL→MEDIUM/LOW | §28, §28.2 |
| TUTORIAL-006 | BEH | Critical step завершается именно rejected SEND | §28 |
| TUTORIAL-007 | BEH | Exploration только после успешного return (шаг 7) | §28 |
| TUTORIAL-008 | BEH | Tutorial Energy не сбрасывается при переходе в Live | §28 |
| TUTORIAL-009 | BEH | Step 0 не меняется от tick; explicit intro signal создаёт prepared Portal и переводит к Step 1 | §28.1–28.2 |
| TUTORIAL-010 | BEH | Details/Event Log продвигают Tutorial только через явный signal command; GET не мутирует state | §28.1, §35 |
| TUTORIAL-011 | BEH | Step 2 завершается ожиданием очистки corridor без обязательного rejected SEND | §28.1 |
| TUTORIAL-012 | BEH | Истёкший/сломанный prepared Portal автоматически заменяется эквивалентным с новым ID | §28.1 |
| TUTORIAL-013 | BEH | LOST возвращает к Step 6; replacement проходит normal SEND/research/RECALL без teleport | §28.1–28.2 |
| TUTORIAL-014 | BEH | Tutorial start идемпотентен; reset полностью восстанавливает исходный Tutorial и удаляет его прежнюю историю | §28.1 |
| TUTORIAL-015 | BEH | Live разрешён только после Step 9; Tutorial Portals закрываются бесплатно, continuity сохраняется, OPEN Portals = 0 | §28.1 |
| TUTORIAL-016 | BEH | Каждый step задаёт contextual guidance, действие игрока, system transition и completion condition | §28.2 |
| TUTORIAL-017 | INV | Prepared Portals детерминированы требуемыми свойствами; broken target получает новый ID | §28.1–28.2 |
| TUTORIAL-018 | INV | Natural Generator отключён во всём Tutorial и запускается только при переходе в Live | §28.1–28.2 |
| TUTORIAL-019 | UI | Step 0 содержит только лор, цель и карту интерфейса; цены и правила показываются на релевантных шагах | §28.2 |
| TUTORIAL-020 | UI | Tutorial — overlay без layout shift/More context; skipped completed step показывается 7 sec; Back/Forward не пропускают gameplay | §28.3 |

---

## Транспорт / UI / Persistence / Sessions (выведены напрямую из Final Spec)

### API

| ID | Type | Rule | Spec |
|---|---|---|---|
| API-001 | BEH | `GET /api/state` | §35 |
| API-002 | BEH | `GET /api/portals/{id}` | §35 |
| API-003 | BEH | `GET /api/events` | §35 |
| API-004 | BEH | `POST /api/portals/{id}/stabilize` | §35 |
| API-005 | BEH | `POST /api/portals/{id}/close` | §35 |
| API-006 | BEH | `POST /api/portals/{id}/send-observer` | §35 |
| API-007 | BEH | `POST /api/portals/{id}/recall-observer` | §35 |
| API-008 | BEH | `POST /api/extraction/open` | §35 |
| API-009 | BEH | `POST /api/tutorial/start`, `POST /api/tutorial/reset`, `POST /api/live/start` | §35 |
| API-010 | BEH | Confirmation flow: повтор того же endpoint с `{"confirm": true}` | §29 |
| API-011 | BEH | Backend возвращает domain errors; типичный conflict — `409` | §29 |
| API-012 | BEH | `POST /api/tutorial/signal` принимает закрытый enum UI-сигналов и не смешивает reads с mutations | §28.1, §35 |
| API-013 | INV | Каждый REST request разрешает anonymous session и выполняется только в её `lab_id`; cross-lab entity выглядит как not found | §35.1, §39 |

### WS

| ID | Type | Rule | Spec |
|---|---|---|---|
| WS-001 | BEH | Endpoint `/ws/lab` | §35 |
| WS-002 | BEH | Broadcast authoritative snapshot ~1/sec (тик симуляции) | §33, §35 |
| WS-003 | BEH | Немедленный snapshot после значимых действий | §35 |
| WS-004 | INV | `/ws/lab` использует ту же anonymous session и получает broadcasts только своей `lab_id` | §31, §35.1 |

### UI

| ID | Type | Rule | Spec |
|---|---|---|---|
| UI-001 | UI | Dashboard: 7 равных фиксированных Slots без scrolling; desktop `4 + centered 3`, compact `2+2+2+1`; summary, actions, Needs Attention, Empty State | §5, §24, §24.1, §27 |
| UI-002 | UI | Dashboard Slot НЕ показывает Risk / Recommendation / History | §5 |
| UI-003 | UI | Portal Details `/portals/:id` без page scrolling: центральный Portal/Actions, боковые facts/diagnostics и внутренняя scrolling History | §25, §25.1 |
| UI-004 | UI | Global Event Log `/events`; timestamp → title → details, формат `HH:mm:ss-dd-MM-yyyy` | §26 |
| UI-005 | UI | AI Worklog `/ai-worklog` с требуемым содержимым | §36 |
| UI-006 | UI | Energy показывается с одним десятичным; Risk number / decay / hidden timestamp / energy_lifetime не показываются | §9, §13 |
| UI-007 | UI | Needs Attention: один priority портал (highest risk_score → UNSTABLE → lower effective_lifetime → older opened_at); Risk number не показывается | §6 |
| UI-008 | UI | Все routes используют одну цельную magical-laboratory surface; левая zone не отделяется собственным background/frame | §30.1 |
| UI-009 | UI | Empty Slot controls действительно disabled; команды имеют pressed/pending/success/failure feedback и не дублируются | §24.1, §29 |
| UI-010 | UI | Tutorial overlay не сдвигает layout; без More context; skipped completed step показывается 7 sec; Back не пропускает будущие steps | §28.3 |
| UI-011 | UI | `/help` объясняет lore и все игровые механики; `AI Workflow` — последний navigation item и ведёт на `/ai-worklog` | §30.1, §36, §36.1 |
| UI-012 | UI | Terminal Portal grayscale; UNSTABLE card red; Override меняет весь Dashboard; reduced-motion сохраняет читаемость state | §24.1, §25.1, §30.1 |
| UI-013 | UI | Все 85 Plane artworks локальны, без card text/frame; manifest сохраняет source/artist/policy metadata | §30.1 |
| UI-014 | UI | Dashboard и Details показывают authoritative Observer transit direction и remaining time | §5, §25.1, §35.1 |

### PERSIST

| ID | Type | Rule | Spec |
|---|---|---|---|
| PERSIST-001 | BEH | Таблицы `labs`, `sessions`, `planes`, `portals`, `observers`, `events`, `lab_state`, `app_state`; gameplay rows принадлежат `lab_id` | §34 |
| PERSIST-002 | INV | Не обновлять derived realtime поля (energy и т.п.) каждую секунду | §34 |
| PERSIST-003 | BEH | Persist meaningful transitions and baselines | §34 |
| PERSIST-004 | BEH | Restart recovery из persisted state | Roadmap Stage 10 |
| PERSIST-005 | INV | Каждый tenant-owned read/write scoped по `lab_id`; keys/FKs не допускают cross-lab references | §34, §39 |
| PERSIST-006 | BEH | Existing singleton state мигрирует в legacy lab; expired lab удаляется каскадно без race с renewal/action | §34, §35.1 |

### SESSION

| ID | Type | Rule | Spec |
|---|---|---|---|
| SESSION-001 | BEH | Первый request без valid cookie атомарно создаёт lab, bootstrap state и anonymous session | §35.1 |
| SESSION-002 | BEH | Session имеет sliding expiry 30 days; REST/active WS считаются activity; browser restart сохраняет игру, expired/cleared cookie начинает новую | §35.1 |
| SESSION-003 | SEC | Cookie `HttpOnly`, `SameSite=Lax`, production `Secure`; token содержит ≥256 bits entropy и хранится только как hash | §35.1, §39 |
| SESSION-004 | INV | Один browser profile/tabs используют одну lab; другой profile/device получает изолированную lab | §35.1 |
| SESSION-005 | BEH | Background cleanup удаляет expired sessions/labs и не удаляет конкурентно продлённую active lab | §34, §35.1 |

---

## Infrastructure (не входит в ID-семейства; покрывается пакетными тестами)

Не являются продуктовыми requirements, но нужны для TDD (Final Spec / Stage 0–1 plan):

- Production domain code не вызывает `time.Now()` напрямую — только `clock.Clock` (`internal/clock`).
- Tests используют `testutil.FakeClock.Advance`; без `time.Sleep` в domain-тестах.
- Production random — `random.Random` (`internal/random`); tests — `testutil.FakeRandom` (preloaded, fail-loud).
- Test builders — фикстуры в `testutil`, не production-конструкторы.
- Balance config централизован в `internal/config`; сверяется тестом с Final Spec §37.
