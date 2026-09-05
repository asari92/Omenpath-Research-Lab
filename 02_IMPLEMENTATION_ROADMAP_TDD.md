# Omenpath Research Lab — Implementation Roadmap (TDD)

## Общий принцип

Единственный source of truth — `00_FINAL_SPEC_v5.md`.

Backend/domain разрабатывается через TDD:

```text
Requirement → Test → RED → Minimal Implementation → GREEN → Refactor
```

Полностью расписывать все будущие этапы до последней функции заранее не
требуется. Stages 0–8 выполнялись по отдельным detailed plans; после Stage 8
rolling-wave planning работает блоками: подробно планируется только ближайший
Block, он реализуется stage-by-stage, затем проходит обязательную пользовательскую
сверку перед планированием следующего Block.

## Текущее состояние delivery

Статус отражает фактически доказанную границу блока; отдельные оставшиеся
`PARTIAL`/`PLANNED` requirements всегда перечислены точнее в
[`docs/traceability.md`](docs/traceability.md).

| Block | Stages | Status | Execution documents | Boundary |
|---|---:|---|---|---|
| A — Specification & Domain Foundation | 0–2 | GREEN | `03_STAGE_00_01_TDD_FOUNDATION.md`, `04_STAGE_02_PORTAL_CORE_TDD.md` | завершён и проверен |
| B — Observers & Simulation | 3–8 | GREEN | `05_STAGE_03_OBSERVER_LIFECYCLE_TDD.md` … `10_STAGE_08_SIMULATION_TDD.md` | завершён и проверен |
| C — Persistence & Transport | 9–14 | GREEN | `11_BLOCK_C_STAGE_09_14_TDD.md` | завершён; итоговый audit APPROVED |
| D — Frontend | 15–21 | PLANNED | `12_BLOCK_D_STAGE_15_21_TDD.md`; frontend design spec | detailed plan утверждён; implementation не начат |
| E — Quality & Delivery | 22–27 | PLANNED | создаётся только после сверки Block D | начинать нельзя |

### Блоковый delivery-режим после Stage 8

Из-за срока сдачи Stages 9–27 выполняются блоками:

```text
Block C = Stages 9–14
Block D = Stages 15–21
Block E = Stages 22–27
```

На каждый блок создаётся один detailed plan с отдельными scope/DoD и
stage-level RED/GREEN evidence. Дополнительное подтверждение между стадиями
одного утверждённого блока не требуется. Каждая стадия заканчивается полным
ordinary suite и обновлением traceability/worklog. На границе блока повторяется
полный quality suite с race detector, выполняется сверка с Final Spec и работа
останавливается для пользовательской «сверки часов». Следующий блок до этой
сверки не начинается.

Полное описание режима:
[`docs/superpowers/specs/2026-09-04-block-delivery-mode-design.md`](docs/superpowers/specs/2026-09-04-block-delivery-mode-design.md).

## Block A — Specification & Domain Foundation

**Status: GREEN.** Stages 0–2 завершены; подробные границы и evidence находятся
в `03_STAGE_00_01_TDD_FOUNDATION.md`, `04_STAGE_02_PORTAL_CORE_TDD.md`, worklog
и traceability.

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

**Status: GREEN.** Stages 3–8 завершены по планам
`05_STAGE_03_OBSERVER_LIFECYCLE_TDD.md` … `10_STAGE_08_SIMULATION_TDD.md` и
прошли итоговую корректирующую сверку.

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

**Status: GREEN.** Detailed execution-plan:
`11_BLOCK_C_STAGE_09_14_TDD.md`. Stages 9–14 выполнены последовательно,
traceability/worklog обновлены, полный ordinary/race suite и независимый
итоговый audit прошли. Stage 15 в этом блоке не начинался.

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
- Context-cancellable exclusive ownership lock.
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
- Deterministic Recommendation Engine для Portal Details по Final Spec §23.1.
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

**Status: PLANNED; detailed plan готов, implementation не начат.** Текущий
execution document — [`12_BLOCK_D_STAGE_15_21_TDD.md`](12_BLOCK_D_STAGE_15_21_TDD.md),
визуальная и interaction-модель зафиксирована в
[`docs/superpowers/specs/2026-09-05-block-d-frontend-design.md`](docs/superpowers/specs/2026-09-05-block-d-frontend-design.md).
Оба документа проверены против Final Spec и текущих REST/WebSocket/Tutorial
contracts. Frontend потребляет существующий публичный transport contract и не
вводит новые gameplay semantics. Если UI выявляет реальный backend defect или
недостающую интеграционную границу, она исправляется отдельным доказанным TDD
corrective pass без произвольного изменения завершённых Stages 0–14.

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

**Status: PLANNED.** Block E нельзя начинать до завершения Block D, его полного
boundary verification и пользовательской сверки.

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

## Detailed-planning and verification rule

Before implementing an approved stage or block:
1. required product/domain contracts must already be fixed in Final Spec;
2. the current detailed plan must be checked against Final Spec;
3. scope, checkpoints, RED/GREEN evidence and per-stage DoD must be explicit;
4. implementation of the following block must remain out of scope.

Inside an approved block, stages proceed sequentially without additional user
approval. After every stage:
1. `gofmt -l .`, `go vet ./...`, `go build ./...` and
   `go test -count=1 ./...` pass;
2. traceability is updated to the actually proven boundary;
3. actual decisions, errors and RED/GREEN hashes are appended to Worklog;
4. later stages in the current block are reviewed for consequences.

At every block boundary:
1. the complete ordinary suite is repeated;
2. `go test -race -count=1 ./...` passes;
3. Final Spec, roadmap, requirements, traceability, worklog, plans and actual
   implementation are reconciled;
4. the worktree is clean and the next block has not started;
5. work stops for the required user checkpoint.
