# Tutorial guidance и system transitions — дизайн Block C

## 1. Цель

Tutorial должен не показывать справочник заранее, а объяснять механику в момент,
когда игрок применяет её. Каждый step имеет четыре явных контракта:

```text
что объясняется
→ что делает игрок
→ что изменяет система
→ какое состояние завершает step
```

Backend Stage 14 владеет state machine, prepared entities, targets, phases и
completion conditions. Frontend Stage 20 владеет формулировками и визуальной
подачей, следуя обязательному содержанию этого design и Final Spec §28.2.

## 2. Persisted Tutorial context

`AppState` расширяется сохраняемым контекстом:

```go
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
```

Persisted fields: `tutorial_step`, `tutorial_phase`, target Portal/Plane/Observer
IDs. `expected_action` выводится детерминированно из step/phase и отдельно не
хранится. Restart продолжает тот же step и не создаёт дубликаты targets/events.

Public snapshot возвращает step, phase, target IDs и expected action. Он не
возвращает hidden prepared values.

## 3. Signals и Step 0

Allowed signal enum:

```text
TUTORIAL_INTRO_COMPLETED
PORTAL_DETAILS_OPENED
EVENT_LOG_OPENED
```

Step 0 не меняется от simulation ticks. До явного intro signal:

- Mode = TUTORIAL;
- Step = 0;
- 0 OPEN Portals;
- Natural Generator paused;
- UI объясняет лор лаборатории, цель 85/85 и расположение основных частей
  интерфейса;
- action prices, Risk rules и Observer lifecycle ещё не перечисляются.

`TUTORIAL_INTRO_COMPLETED` одной transaction создаёт Step 1 target Portal,
сохраняет Step 1 и событие открытия, затем отправляет realtime update.
`PORTAL_DETAILS_OPENED` дополнительно несёт обязательный `portal_id`; два
остальных signal не принимают target ID.

## 4. Полная step machine

### Step 1 — Portal Details

Prepared target:

- NATURAL, OPEN, STABLE;
- destination — детерминированно выбранный UNEXPLORED Plane;
- creatures > 0;
- Energy/lifetime гарантированно позволяют дождаться clearance и выполнить
  max outbound transit;
- flow NONE.

Guidance объясняет Portal Energy отдельно от Lab Energy, индивидуальную скорость
расходования, независимость scheduled time от energy depletion, Stability,
Risk, Recommendation и History.

Игрок открывает именно target Details. GET остаётся read-only; отдельный signal
с matching Portal ID переводит к Step 2. Wrong ID оставляет Step 1.

### Step 2 — Corridor

Guidance объясняет creature block и интервал 2 sec. Игрок ждёт. Domain ticks
уменьшают derived creatures; при нуле система автоматически ставит Step 3.
Target остаётся тем же.

Если target стал terminal до completion, сохраняется его история и создаётся
новый эквивалентный Portal с новым ID, creatures > 0 и безопасным horizon.

### Step 3 — SEND

Guidance: SEND cost 0, AVAILABLE Observer, transit 5–15 sec, first use fixes
OUTBOUND. Игрок вызывает SEND на target. Обычная domain-команда выбирает
lowest-ID AVAILABLE Observer и создаёт обычные Events.

После успеха система сохраняет target Observer/Plane и создаёт отдельный Step 4
target. Observer transit и research продолжаются обычными ticks. Step 6 phase
будет определена при фактическом входе на этот step.

### Step 4 — STABILIZE

Prepared target:

- OPEN, UNSTABLE;
- current Portal Energy ≤85%;
- current Risk HIGH или CRITICAL;
- обычный Stabilize +15 и removal instability дают Risk MEDIUM или LOW;
- lifetime позволяет игроку выполнить действие; expiry пересоздаёт target.

Guidance: cost 20 Lab Energy, Lab range 0–100, regen +1/sec, precondition ≤85%,
+15 Portal Energy и UNSTABLE→STABLE. Игрок вызывает обычный STABILIZE. Команда
списывает реальную effective cost и создаёт `PORTAL_STABILIZED` плюс одно
`RISK_LEVEL_CHANGED`.

Completion требует STABLE и band transition HIGH/CRITICAL→MEDIUM/LOW. Затем
система создаёт Step 5 target.

### Step 5 — expected critical rejection

Prepared target:

- OPEN, STABLE, CRITICAL;
- CRITICAL получается из короткого scheduled lifetime;
- Energy depletion наступает после NATURAL_CLOSE, поэтому target не вызывает
  учебный Collapse;
- corridor clear и flow NONE.

Guidance: CRITICAL blocks SEND/RECALL, CLOSE cost 5, CLOSED отличается от
COLLAPSED, Collapse resets Lab Energy to 0 и starts/restarts 20-sec Override.

Игрок вызывает SEND. `ErrPortalCriticalRisk` является ожидаемым результатом:
той же transaction сохраняются `ACTION_REJECTED` и Step 6. HTTP остаётся 409,
а realtime snapshot уже показывает новый step. До попытки истёкший target
пересоздаётся; после completion он просто заканчивает normal lifecycle.

### Step 6 — research и RECALL

Обычный путь после Step 3:

1. при входе Observer OUTBOUND/EXPLORING → phase WAIT_RESEARCH; уже
   WAITING_RETURN → сразу создать return Portal и phase RECALL_READY; LOST →
   phase SEND_REPLACEMENT;
2. tracked Observer проходит OUTBOUND→EXPLORING→WAITING_RETURN;
3. после `RESEARCH_COMPLETED` система создаёт новый OPEN/STABLE/clear Portal к
   тому же Plane с lifetime > ObserverTransitMax и flow NONE;
4. phase RECALL_READY, expected action RECALL_OBSERVER;
5. игрок вызывает RECALL; обычная selection выбирает longest-waiting Observer,
   Portal фиксирует INBOUND, Observer становится RETURNING;
6. Tutorial переходит к Step 7.

Guidance появляется по phase: сначала research duration 20 sec, затем RECALL
cost 0, longest-waiting и INBOUND direction.

### Step 7 — return и exploration

Expected action WAIT_RETURN. Ticks выполняют обычный return lifecycle.

- success: Observer AVAILABLE, Plane EXPLORED, события return/exploration,
  Step 8;
- portal terminal during transit: Observer LOST, события terminal/lost,
  Tutorial возвращается к Step 6 phase SEND_REPLACEMENT.

Retry не телепортирует Observer:

1. система создаёт безопасный outbound Portal к тому же ещё UNEXPLORED Plane;
2. игрок выполняет SEND другим AVAILABLE Observer;
3. phase WAIT_RESEARCH и обычные transit/research events;
4. новый safe inbound-capable Portal;
5. phase RECALL_READY и повтор RECALL.

Если AVAILABLE Observer больше нет, state остаётся на Step 6 с domain error и
возможностью явного Tutorial reset; фиктивный Observer не создаётся.

Если tracked Observer стал LOST из-за неправильного необратимого действия ещё
на Steps 4–5, вход в Step 6 использует тот же SEND_REPLACEMENT flow. Tutorial
не зависает из-за более ранней потери.

### Step 8 — Event Log

Guidance объясняет shared Event source. Игрок открывает Event Log. GET не
мутирует state; `EVENT_LOG_OPENED` signal переводит на Step 9.

### Step 9 — Live handoff

Guidance объясняет Natural Portals, Extraction cost 30, 5-sec synchronization,
automatic first return и Live Mode. Игрок вызывает live/start.

Одной transaction система:

1. проверяет Step 9;
2. бесплатно закрывает оставшиеся OPEN Tutorial Portals как MANUAL_CLOSE с
   event payload source `TUTORIAL_COMPLETION`;
3. сохраняет Lab Energy, Events, Observer states и Plane exploration;
4. выставляет Mode LIVE и очищает Tutorial targets/phase;
5. создаёт fresh Natural schedule;
6. публикует snapshot с 0 OPEN Portals.

## 5. Wrong actions и retry

- Wrong reversible command выполняет обычную domain semantics, но не меняет
  Tutorial step.
- Wrong irreversible command, сделавшая текущий scenario невозможным, сохраняет
  настоящий результат и Events, затем создаёт equivalent fresh target с новым
  Portal ID.
- Старый Portal никогда не resurrect.
- Recreate выполняется атомарно с обнаружившим его tick/command.
- Natural Generator не используется для retry и остаётся paused.
- Prepared values выбираются deterministic factory, проверяющей свойства, а не
  hardcoded ID или random sequence.

## 6. Content boundary

Step 0 содержит только:

- короткий лор Omenpath Research Lab;
- цель исследования 85 Planes;
- Dashboard, Lab Summary, Observer summary, seven Slots, Needs Attention,
  Portal Details и Event Log;
- Lab Energy как общий ресурс без списка цен.

Остальная информация появляется контекстно:

- Portal energy/time/risk — Step 1;
- creatures — Step 2;
- SEND/transit/outbound — Step 3;
- Lab numbers/STABILIZE/UNSTABLE — Step 4;
- CRITICAL/CLOSE/COLLAPSE/Override — Step 5;
- research/RECALL/inbound — Step 6;
- LOST/exploration — Step 7;
- events — Step 8;
- Extraction/Natural/Live — Step 9.

Конкретные decay rate, numeric risk score и hidden collapse timestamp не
раскрываются.

## 7. Verification

Tests обязаны отдельно доказать:

- ticks не меняют Step 0 и не создают Portals;
- intro signal атомарно создаёт ровно один target/event;
- каждый step реагирует только на свою action/condition/target;
- prepared Portal удовлетворяет свойствам, а не одному fixture ID;
- Step 4 допускает обе исходные и обе итоговые Risk bands;
- Step 5 одновременно возвращает rejection и сохраняет progress/event;
- Step 6 phases переживают SQLite restart;
- LOST retry использует normal Observer lifecycle и другого Observer;
- broken target получает новый ID без resurrection;
- Tutorial никогда не запускает Natural generator;
- Live transition сохраняет continuity и начинает с 0 OPEN Portals;
- REST/WS возвращают одинаковые step/phase/expected action;
- UI copy requirements остаются traceable до Stage 20.
