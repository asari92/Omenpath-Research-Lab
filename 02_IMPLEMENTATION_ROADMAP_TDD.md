# Omenpath Research Lab — Implementation Roadmap (TDD)

## Общий принцип

Единственный source of truth — `00_FINAL_SPEC_v5.md`.

Backend/domain разрабатывается через TDD:

```text
Requirement → Test → RED → Minimal Implementation → GREEN → Refactor
```

Полностью расписывать все будущие этапы до последней функции заранее не требуется. Используем rolling-wave planning: подробный план ближайших 1–2 этапов, реализация, сверка, затем детализация следующих.

### Блоковый delivery-режим после Stage 8

Из-за срока сдачи Stages 9–27 выполняются блоками:

```text
Block C = Stages 9–14
Block D = Stages 15–21
Block E = Stages 22–27
```

На каждый блок создаётся один detailed plan с отдельными scope/DoD и
RED/GREEN evidence для входящих стадий. Дополнительное подтверждение между
стадиями одного утверждённого блока не требуется. После каждого блока
обязательны полный verification suite, traceability/worklog update, сверка с
Final Spec и остановка для пользовательской «сверки часов». Следующий блок до
этой сверки не начинается.

Полное описание режима:
[`docs/superpowers/specs/2026-09-04-block-delivery-mode-design.md`](docs/superpowers/specs/2026-09-04-block-delivery-mode-design.md).

## Block A — Specification & Domain Foundation

### Stage 0 — Executable specification
- Final Spec фиксируется как source of truth.
- Fixed domain rules отделяются от balance config.
- Требования получают ID.
- Создаётся traceability `requirement → test → implementation`.

### Stage 1 — TDD foundation
- Go module/project skeleton.
- Clock / FakeClock.
- Random / FakeRandom.
- Test builders.
- Первый набор RED tests.
- `go test ./...` и `go test -race ./...`.

### Stage 2 — Portal Core via TDD
1. Portal state.
2. Natural Close.
3. Portal Energy.
4. Energy Collapse.
5. Stability.
6. Hidden Instability Collapse.
7. Stabilize.
8. Creatures.
9. Risk.
10. Portal Slots.

## Block B — Observers & Simulation

### Stage 3 — Observer lifecycle
- AVAILABLE → OUTBOUND → EXPLORING → WAITING_RETURN → RETURNING → AVAILABLE.
- LOST.
- Transit 5–15 sec.
- Research 20 sec.
- Plane EXPLORED only after successful return.

### Stage 4 — Portal observer flow
- NONE / OUTBOUND / INBOUND.
- One transit at a time.
- Direction restrictions.

### Stage 5 — Laboratory Energy
- 0–100 integer.
- +1/sec.
- Close 5, Stabilize 20, Send/Recall 0, Extraction 30.
- Insufficient-energy rules.

### Stage 6 — Emergency
- Collapse → Lab Energy 0.
- Leyline Override 20 sec.
- Close/Stabilize free during Override.

### Stage 7 — Extraction
- Plane selection.
- Cost 30.
- Stable INBOUND Portal.
- 5 sec sync.
- First return automatic.
- Further returns manual.

### Stage 8 — Simulation
- Natural spawn delay 0–20 sec.
- Maximum 7 OPEN Portals.
- Tick processing.
- Needs Attention.

## Block C — Persistence & Transport

Detailed execution-plan: `11_BLOCK_C_STAGE_09_14_TDD.md`. После его
утверждения Stages 9–14 выполняются последовательно без промежуточного
пользовательского подтверждения; после Stage 14 обязательна блоковая сверка.

### Stage 9 — Event system
- Domain events.
- Portal history.
- Global Event Log.
- ACTION_REJECTED.
- Risk level changes.

### Stage 10 — SQLite persistence
- Migrations.
- 85 Planes seed.
- Portals, Observers, Events, Lab/App state.
- Restart recovery.

### Stage 11 — LabManager / concurrency
- In-memory active state.
- `sync.RWMutex`.
- Atomic transitions.
- No duplicate events.
- Race detector.

Orchestration invariant (fixed after the Stage 0–2 audit, before any
time-sensitive command is implemented):

```text
Before executing ANY time-sensitive command (Close, Stabilize, Send,
Recall, Extraction, reads of derived state) on a portal, LabManager MUST
first call ResolveLifecycle(portal, now) under the same lock.
If the portal became terminal during that resolution, the requested
action MUST NOT be executed (it fails with the domain "not open" error);
the terminal transition itself is processed as the winning transition.
```

Rationale:

- exactly one valid transition wins (Final Spec §32): either the user
  command or the scheduled lifecycle event, never both;
- commands never operate on a portal that is already semantically
  terminal but not yet resolved by a tick (late resolution);
- this rule lives in LabManager orchestration, NOT inside
  `Portal.Close` / `Portal.Stabilize` — keeping the portal primitives
  pure simplifies emitting lifecycle Events from a single place later
  (Stage 9) and avoids double-resolution logic.

### Stage 12 — REST API
- Reads.
- Portal actions.
- Extraction.
- Domain errors.
- `409` confirmation flow.

### Stage 13 — WebSocket
- WS hub.
- Initial snapshot.
- 1 sec broadcast.
- Immediate snapshot after significant action.
- Reconnect behaviour.

### Stage 14 — Tutorial engine
- Deterministic state machine.
- Expected completion conditions.
- Retry/recreate.
- Tutorial → Live continuity.

## Block D — Frontend

### Stage 15 — Frontend foundation
React + TypeScript + Vite, router, REST client, WebSocket client, shared state, error handling.

### Stage 16 — Dashboard
Summary, Laboratory Energy, research progress, observer counts, Needs Attention, 7 fixed Portal Slots, quick actions, Empty State, Leyline Override.

### Stage 17 — Portal Details
Portal data, destination data, Risk, Recommendation, How Risk Works, History, actions.

### Stage 18 — Extraction / confirmations / errors
Plane selection, confirmation modals, unstable warning, toast/inline errors.

### Stage 19 — Event Log
Global log and filters.

### Stage 20 — Tutorial UI
Instructions, current step, retry behaviour, completion.

### Stage 21 — AI Worklog UI
Analysis/architecture, backend, frontend, testing, debugging, deployment, final QA.

## Block E — Quality & Delivery

### Stage 22 — Full automated test pass
Domain, integration, persistence, REST, WS, Tutorial, concurrency, race detector.

### Stage 23 — Balance simulation
10k–100k simulated Portals. Measure collapse rates, observer loss, energy economy, 7/7 frequency, exploration rate. Tune config only.

### Stage 24 — README / DevEx
README, `.env.example`, `.gitignore`, Makefile, Docker, run/test/build commands.

### Stage 25 — Deployment
Public app, WebSocket, persistence, restart, health checks.

### Stage 26 — Final QA
Pass the app as an external evaluator.

### Stage 27 — Consistency Audit
Cross-check:
- Final Spec
- Requirement Catalog
- Traceability Matrix
- all Stage Plans
- Domain Models
- DB Schema
- REST Contract
- WebSocket Contract
- Tutorial
- Frontend
- Tests
- AI Worklog
- README
- Actual Implementation

Look for stale constants, enum mismatches, outdated restrictions, UI actions without backend rules, tests for old mechanics and docs describing old architecture.

## Detailed-planning rule

Before implementing a stage:
1. its required domain contracts must already be fixed;
2. the stage plan must be checked against Final Spec;
3. its DoD must be explicit.

After implementing a stage:
1. tests and race detector pass;
2. traceability is updated;
3. actual AI usage/errors are appended to Worklog;
4. future plans are reviewed for consequences.
