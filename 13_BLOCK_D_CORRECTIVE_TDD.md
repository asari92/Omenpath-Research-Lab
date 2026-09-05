# Block D Corrective Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** закрыть обязательный corrective gate Block D: на новой clean-start SQLite schema перейти с одной общей игры на изолированные anonymous laboratories, увеличить roster до 20 Observers, добавить authoritative transit projection и пересобрать frontend в согласованный цельный dark-fantasy интерфейс без page scroll на Dashboard и Portal Details.

**Architecture:** одна SQLite database хранит все laboratories в общей schema с обязательным `lab_id`; anonymous cookie разрешается до любого REST/WebSocket доступа, tenant-bound repository сохраняет прежний `engine.Repository`, а runtime registry выдаёт отдельный `LabManager` и `realtime.Hub` на laboratory. React принимает только authoritative snapshots, запускает WebSocket после успешного REST bootstrap и хранит animation/tutorial presentation отдельно от gameplay state.

**Tech Stack:** Go 1.26, SQLite/modernc, Chi, coder/websocket, Node 20.19+, npm, React 19, TypeScript 6, Vite 8, CSS Modules, Canvas 2D, Vitest 4, React Testing Library, MSW 2, Playwright 1.58.2, ESLint 10, Prettier 3, Sharp 0.35.4; без новых dependencies.

---

## 1. Основание и разрешённая граница

Приоритет документов:

1. `00_FINAL_SPEC_v5.md` — единственный product/domain source of truth.
2. `01_AI_WORKLOG_CURRENT.md` — фактическая история и corrective passes.
3. `02_IMPLEMENTATION_ROADMAP_TDD.md` — порядок blocks/stages.
4. `docs/superpowers/specs/2026-09-06-ui-ux-corrective-design.md` — согласованный corrective design.
5. Этот execution-plan.
6. `docs/requirements.md` и `docs/traceability.md`.

Документационная prerequisite завершена:

- `b74f7f0` — согласованный UI/UX и multi-lab design;
- `a3bbb9d` — Final Spec, roadmap, worklog, requirements и traceability;
- `63557ba` — session constants и WS activity;
- `dd6915d` — clean-start database без legacy import/claim.

Verified starting boundary:

- Stages 0–21 имеют историческое GREEN evidence;
- Block D остаётся `PARTIAL` до завершения corrective gate;
- server использует один global `Store`, `LabManager` и `Hub`;
- persistence tables не имеют `lab_id`;
- canonical roster равен 10, новый Final Spec требует 20;
- DTO показывает только aggregate transit counts;
- UI не удовлетворяет утверждённой layout/theme/tutorial модели;
- untracked `omenpath.db` принадлежит пользователю и не изменяется тестами/Git;
- Block E / Stage 22 начинать нельзя.

Порядок:

```text
DC-1 20 Observers
  → DC-2 multi-lab schema and tenant repository
  → DC-3 anonymous session lifecycle and cleanup
  → DC-4 per-lab runtime, REST and WebSocket isolation
  → DC-5 authoritative transit DTO and safe frontend bootstrap
  → DC-6 unified shell and fixed Dashboard
  → DC-7 no-scroll Details, Event Log and Help
  → DC-8 Tutorial presentation and interaction motion
  → DC-9 complete Plane artwork
  → DC-10 Block D verification, traceability and STOP
```

Для каждого checkpoint:

```text
requirements → tests → observed functional RED → RED commit
→ minimal implementation → focused GREEN → GREEN commit
```

RED не может быть syntax/config/dependency failure. Stage 22 не входит в план.

## 2. Зафиксированные contracts

### 2.1 Multi-lab persistence

Новая schema:

```sql
labs(id TEXT PRIMARY KEY, created_at INTEGER NOT NULL,
     expires_at INTEGER NOT NULL, last_active_at INTEGER NOT NULL)
sessions(id INTEGER PRIMARY KEY AUTOINCREMENT,
         token_hash BLOB NOT NULL UNIQUE CHECK(length(token_hash) = 32),
         lab_id TEXT NOT NULL UNIQUE REFERENCES labs(id) ON DELETE CASCADE,
         expires_at INTEGER NOT NULL, last_seen_at INTEGER NOT NULL)
planes(PRIMARY KEY(lab_id, id), FOREIGN KEY(lab_id) REFERENCES labs(id) ON DELETE CASCADE)
portals(PRIMARY KEY(lab_id, id),
        FOREIGN KEY(lab_id, destination_plane_id) REFERENCES planes(lab_id, id))
observers(PRIMARY KEY(lab_id, id),
          FOREIGN KEY(lab_id, current_plane_id) REFERENCES planes(lab_id, id),
          FOREIGN KEY(lab_id, active_portal_id) REFERENCES portals(lab_id, id))
events(PRIMARY KEY(lab_id, id), FOREIGN KEY(lab_id) REFERENCES labs(id) ON DELETE CASCADE)
lab_state(lab_id TEXT PRIMARY KEY REFERENCES labs(id) ON DELETE CASCADE)
app_state(lab_id TEXT PRIMARY KEY REFERENCES labs(id) ON DELETE CASCADE)
```

Open Slot uniqueness: `(lab_id, slot_index) WHERE status='OPEN'`. Event IDs
возрастают внутри laboratory: в одноподключённой SQLite transaction используется
`MAX(id)+1`, drafts вставляются в semantic order. Event entity references
остаются soft references, но event row всегда tenant-scoped.

Implementation рассчитан только на новую пустую database. `001_initial.sql`
сразу создаёт `labs`, `sessions` и tenant-owned gameplay tables с `lab_id`.
`002_tutorial_context.sql` сохраняется и добавляет Tutorial columns в уже
tenant-aware `app_state`. Migration 003, replacement tables, legacy import и
claim отсутствуют. Старый `omenpath.db` application автоматически не удаляет.

`Store.ForLab` возвращает tenant-bound repository:

```go
type LabID string

type LabRepository struct {
	store *Store
	labID LabID
}

func (s *Store) ForLab(id LabID) (*LabRepository, error)
func (r *LabRepository) Load(context.Context) (Snapshot, error)
func (r *LabRepository) Commit(context.Context, Snapshot, []domain.EventDraft) ([]domain.Event, error)
func (r *LabRepository) ListEvents(context.Context, *int64) ([]domain.Event, error)
func (r *LabRepository) ResetTutorial(context.Context, Snapshot) error
```

No unbound gameplay read/write method remains exported on `Store`.

### 2.2 Anonymous session

`config.Config`:

```go
SessionTTL             = 30 * 24 * time.Hour
SessionRefreshInterval = 12 * time.Hour
SessionCleanupInterval = time.Hour
SessionTokenBytes      = 32
LabIDBytes             = 16
SessionCookieName      = "omenpath_session"
```

Raw token — base64url без padding, 32 bytes from `crypto/rand`; DB хранит только
`sha256(rawToken)`. Lab ID — lowercase hex of 16 random bytes. Cookie:
`Path=/`, `HttpOnly`, `SameSite=Lax`, `MaxAge=2592000`; `Secure` задаётся
`OMENPATH_COOKIE_SECURE` и обязан быть true в production. Cookie также получает
`Expires=now+30d` и перевыдаётся при bounded server-side refresh.

Resolution transaction:

1. valid non-expired hash возвращает lab;
2. после 12h session/lab expiry обновляются до `now + 30d`;
3. missing/invalid/expired cookie создаёт fresh token;
4. одна transaction создаёт lab, 85 Planes, 20 Observers, Lab/App и session;
5. failure не оставляет partial lab/token.

REST resolution или WS handshake являются activity. Live WS держит runtime lease.
Cleanup берёт candidate, затем под registry idle gate выполняет conditional
`DELETE ... WHERE expires_at <= now`: concurrent renewal безопасно выигрывает.

### 2.3 Runtime isolation

`internal/labruntime.Registry`:

```go
type Runtime struct {
	Manager *engine.LabManager
	Hub     *realtime.Hub
}
type Lease struct { Runtime *Runtime }

func (r *Registry) Acquire(context.Context, persistence.LabID) (*Lease, error)
func (l *Lease) Release()
func (r *Registry) TickAll(context.Context) error
func (r *Registry) DeleteIfIdle(persistence.LabID, func() error) (bool, error)
func (r *Registry) Close()
```

`Acquire` single-flight. REST держит lease до окончания response, WS — до
disconnect. `DeleteIfIdle` сериализует eviction/delete с Acquire. Каждый manager
получает свой checkpointable random source.

### 2.4 Transit DTO

```go
type ObserverTransitDTO struct {
	ObserverID       int64                 `json:"observer_id"`
	PortalID         int64                 `json:"portal_id"`
	Direction        domain.ObserverStatus `json:"direction"`
	StartedAt        time.Time             `json:"started_at"`
	CompletesAt      time.Time             `json:"completes_at"`
	RemainingSeconds int64                 `json:"remaining_seconds"`
}
```

`StateSnapshot.ObserverTransits` содержит active OUTBOUND/RETURNING, sorted by
Observer ID. `PortalDetails.ObserverTransit` — matching transit или null.
Required phase fields nil/mismatched означают `ErrSimulationInvariant`.
Remaining вычисляется от resolved server time и clamp до zero.

### 2.5 Frontend bootstrap and visual invariants

Первый `GET /api/state` должен завершиться/установить cookie до WebSocket:

```ts
const snapshot = await api.state(signal);
store.acceptSnapshot(snapshot);
realtime.start();
```

При reconnect последний snapshot остаётся видимым, commands блокируются.

- одна decorative frame и поверхность на весь `100dvh`;
- левая layout-zone без отдельной sidebar: stats сверху, nav/actions снизу;
- порядок: Dashboard, Event Log, Help, Open Extraction, Artwork Credits,
  AI Worklog последним;
- labels: `Planar link stable`, `Planar paths unstable`,
  `Disconnected from the planes`;
- desktop Dashboard exact `4 + centered 3` на восьми columns;
- phone `2 + 2 + 2 + centered 1`; все seven visible;
- occupied/empty equal; empty actions native disabled;
- occupied unavailable объясняется без POST;
- UNSTABLE меняет whole card, Override — whole shell;
- Details no page scroll: center Portal/actions, side facts, bottom History с
  единственным inner scroll;
- terminal Portal fully grayscale;
- Risk LOW green, MEDIUM yellow, HIGH orange, CRITICAL red;
- Event: `HH:mm:ss-dd-MM-yyyy`, затем title, затем details;
- Tutorial floating, no More Context, не закрывает target;
- normal action advances immediately; skipped intermediate completed step 7s;
- replacement presentation delay 2s, backend не задерживается;
- command states pressed → pending → success/error pulse + toast;
- reduced motion обязателен.

## 3. Целевая карта файлов

```text
internal/config/config.go
internal/config/config_test.go
internal/persistence/migrations/001_initial.sql
internal/persistence/migrations/002_tutorial_context.sql
internal/persistence/lab_id.go
internal/persistence/lab_repository.go
internal/persistence/lab_repository_test.go
internal/persistence/session_store.go
internal/persistence/session_store_test.go
internal/persistence/store.go
internal/persistence/state.go
internal/persistence/load.go
internal/persistence/store_migration_test.go
internal/persistence/tutorial_reset_test.go
internal/session/service.go
internal/session/service_test.go
internal/labruntime/registry.go
internal/labruntime/registry_test.go
internal/httpapi/session_context.go
internal/httpapi/session_middleware.go
internal/httpapi/session_middleware_test.go
internal/httpapi/router.go
internal/httpapi/router_test.go
internal/httpapi/realtime_test.go
internal/httpapi/multilab_integration_test.go
internal/transport/dto.go
internal/transport/dto_test.go
cmd/server/main.go
cmd/server/main_test.go

web/src/api/types.ts
web/src/api/realtime.ts
web/src/api/realtime.test.ts
web/src/state/SnapshotProvider.tsx
web/src/state/SnapshotProvider.test.tsx
web/src/state/selectors.ts
web/src/state/selectors.test.ts
web/src/styles/tokens.css
web/src/styles/global.css
web/src/app/AppShell.tsx
web/src/app/AppShell.module.css
web/src/app/App.test.tsx
web/src/app/router.tsx
web/src/components/dashboard/*
web/src/components/feedback/*
web/src/components/events/*
web/src/components/portal/*
web/src/features/portal-actions/*
web/src/features/tutorial/*
web/src/pages/DashboardPage*
web/src/pages/PortalDetailsPage*
web/src/pages/EventLogPage*
web/src/pages/HelpPage*
web/src/portal-fx/*
web/src/assets/plane-art*
web/scripts/audit-plane-art.mjs
web/public/planes/*.webp
data/plane_image_sources.json
data/plane_images_manifest.json
data/plane_art_audit.json
web/e2e/session-isolation.spec.ts
web/e2e/dashboard.spec.ts
web/e2e/portal-details.spec.ts
web/e2e/events.spec.ts
web/e2e/tutorial.spec.ts
web/e2e/app-integration.spec.ts
```

## 4. DC-1 — 20 Observers

Requirements: `OBSERVER-001`, `OBSERVER-014`, `TUTORIAL-014`.

### Task 1.1 — RED

- [ ] Require `config.Default().ObserverCount == 20`.
- [ ] Require bootstrap/reset IDs 1–20, exactly 20, all reset AVAILABLE.
- [ ] Replace canonical test fixtures hard-coding 10 with configured count;
  preserve unit cases where 10 is merely local input.
- [ ] Add transport assertion that aggregate status total is 20.
- [ ] Run:

```bash
go test -count=1 ./internal/config ./internal/persistence ./internal/transport
```

Expected RED: actual canonical count is 10.

- [ ] Commit:

```bash
git add internal
git commit -m "test(block-d-corrective): RED require twenty observers"
```

### Task 1.2 — GREEN

- [ ] Set default to 20.
- [ ] Remove canonical literal 10 from persistence validation/capacity/reset.
- [ ] Keep tutorial reset atomic and scoped to its laboratory later in DC-2.
- [ ] Run `go test -count=1 ./...`.
- [ ] Commit:

```bash
git add internal
git commit -m "feat(block-d-corrective): GREEN expand roster to twenty observers"
```

## 5. DC-2 — Fresh multi-lab schema and tenant repository

Requirements: `PERSIST-001`–`PERSIST-006` fresh-schema half.

### Task 2.1 — RED

- [ ] Build a fresh database and assert the complete expected tables, composite
  tenant keys/FKs, indexes and Tutorial columns after migrations 001–002.
- [ ] Schema migration alone must not create a laboratory or gameplay rows;
  first session resolution owns canonical bootstrap.
- [ ] Bootstrap labs A/B with overlapping entity IDs; prove Load/Commit/Events/
  Reset cannot see or mutate the other.
- [ ] Prove cross-lab FKs fail, same-lab duplicate open Slot fails, same Slot in
  different labs succeeds.
- [ ] Run `go test -count=1 ./internal/persistence`.
- [ ] Commit:

```bash
git add internal/persistence
git commit -m "test(block-d-corrective): RED specify tenant scoped persistence"
```

### Task 2.2 — GREEN

- [ ] Rewrite `001_initial.sql` directly as the final multi-lab base schema.
- [ ] Retain `002_tutorial_context.sql` as the only additive migration and make
  its ALTER statements target tenant-aware `app_state`; do not add migration 003.
- [ ] Add strict `LabID` and `Store.ForLab`.
- [ ] Move Bootstrap/Load/Commit/ListEvents/ResetTutorial to `LabRepository`;
  every SQL statement scopes `lab_id`.
- [ ] Allocate per-lab event IDs within the atomic commit.
- [ ] Keep `Store` responsible only for DB/migrations/session operations.
- [ ] Run:

```bash
go test -count=1 ./internal/persistence
go test -count=1 ./internal/engine ./internal/httpapi ./internal/realtime
```

- [ ] Commit:

```bash
git add internal/persistence internal/engine internal/httpapi internal/realtime cmd
git commit -m "feat(block-d-corrective): GREEN scope persistence by laboratory"
```

## 6. DC-3 — Anonymous session lifecycle and cleanup

Requirements: `SESSION-001`–`SESSION-006`, `PERSIST-006`.

### Task 3.1 — RED

- [ ] With fake clock/bytes test token length/encoding, lab ID, hash-only storage
  and absence of raw token in errors.
- [ ] Prove two independent fresh tokens create two canonical isolated labs;
  concurrent resolutions create complete labs with no partial/duplicate rows.
- [ ] Prove reuse before expiry, no write before 12h, refresh at 12h, new lab after
  expiry.
- [ ] Prove session/bootstrap failure fully rolls back.
- [ ] Prove cascade cleanup and conditional no-delete after concurrent renewal.
- [ ] Verify all six session config constants.
- [ ] Run:

```bash
go test -count=1 ./internal/session ./internal/persistence ./internal/config
```

Expected RED: session package/store/config contracts are absent.

- [ ] Commit:

```bash
git add internal/session internal/persistence internal/config
git commit -m "test(block-d-corrective): RED specify anonymous session lifecycle"
```

### Task 3.2 — GREEN

- [ ] Add exact config defaults.
- [ ] Implement injected cryptographic generator; reject short output before DB.
- [ ] Implement hash lookup, bounded refresh, atomic new-lab bootstrap, cleanup
  candidates and conditional delete.
- [ ] Return raw token only for a newly generated cookie; on bounded refresh the
  middleware reuses the incoming cookie value transiently. Never retain/log it.
- [ ] Run focused tests.
- [ ] Commit:

```bash
git add internal/config internal/persistence internal/session
git commit -m "feat(block-d-corrective): GREEN add anonymous laboratory sessions"
```

## 7. DC-4 — Per-lab runtime, REST and WebSocket isolation

Requirements: `API-001`–`API-013`, `WS-001`–`WS-004`, `SESSION-004`,
`SESSION-005`.

### Task 4.1 — RED

- [ ] Registry tests: concurrent Acquire creates one runtime; labs get distinct
  manager/random/update channel; Release/Close idempotent.
- [ ] `DeleteIfIdle` rejects live lease and serializes with Acquire.
- [ ] Middleware tests: cookie attributes/reuse/replacement, typed context and
  non-leaking 500.
- [ ] Two-cookie integration: action/events in A do not affect B despite matching
  entity IDs.
- [ ] Two-WebSocket integration: A action broadcasts only to A; B gets its own
  ticks. Direct no-cookie WS receives cookie and one lab.
- [ ] Main tests require TickAll, hourly cleanup, cookie-secure env and shutdown.
- [ ] Run:

```bash
go test -count=1 ./internal/labruntime ./internal/httpapi ./cmd/server
```

- [ ] Commit:

```bash
git add internal/labruntime internal/httpapi cmd/server
git commit -m "test(block-d-corrective): RED require isolated lab runtimes"
```

### Task 4.2 — GREEN

- [ ] Implement runtime factory with `Store.ForLab`, manager, per-lab random, Hub.
- [ ] Apply session middleware to `/api/*` and `/ws/lab` only.
- [ ] Resolve request manager from lease context, not global field.
- [ ] Hold WebSocket lease until disconnect.
- [ ] Startup owns Store + SessionService + Registry; 1s TickAll and 1h cleanup.
- [ ] Shutdown: HTTP → workers → hubs → DB.
- [ ] Run:

```bash
go test -count=1 ./internal/labruntime ./internal/httpapi ./internal/realtime ./cmd/server
go test -race -count=1 ./internal/labruntime ./internal/httpapi ./internal/realtime
```

- [ ] Commit:

```bash
git add internal/labruntime internal/httpapi internal/realtime cmd/server
git commit -m "feat(block-d-corrective): GREEN isolate REST and realtime by lab"
```

## 8. DC-5 — Transit DTO and safe frontend bootstrap

Requirements: `UI-014`, `API-001`, `API-002`, `WS-002`, `SESSION-001`.

### Task 5.1 — RED

- [ ] DTO tests: outbound, returning, none, distinct portals, exact deadline,
  sorted IDs and malformed transit.
- [ ] REST/Details/WS must serialize identical transit.
- [ ] Add exact TS type/builders.
- [ ] Prove WebSocket constructor is untouched until state promise resolves and
  never starts after bootstrap failure/abort.
- [ ] Prove reconnect keeps last snapshot.
- [ ] Run:

```bash
go test -count=1 ./internal/transport ./internal/httpapi ./internal/realtime
npm --prefix web run test -- --run src/state/SnapshotProvider.test.tsx src/api/realtime.test.ts
```

- [ ] Commit:

```bash
git add internal/transport internal/httpapi internal/realtime web/src/api web/src/state web/src/test
git commit -m "test(block-d-corrective): RED project observer transit safely"
```

### Task 5.2 — GREEN

- [ ] Implement one strict transit builder reused by state/details.
- [ ] Add TS types and selector by portal ID.
- [ ] Await accepted REST bootstrap before realtime start; preserve StrictMode
  cleanup.
- [ ] Do not expose lab/session/token in DTO/browser state.
- [ ] Run Go focused tests and `npm --prefix web run test`.
- [ ] Commit:

```bash
git add internal/transport internal/httpapi internal/realtime web/src/api web/src/state web/src/test
git commit -m "feat(block-d-corrective): GREEN expose authoritative observer transit"
```

## 9. DC-6 — Unified shell and fixed Dashboard

Requirements: `UI-001`, `UI-002`, `UI-008`–`UI-012`, `UI-014`.

### Task 6.1 — RED

- [ ] Require exact navigation order, Help, and AI Worklog on existing route.
- [ ] Require three magical connection labels and offline command block.
- [ ] Require 20 Observer pips/counts, transit identity/direction/time,
  UNSTABLE/Override states, seven equal slots and native-disabled empty actions.
- [ ] Occupied unavailable explains without POST; available action renders
  pending then success/error feedback.
- [ ] Desktop Playwright measures exact 4+3 centered rows, ≤1px size variance,
  no document/board scroll.
- [ ] Phone measures row counts 2/2/2/1, all 28 commands visible, no scroll.
- [ ] Run:

```bash
npm --prefix web run test -- --run src/app/App.test.tsx src/pages/DashboardPage.test.tsx src/components/feedback/ConnectionState.test.tsx src/features/portal-actions
npm --prefix web run test:e2e -- --project=desktop e2e/dashboard.spec.ts
```

- [ ] Commit:

```bash
git add web/src web/e2e/dashboard.spec.ts
git commit -m "test(block-d-corrective): RED specify unified dashboard experience"
```

### Task 6.2 — GREEN

- [ ] Replace tokens/global styles with approved obsidian/bronze/parchment system
  and one viewport ornament layer; no proprietary copied assets.
- [ ] Stats upper-left, nav/actions lower-left, no separate sidebar background.
- [ ] Render compact energy/exploration and 20 accessible Observer pips.
- [ ] Desktop placements: `1/3,3/5,5/7,7/9` then `2/4,4/6,6/8`; phone four rows.
- [ ] Use clamp/minmax so equal cards fit `100dvh` with no shell/main/board scroll.
- [ ] Show shared transit; retain four actions plus Details link.
- [ ] Empty actions native disabled; occupied unavailable remains explainable.
- [ ] Whole-card UNSTABLE and whole-shell Override.
- [ ] Add pressed/pending/success/error feedback without optimistic state.
- [ ] Run unit, both responsive E2E, typecheck and lint.
- [ ] Commit:

```bash
git add web/src web/e2e/dashboard.spec.ts
git commit -m "feat(block-d-corrective): GREEN rebuild shell and fixed dashboard"
```

## 10. DC-7 — Details, Event Log and Help

Requirements: `UI-003`–`UI-006`, `UI-008`, `UI-011`–`UI-014`,
`EVENT-010`, `EVENT-011`.

### Task 7.1 — RED

- [ ] Details requires center Portal, unframed Actions below, side facts, transit,
  terminal grayscale, no Refreshing, semantic Risk/Recommendation.
- [ ] History collapsed summary includes count; expansion scrolls only inner list.
- [ ] Event fixed UTC case formats `15:04:05-06-09-2026` then title then details.
- [ ] Help test requires every section from corrective design §8.
- [ ] Playwright proves Details document no-scroll desktop/phone and inner History
  overflow.
- [ ] Run:

```bash
npm --prefix web run test -- --run src/pages/PortalDetailsPage.test.tsx src/components/events src/pages/EventLogPage.test.tsx src/pages/HelpPage.test.tsx
npm --prefix web run test:e2e -- e2e/portal-details.spec.ts e2e/events.spec.ts
```

- [ ] Commit:

```bash
git add web/src web/e2e/portal-details.spec.ts web/e2e/events.spec.ts
git commit -m "test(block-d-corrective): RED specify details events and help"
```

### Task 7.2 — GREEN

- [ ] Recompose fixed Details header/body/footer; retain History.
- [ ] Pass status into PortalEffect; grayscale image/ring/sparks/status terminal.
- [ ] Risk/recommendation use icon/text plus color.
- [ ] Remove visible background-refresh marker, retain error recovery.
- [ ] Add deterministic timestamp formatter and original ISO `dateTime`.
- [ ] Help covers lore, interface, energies, lifecycle, risk, Observers/transit/
  LOST, research/creatures, four actions, Extraction, Override, Recommendations,
  Tutorial recap and glossary.
- [ ] Event Log scrolls only results list.
- [ ] Run full frontend tests/E2E/build.
- [ ] Commit:

```bash
git add web/src web/e2e/portal-details.spec.ts web/e2e/events.spec.ts
git commit -m "feat(block-d-corrective): GREEN compact details events and help"
```

## 11. DC-8 — Tutorial presentation and motion

Requirements: `TUTORIAL-001`–`TUTORIAL-020`, `UI-007`, `UI-009`,
`UI-010`, `UI-012`.

### Task 8.1 — RED

- [ ] Pure reducer tests: authoritative current, shown history, Back,
  Forward-to-current, no future skip, reset/live cleanup.
- [ ] Fake timer jump 1→3 shows completed Step 2 through 6999ms and advances at
  7000ms; normal 2→3 shows 3 immediately.
- [ ] Terminal tutorial target stays as presentation ghost 2000ms then replacement
  entrance; authoritative snapshot was accepted immediately.
- [ ] TutorialPanel: floating, all copy visible, no More Context, system/action/
  completion copy, navigation, target unobscured.
- [ ] PortalEffect: entrance/terminal classes, feedback pulse, reduced motion.
- [ ] Playwright uses same page session and verifies skipped-step timing.
- [ ] Run:

```bash
npm --prefix web run test -- --run src/features/tutorial src/portal-fx src/features/portal-actions
npm --prefix web run test:e2e -- --project=desktop e2e/tutorial.spec.ts
```

- [ ] Commit:

```bash
git add web/src/features/tutorial web/src/portal-fx web/src/features/portal-actions web/e2e/tutorial.spec.ts
git commit -m "test(block-d-corrective): RED specify tutorial presentation and motion"
```

### Task 8.2 — GREEN

- [ ] Implement pure presentation state and one timer hook; StrictMode cleanup and
  generation guards.
- [ ] Distinguish normal expected completion from missed intermediate render.
- [ ] Floating grimoire pointer events only on controls; position avoids target.
- [ ] Back browses shown cards; Forward cannot emit Tutorial signal/skip.
- [ ] Dashboard ghosts are keyed old/new portal+slot and never actionable.
- [ ] Add entrance sparks, terminal contraction/grayscale and command pulses.
- [ ] Reduced motion replaces loops with short opacity changes.
- [ ] Run full frontend tests and tutorial E2E.
- [ ] Commit:

```bash
git add web/src web/e2e/tutorial.spec.ts
git commit -m "feat(block-d-corrective): GREEN add tutorial presentation and portal motion"
```

## 12. DC-9 — Complete Plane artwork

Requirements: `UI-010`, `UI-013`, Final Spec §30.1.

### Task 9.1 — RED

- [ ] Add `plane_art_audit.json` with 85 seed-matching rows and fields
  `plane_id/local_path/source_kind/text_free/frame_free/reviewed`, recording
  current truth.
- [ ] Audit requires 85 unique IDs, existing hash-matching ≤180KiB decodable
  512×512 WebP, `fallback=false`, reviewed text/frame-free and local runtime path.
- [ ] Require hashes for 14/21/32/75 to differ from current bad assets.
- [ ] Run:

```bash
node web/scripts/audit-plane-art.mjs
npm --prefix web run test -- --run src/assets/plane-art.test.ts
```

Expected RED: current fallbacks/bad images fail.

- [ ] Commit tests/true audit metadata only:

```bash
git add data/plane_art_audit.json web/scripts/audit-plane-art.mjs web/src/assets/plane-art.test.ts
git commit -m "test(block-d-corrective): RED require complete text free plane art"
```

### Task 9.2 — GREEN

- [ ] For each failure, first search a permitted landscape/art crop without card
  text/frame and record URL/credit/policy.
- [ ] If absent, use approved image-generation workflow from canonical Plane
  description in coherent painterly style; record stable generated source/credit.
- [ ] Normalize with existing Sharp only; install nothing.
- [ ] Update source/runtime manifests, hashes, audit and credits together.
- [ ] Visually inspect changed assets and all-85 contact sheet; reject text, logos,
  card borders, unrelated art and empty placeholders.
- [ ] Run:

```bash
node web/scripts/audit-plane-art.mjs
npm --prefix web run assets:prepare -- --offline
npm --prefix web run test -- --run src/assets/plane-art.test.ts src/components/art/ArtCreditsDialog.test.tsx
```

Expected GREEN: 85/85 local, zero fallback, known four replaced.

- [ ] Commit:

```bash
git add data/plane_image_sources.json data/plane_images_manifest.json data/plane_art_audit.json web/public/planes web/scripts/audit-plane-art.mjs web/src/assets
git commit -m "feat(block-d-corrective): GREEN complete plane artwork catalog"
```

## 13. DC-10 — Integration and Block D closure

### Task 10.1 — Browser isolation

- [ ] Bind E2E reset/helpers to each browser context cookie: bootstrap page first,
  then use context-owned request/same-origin fetch.
- [ ] Two fresh contexts prove distinct cookies, isolated state/events/WS, reload
  continuation and cleared-cookie new game.
- [ ] Active-WS cleanup race uses a test-owned Go harness, never production route.
- [ ] Temporary-DB server restart proves same cookie resumes same lab.
- [ ] Commit failing integration evidence, repair only discovered integration gaps,
  then commit GREEN:

```bash
git commit -m "test(block-d-corrective): RED prove browser laboratory isolation"
git commit -m "fix(block-d-corrective): GREEN close session integration gaps"
```

### Task 10.2 — Full verification

- [ ] Format:

```bash
gofmt -w cmd internal
npm --prefix web run format
```

- [ ] Required backend suite:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
go test -race -count=1 ./...
```

Expected: empty `gofmt -l`; remaining exit 0.

- [ ] Frontend/assets/browser suite:

```bash
npm --prefix web run format:check
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
node web/scripts/audit-plane-art.mjs
npm --prefix web run assets:prepare -- --offline
npm --prefix web run test:e2e
```

Expected: all exit 0; desktop and phone pass.

- [ ] Hygiene:

```bash
git diff --check
git status --short
```

Expected: only deliberate docs pending; `omenpath.db` remains untouched/untracked.

### Task 10.3 — Traceability and STOP

- [ ] Move affected Observer/Tutorial/API/WS/UI/Persistence/Session rows to GREEN
  only with exact tests and implementation references.
- [ ] Append every RED/GREEN hash and verification result to worklog.
- [ ] Set roadmap Block D GREEN only if every corrective requirement passes.
- [ ] Keep Block E PLANNED.
- [ ] Verify requirements/traceability ID parity:

```bash
awk 'FNR==NR {if ($0 ~ /^\| [A-Z]+-[0-9][0-9][0-9] /) req[$2]=1; next} {if ($0 ~ /^\| [A-Z]+-[0-9][0-9][0-9] /) trace[$2]=1} END {for (id in req) if (!(id in trace)) print "missing traceability", id; for (id in trace) if (!(id in req)) print "missing requirement", id}' docs/requirements.md docs/traceability.md
```

- [ ] Commit:

```bash
git add 01_AI_WORKLOG_CURRENT.md 02_IMPLEMENTATION_ROADMAP_TDD.md docs/traceability.md
git commit -m "docs(block-d): close corrective gate evidence"
```

- [ ] Stop and report scope, files, hashes, verification, traceability and explicit
  confirmation that Stage 22 was not started.

## 14. Author self-review

- [x] Every approved complaint maps to requirement/checkpoint.
- [x] Multi-lab precedes frontend bootstrap/E2E changes.
- [x] REST and WS share cookie/runtime boundary.
- [x] Clean-start schema contains no legacy import/claim path.
- [x] Cleanup cannot race active/renewed lab deletion.
- [x] Dashboard/Details no-scroll have browser geometry evidence.
- [x] Tutorial timings exact: skipped 7s, replacement 2s.
- [x] 20 Observers covered by bootstrap, validation, reset, fresh schema and UI.
- [x] 85 images local, audited and credited.
- [x] No new dependencies.
- [x] No unrelated gameplay semantics change.
- [x] Block E remains unimplemented.
