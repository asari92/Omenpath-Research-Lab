# Omenpath Research Lab — Block D (Stages 15–21) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) or
> `superpowers:executing-plans` to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** построить полный React frontend поверх завершённых backend-контрактов:
realtime Dashboard, Portal Details, Extraction, Event Log, Tutorial UI и
AI Worklog, не перенося gameplay semantics в браузер.

**Architecture:** Go backend остаётся единственным владельцем состояния и
правил; frontend принимает целые authoritative snapshots через REST/WebSocket и
атомарно заменяет свой store. React pages собираются из typed API/features,
Portal animation изолирована в Canvas 2D subsystem с общим scheduler, а
локальные Plane artworks разрешаются по committed manifest без runtime network.

**Tech Stack:** Go 1.26, Node 20.19+, npm, React 19, TypeScript 6, Vite 8,
React Router 7, CSS Modules, Canvas 2D, Vitest 4, React Testing Library, MSW 2,
Playwright 1.58.2, ESLint 10, Prettier 3.

---

## 1. Основание, статус и порядок исполнения

Приоритет документов:

1. `00_FINAL_SPEC_v5.md` — единственный product/domain source of truth.
2. `01_AI_WORKLOG_CURRENT.md` — фактическая история и corrective passes.
3. `02_IMPLEMENTATION_ROADMAP_TDD.md` — порядок stages и архитектурные границы.
4. `docs/superpowers/specs/2026-09-05-block-d-frontend-design.md` —
   согласованный UX/technical design Block D.
5. Этот execution-plan.
6. `docs/requirements.md` и `docs/traceability.md`.

Verified starting boundary:

- Blocks A–C / Stages 0–14 — GREEN;
- Block C final audit — APPROVED;
- REST, WebSocket, Recommendation и Tutorial backend contracts существуют;
- frontend workspace отсутствует;
- Block E / Stages 22–27 — PLANNED, начинать нельзя.

Block D выполняется строго последовательно:

```text
Stage 15 Frontend foundation
  → Stage 16 Dashboard
  → Stage 17 Portal Details
  → Stage 18 Extraction / confirmations / errors
  → Stage 19 Event Log
  → Stage 20 Tutorial UI
  → Stage 21 AI Worklog UI
  → Block D full verification, traceability audit и STOP
```

После утверждения этого файла дополнительное подтверждение между Stages 15–21
не требуется. Stage 22 нельзя начинать даже при полностью зелёном Block D.

Для каждого meaningful checkpoint:

```text
requirements → failing tests → observed RED → RED commit
→ minimal implementation → focused GREEN → GREEN commit
```

RED обязан падать по ожидаемой функциональной причине, а не из-за syntax error,
сломанной конфигурации или отсутствующей установленной зависимости. Generated
artifacts (`package-lock.json`, asset manifest) разрешены в RED только когда они
нужны для запуска самого теста; production implementation в RED не попадает.

В конце каждой стадии:

```bash
npm --prefix web run format:check
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
```

`gofmt -l .` обязан вернуть пустой output. После команд обновляются
`docs/traceability.md` и `01_AI_WORKLOG_CURRENT.md` с реальными test results и
hashes. Race detector обязателен дополнительно на границе всего Block D.

## 2. Сквозные инварианты Block D

- Frontend не вычисляет Risk, Recommendation, command eligibility, Observer
  selection, costs, lifecycle deadlines или simulation tick.
- REST и WebSocket snapshots входят в один `acceptSnapshot`; старый
  `generated_at` не может перезаписать новый.
- Ни одна gameplay mutation не показывается оптимистично.
- Один command-in-flight блокирует только свой `portalID/action` key.
- `409 confirmable=true` повторяет ровно тот же endpoint с
  `{"confirm":true}`; cancel ничего не отправляет.
- Unavailable quick action не отправляет POST и не создаёт `ACTION_REJECTED`,
  кроме matching Tutorial Step 5 `ATTEMPT_CRITICAL_SEND`, где Final Spec требует
  настоящий rejected SEND для завершения шага.
- Confirmation-required остаётся available action: реальный первый POST обязан
  создать предусмотренный Final Spec `ACTION_REJECTED` и открыть modal.
- Dashboard никогда не показывает Risk, Recommendation, History, numeric risk
  score, decay rate, hidden instability timestamp или energy lifetime.
- Portal Details показывает Risk enum и Recommendation enum, но не hidden score.
- Slot positions определяются только `slot_index` и не сортируются.
- Все 7 Slot остаются одновременно monitorable на phone command-board; carousel,
  pagination и скрытые вкладки для Slots запрещены.
- Runtime не запрашивает Plane artwork у Scryfall/Wizards; только local WebP или
  local fallback.
- Canvas декоративен, не перекрывает клики и не заменяет semantic HTML.
- Tutorial progress меняют только commands/явные UI signals; GET и render —
  read-only.
- Event Log и Portal History форматируют один EventDTO contract и не меняют
  chronological order.
- React Strict Mode не должен удваивать WebSocket connection, Tutorial signal,
  command POST или Canvas scheduler registration.
- Existing Stages 0–14 semantics меняются только двумя доказанными additive
  transport passes из approved design: unavailable reasons и per-Plane Observer
  presence.

## 3. Зафиксированные публичные контракты

### 3.1 Quick-action reasons

Существующие booleans сохраняются. `QuickActionsDTO` получает nullable fields:

```go
type QuickActionsDTO struct {
	CanStabilize               bool    `json:"can_stabilize"`
	StabilizeUnavailableReason *string `json:"stabilize_unavailable_reason"`
	CanClose                   bool    `json:"can_close"`
	CloseUnavailableReason     *string `json:"close_unavailable_reason"`
	CanSend                    bool    `json:"can_send_observer"`
	SendUnavailableReason      *string `json:"send_observer_unavailable_reason"`
	CanRecall                  bool    `json:"can_recall_observer"`
	RecallUnavailableReason    *string `json:"recall_observer_unavailable_reason"`
}
```

Правило пары: `can_* == true` требует `reason == nil`; `can_* == false` требует
непустой public domain code. Для occupied terminal Portal причина
`PORTAL_NOT_OPEN`; Empty Slot формируется frontend и использует локальное
объяснение `No Portal occupies this Slot`.

Допустимые причины используют существующие HTTP error codes:

```text
PORTAL_NOT_OPEN
PORTAL_ALREADY_STABLE
PORTAL_OVERCHARGE_RISK
PORTAL_CRITICAL_RISK
PORTAL_DIRECTION_CONFLICT
PORTAL_BUSY
PORTAL_CREATURES_PRESENT
NO_AVAILABLE_OBSERVER
NO_WAITING_OBSERVER
INSUFFICIENT_LAB_ENERGY
EXTRACTION_SYNCHRONIZING
```

`CONFIRMATION_REQUIRED` не является unavailable reason. Domain validation
вычисляет eligibility как будто пользователь готов подтвердить предупреждение,
но без мутации, random draw или Event.

### 3.2 Plane presence

`PlaneDTO` additive fields:

```go
ObserversInPlane       int `json:"observers_in_plane"`
ObserversWaitingReturn int `json:"observers_waiting_return"`
```

`ObserversInPlane` считает EXPLORING, WAITING_RETURN и RETURNING с совпадающим
`CurrentPlaneID`; OUTBOUND ещё не находится в Plane, успешный return уже не
находится. Поля derived из того же resolved snapshot и не persist отдельно.

### 3.3 TypeScript transport boundary

`web/src/api/types.ts` вручную отражает публичный JSON contract и закрытые enums:

```ts
export type AppMode = 'TUTORIAL' | 'LIVE';
export type PortalStatus = 'OPEN' | 'CLOSED' | 'COLLAPSED';
export type PortalStability = 'STABLE' | 'UNSTABLE';
export type PortalFlow = 'NONE' | 'OUTBOUND' | 'INBOUND';
export type RiskLevel = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
export type Recommendation =
  | 'LEAVE OPEN'
  | 'STABILIZE'
  | 'RECALL OBSERVER'
  | 'WAIT FOR CORRIDOR'
  | 'SEND OBSERVER'
  | 'CLOSE';

export interface ActionAvailability {
  available: boolean;
  unavailableReason: string | null;
}

export interface StateSnapshot {
  generated_at: string;
  app: AppDTO;
  lab: LabDTO;
  exploration: ExplorationDTO;
  observers: ObserverCountsDTO;
  portals: PortalCountsDTO;
  needs_attention_portal_id: number | null;
  slots: SlotDTO[];
  planes: PlaneDTO[];
}
```

JSON snake_case остаётся на boundary; selectors могут собирать удобные view
objects, но не меняют смысл значений.

### 3.4 Snapshot acceptance

Единственная state mutation для server state:

```ts
export function shouldAcceptSnapshot(
  currentGeneratedAt: string | null,
  candidateGeneratedAt: string,
): boolean {
  const candidateTime = Date.parse(candidateGeneratedAt);
  if (Number.isNaN(candidateTime)) return false;
  if (currentGeneratedAt === null) return true;
  const currentTime = Date.parse(currentGeneratedAt);
  return !Number.isNaN(currentTime) && candidateTime >= currentTime;
}
```

В production сравнение выполняется по parsed milliseconds; равный timestamp
идемпотентно разрешён. Invalid timestamp становится protocol error и не меняет
store.

### 3.5 Routes and Tutorial navigation signals

```text
/                 Dashboard
/portals/:id      Portal Details
/events           Event Log
/ai-worklog       AI Worklog
```

Signal отправляется только если current snapshot ожидает его:

```ts
function matchingNavigationSignal(
  expected: TutorialExpectedAction | null,
  destination: { kind: 'portal'; id: number } | { kind: 'events' },
): TutorialSignalRequest | null;
```

Portal link требует `expected_action == OPEN_PORTAL_DETAILS` и matching
`tutorial_portal_id`. Event Log link требует
`expected_action == OPEN_EVENT_LOG`. Обычная навигация на других шагах signal
не отправляет.

## 4. Целевая карта файлов

```text
data/plane_image_sources.json               # one-time discovery/source choices
data/plane_images_manifest.json             # committed runtime/credit manifest
internal/domain/action_availability.go
internal/domain/action_availability_test.go
internal/transport/error_descriptor.go
internal/transport/dto.go
internal/transport/dto_test.go
internal/httpapi/command_handlers.go
internal/httpapi/command_handlers_test.go

web/.gitignore
web/package.json
web/package-lock.json
web/index.html
web/tsconfig.json
web/tsconfig.app.json
web/tsconfig.node.json
web/vite.config.ts
web/eslint.config.js
web/playwright.config.ts
web/scripts/prepare-plane-art.mjs
web/scripts/run-e2e-backend.sh
web/public/planes/*.webp
web/src/vite-env.d.ts
web/src/main.tsx
web/src/app/App.tsx
web/src/app/AppShell.tsx
web/src/app/AppShell.module.css
web/src/app/router.tsx
web/src/api/types.ts
web/src/api/errors.ts
web/src/api/client.ts
web/src/api/realtime.ts
web/src/state/snapshot-store.ts
web/src/state/SnapshotProvider.tsx
web/src/state/selectors.ts
web/src/components/feedback/ToastRegion.tsx
web/src/components/feedback/ConfirmDialog.tsx
web/src/components/feedback/ConnectionState.tsx
web/src/components/art/ArtCreditsDialog.tsx
web/src/components/events/EventList.tsx
web/src/components/events/EventRow.tsx
web/src/features/portal-actions/PortalActions.tsx
web/src/features/portal-actions/action-copy.ts
web/src/features/portal-actions/usePortalCommand.ts
web/src/features/extraction/ExtractionDialog.tsx
web/src/features/extraction/plane-filter.ts
web/src/features/tutorial/TutorialPanel.tsx
web/src/features/tutorial/tutorial-copy.ts
web/src/features/tutorial/navigation-signal.ts
web/src/portal-fx/particles.ts
web/src/portal-fx/scheduler.ts
web/src/portal-fx/PortalEffect.tsx
web/src/assets/plane-art.ts
web/src/styles/tokens.css
web/src/styles/global.css
web/src/pages/DashboardPage.tsx
web/src/pages/DashboardPage.module.css
web/src/pages/PortalDetailsPage.tsx
web/src/pages/PortalDetailsPage.module.css
web/src/pages/EventLogPage.tsx
web/src/pages/EventLogPage.module.css
web/src/pages/AIWorklogPage.tsx
web/src/pages/AIWorklogPage.module.css
web/src/pages/NotFoundPage.tsx
web/src/test/setup.ts
web/src/test/builders.ts
web/src/test/server.ts
web/src/test/FakeWebSocket.ts
web/src/**/*.test.ts
web/src/**/*.test.tsx
web/e2e/dashboard.spec.ts
web/e2e/portal-details.spec.ts
web/e2e/extraction.spec.ts
web/e2e/events.spec.ts
web/e2e/tutorial.spec.ts

docs/requirements.md
docs/traceability.md
01_AI_WORKLOG_CURRENT.md
02_IMPLEMENTATION_ROADMAP_TDD.md
```

Допускается разделить CSS или test builders дополнительно внутри указанной
responsibility. Нельзя создавать второй server-state store, второй Event model
или client-side domain package.

## 5. Dependency baseline

Stage 15 создаёт lockfile следующими явными project-local install commands.
Ничего не устанавливается глобально. npm cache направляется в `/tmp`, а
`PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1` запрещает повторную загрузку уже имеющегося
browser bundle:

```bash
PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 npm_config_cache=/tmp/omenpath-npm-cache \
  npm --prefix web install --save-exact \
  react@19.2.8 react-dom@19.2.8 react-router-dom@7.18.3 \
  react-markdown@10.1.0 remark-gfm@4.0.1

PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 npm_config_cache=/tmp/omenpath-npm-cache \
  npm --prefix web install --save-dev --save-exact \
  typescript@6.0.3 vite@8.2.2 @vitejs/plugin-react@6.1.1 \
  vitest@4.1.11 jsdom@29.1.1 \
  @testing-library/react@16.3.3 @testing-library/user-event@14.6.7 \
  @testing-library/jest-dom@6.10.0 \
  msw@2.15.0 @playwright/test@1.58.2 sharp@0.35.4 \
  eslint@10.10.0 typescript-eslint@8.69.0 \
  eslint-plugin-react-hooks@7.1.1 eslint-plugin-react-refresh@0.5.6 \
  prettier@3.9.6 globals@17.12.0 \
  @types/react@19.2.18 @types/react-dom@19.2.7 \
  @types/node@20
```

Этот baseline проверен для repository Node `v20.19.3`: Vite 8 требует минимум
20.19, Vitest 4 поддерживает Node 20, jsdom 29 поддерживает 20.19, а
typescript-eslint 8 принимает TypeScript `<6.1`, поэтому TypeScript 7 намеренно
не используется. Playwright `1.58.2` намеренно совпадает с уже установленными
в `/home/asari/.cache/ms-playwright` Chromium/headless-shell revision `1208` и
FFmpeg revision `1011`; `playwright install` в Block D запускать нельзя. Если
этот cache перестанет быть доступен, выполнение останавливается для решения с
пользователем, а не начинает новый download автоматически.

## 6. Stage 15 — Frontend foundation

### Checkpoint 15A — workspace, test harness и route shell

**Files:**

- Create: `web/package.json`, `web/package-lock.json`, `web/.gitignore`
- Create: `web/index.html`, `web/tsconfig*.json`, `web/vite.config.ts`
- Create: `web/eslint.config.js`, `web/playwright.config.ts`
- Create: `web/src/test/setup.ts`, `web/src/test/builders.ts`
- Create: `web/src/app/App.test.tsx`
- Create in GREEN: `web/src/main.tsx`, `web/src/app/App.tsx`,
  `web/src/app/AppShell.tsx`, `web/src/app/router.tsx`, base styles/pages

- [ ] **Step 1: create deterministic frontend tooling.**

`web/package.json` scripts must be exactly addressable from repository root:

```json
{
  "name": "omenpath-research-lab-web",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "typecheck": "tsc -b --pretty false",
    "lint": "eslint . --max-warnings=0",
    "format": "prettier --write .",
    "format:check": "prettier --check .",
    "test": "vitest run",
    "test:watch": "vitest",
    "test:e2e": "playwright test",
    "assets:prepare": "node scripts/prepare-plane-art.mjs"
  }
}
```

`web/.gitignore` contains only frontend outputs:

```gitignore
node_modules/
dist/
coverage/
playwright-report/
test-results/
```

`vite.config.ts` configures React, jsdom tests with `src/test/setup.ts`, CSS
Modules and dev proxies `/api` and `/ws` to
`VITE_BACKEND_TARGET || http://127.0.0.1:8080` with WebSocket enabled for `/ws`.
`setup.ts` imports `@testing-library/jest-dom/vitest` and performs DOM cleanup
after every test.

`playwright.config.ts` sets `fullyParallel:false`, `workers:1`, Chromium desktop
and phone projects, and the two web servers described in Checkpoint 16B. Every
spec resets Tutorial through the real API before its first assertion, so tests do
not depend on execution order despite sharing one temporary server process.

- [ ] **Step 2: install the pinned dependency baseline from §5 and generate
  `package-lock.json`.**

- [ ] **Step 3: add the first failing route-shell tests.**

```tsx
it('renders the shared shell and dashboard at /', async () => {
  render(<App initialEntries={['/']} />);
  expect(await screen.findByRole('banner')).toBeInTheDocument();
  expect(screen.getByRole('heading', { name: /laboratory overview/i })).toBeInTheDocument();
});

it.each([
  ['/events', /event log/i],
  ['/ai-worklog', /ai worklog/i],
  ['/unknown', /not found/i],
])('maps %s to its page boundary', async (path, heading) => {
  render(<App initialEntries={[path]} />);
  expect(await screen.findByRole('heading', { name: heading })).toBeInTheDocument();
});
```

- [ ] **Step 4: run the focused test and observe RED.**

```bash
npm --prefix web run test -- src/app/App.test.tsx
```

Expected: FAIL because `App`, router and pages do not exist; test runner itself
must start successfully.

- [ ] **Step 5: commit RED.**

```bash
git add web/package.json web/package-lock.json web/.gitignore web/index.html \
  web/tsconfig.json web/tsconfig.app.json web/tsconfig.node.json \
  web/vite.config.ts web/eslint.config.js web/playwright.config.ts \
  web/src/test web/src/app/App.test.tsx
git commit -m "test(stage15): RED frontend workspace and route shell"
```

- [ ] **Step 6: implement the minimal route shell.**

`App` accepts test-only initial entries without branching production behavior:

```tsx
export interface AppProps {
  initialEntries?: string[];
}

export function App({ initialEntries }: AppProps) {
  const router = initialEntries
    ? createMemoryRouter(routes, { initialEntries })
    : createBrowserRouter(routes);
  return <RouterProvider router={router} />;
}
```

`AppShell` supplies semantic `header`, `nav`, `main`, toast and modal hosts.
Placeholder pages contain only their heading and no Stage 16–21 functionality.

- [ ] **Step 7: run focused GREEN and static gates.**

```bash
npm --prefix web run test -- src/app/App.test.tsx
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run build
```

Expected: all PASS; Vite produces `web/dist` and the route test has no act or
Strict Mode warnings.

- [ ] **Step 8: commit GREEN.**

```bash
git add web
git commit -m "feat(stage15): GREEN frontend workspace and route shell"
```

### Checkpoint 15B — typed REST/WebSocket clients и authoritative store

**Files:**

- Create: `web/src/api/types.ts`, `web/src/api/errors.ts`
- Create: `web/src/api/client.ts`, `web/src/api/client.test.ts`
- Create: `web/src/api/realtime.ts`, `web/src/api/realtime.test.ts`
- Create: `web/src/state/snapshot-store.ts`,
  `web/src/state/snapshot-store.test.ts`
- Create: `web/src/state/SnapshotProvider.tsx`, `web/src/state/selectors.ts`
- Create: `web/src/test/server.ts`, `web/src/test/FakeWebSocket.ts`
- Modify in GREEN: `web/src/app/App.tsx`

- [ ] **Step 1: define the complete TypeScript JSON types from current
  `internal/transport/dto.go`.**

No `any` is allowed at the transport boundary. `ApiError` is:

```ts
export interface ApiErrorBody {
  error: { code: string; message: string; confirmable: boolean };
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    readonly confirmable: boolean,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}
```

- [ ] **Step 2: write failing client/store tests.**

Required test names:

```text
client: GET state/details/events parses success and structured errors
client: each portal command uses exact endpoint and confirm body
client: extraction/tutorial/signal/live use exact request shapes
store: bootstrap snapshot is accepted atomically
store: older REST response cannot replace newer WebSocket snapshot
store: equal generated_at is idempotently accepted
store: invalid generated_at becomes protocol error
store: one command key does not block another command
realtime: one initial snapshot reaches the shared store
realtime: disconnect exposes status and bounded 1/2/5/10 second backoff
realtime: reconnect accepts latest initial snapshot
realtime: Strict Mode mount/unmount leaves one live socket
```

Representative ordering proof:

```ts
store.acceptSnapshot(snapshotAt('2026-09-05T10:00:02Z', 72));
store.acceptSnapshot(snapshotAt('2026-09-05T10:00:01Z', 10));
expect(store.getState().snapshot?.lab.current_energy).toBe(72);
```

- [ ] **Step 3: run and observe RED.**

```bash
npm --prefix web run test -- src/api src/state
```

Expected: FAIL on missing client/store/realtime implementations, not test setup.

- [ ] **Step 4: commit RED.**

```bash
git add web/src/api web/src/state web/src/test
git commit -m "test(stage15): RED authoritative clients and snapshot store"
```

- [ ] **Step 5: implement minimal clients and store.**

Required public shapes:

```ts
export interface OmenpathApi {
  state(signal?: AbortSignal): Promise<StateSnapshot>;
  portal(id: number, signal?: AbortSignal): Promise<PortalDetails>;
  events(signal?: AbortSignal): Promise<EventDTO[]>;
  stabilize(id: number): Promise<StateSnapshot>;
  close(id: number, confirm?: boolean): Promise<StateSnapshot>;
  sendObserver(id: number, confirm?: boolean): Promise<StateSnapshot>;
  recallObserver(id: number, confirm?: boolean): Promise<StateSnapshot>;
  openExtraction(planeId: number): Promise<StateSnapshot>;
  startTutorial(): Promise<StateSnapshot>;
  resetTutorial(): Promise<StateSnapshot>;
  tutorialSignal(request: TutorialSignalRequest): Promise<StateSnapshot>;
  startLive(): Promise<StateSnapshot>;
}

export interface SnapshotStoreState {
  snapshot: StateSnapshot | null;
  bootstrap: 'idle' | 'loading' | 'ready' | 'failed';
  connection: 'connecting' | 'connected' | 'reconnecting' | 'offline';
  commandKeys: ReadonlySet<string>;
  protocolError: string | null;
}
```

WebSocket URL is derived from `window.location`: `http→ws`, `https→wss`, path
`/ws/lab`. Reconnect delays are exactly `[1000, 2000, 5000, 10000]` ms and cap
at 10 seconds. Successful message resets the attempt counter. Cleanup cancels
socket, timers and pending bootstrap fetch.

- [ ] **Step 6: run focused GREEN.**

```bash
npm --prefix web run test -- src/api src/state
```

Expected: PASS with fake timers fully drained and no real network.

- [ ] **Step 7: commit GREEN.**

```bash
git add web/src/api web/src/state web/src/test web/src/app/App.tsx
git commit -m "feat(stage15): GREEN authoritative clients and snapshot store"
```

### Checkpoint 15C — backend unavailable-reason corrective pass

**Files:**

- Create: `internal/domain/action_availability.go`
- Create: `internal/domain/action_availability_test.go`
- Create: `internal/transport/error_descriptor.go`
- Modify: `internal/transport/dto.go`, `internal/transport/dto_test.go`
- Modify: `internal/httpapi/command_handlers.go`
- Modify: `internal/httpapi/command_handlers_test.go`
- Modify: `web/src/api/types.ts`
- Create: `web/src/features/portal-actions/action-copy.ts`,
  `action-copy.test.ts`

- [ ] **Step 1: write backend characterization/failing tests before the
  refactor.**

Table cases must cover all reasons listed in §3.1, active Override zero-cost,
EXTRACTION before/after synchronization, flow conflict, busy transit, creatures,
no observer and confirmable UNSTABLE Send/Recall/Close.

Core equivalence assertion:

```go
availability, err := domain.PortalActionAvailability(state, portal.ID, action, now, cfg)
require.NoError(t, err)
if availability.Available {
	require.NoError(t, runOnCloneWithConfirmTrue(state, portal.ID, action, now, cfg))
} else {
	require.ErrorIs(t, runOnCloneWithConfirmTrue(state, portal.ID, action, now, cfg), availability.Cause)
}
```

Also assert:

```text
BuildStateSnapshot emits boolean/reason pairs for every occupied Slot
BuildPortalDetails emits the identical pairs
availability read does not mutate state or consume Random
GET /api/state with unavailable actions creates no ACTION_REJECTED
HTTP domain error JSON remains byte-compatible after descriptor extraction
frontend copy maps every known reason and safely renders an unknown code
```

- [ ] **Step 2: run and observe RED.**

```bash
go test -count=1 ./internal/domain ./internal/transport ./internal/httpapi \
  -run 'TestPortalActionAvailability|TestQuickActionReasons|TestDomainConflictMapping'
```

Expected: compile/test FAIL because availability API and reason fields do not
exist.

- [ ] **Step 3: commit RED.**

```bash
git add internal/domain/action_availability_test.go internal/transport/dto_test.go \
  internal/httpapi/command_handlers_test.go \
  web/src/features/portal-actions/action-copy.test.ts
git commit -m "test(stage15): RED authoritative unavailable action reasons"
```

- [ ] **Step 4: extract pure validation without changing command semantics.**

Required domain contract:

```go
type PortalAction string

const (
	PortalActionStabilize PortalAction = "STABILIZE"
	PortalActionClose     PortalAction = "CLOSE"
	PortalActionSend      PortalAction = "SEND"
	PortalActionRecall    PortalAction = "RECALL"
)

type ActionAvailability struct {
	Available bool
	Cause     error
}

func PortalActionAvailability(
	state SimulationState,
	portalID int64,
	action PortalAction,
	now time.Time,
	cfg config.Config,
) (ActionAvailability, error)
```

Command functions and availability call the same extracted validation helpers.
Availability evaluates confirmation-sensitive commands with confirmation granted,
never reaches mutation/random, and returns invariant failures as `error` rather
than an unavailable public cause.

Move the known-domain descriptor into `internal/transport/error_descriptor.go`:

```go
type ErrorDescriptor struct {
	Code        string
	Message     string
	Confirmable bool
}

func DescribeDomainError(err error) (ErrorDescriptor, bool)
```

Both HTTP error mapping and DTO reason mapping use this single descriptor.

- [ ] **Step 5: implement additive DTO fields and frontend copy map.**

Unknown future codes render `Action unavailable: <code>`; known codes have
specific English messages. No frontend predicate may choose a reason.

- [ ] **Step 6: run focused and regression GREEN.**

```bash
gofmt -w internal/domain/action_availability.go \
  internal/domain/action_availability_test.go \
  internal/transport/error_descriptor.go internal/transport/dto.go \
  internal/transport/dto_test.go internal/httpapi/command_handlers.go \
  internal/httpapi/command_handlers_test.go
go test -count=1 ./internal/domain ./internal/transport ./internal/httpapi
npm --prefix web run test -- src/api src/features/portal-actions
```

Expected: PASS and existing command/error tests unchanged.

- [ ] **Step 7: commit GREEN.**

```bash
git add internal/domain internal/transport internal/httpapi web/src/api \
  web/src/features/portal-actions
git commit -m "feat(stage15): GREEN authoritative unavailable action reasons"
```

### Checkpoint 15D — local artwork pipeline и Canvas boundary

**Files:**

- Create: `data/plane_image_sources.json`, `data/plane_images_manifest.json`
- Create: `web/scripts/prepare-plane-art.mjs`
- Create: `web/src/assets/plane-art.ts`, `web/src/assets/plane-art.test.ts`
- Create: `web/src/components/art/ArtCreditsDialog.tsx`, test and CSS
- Create: `web/public/planes/*.webp`
- Create: `web/src/portal-fx/particles.ts`, `particles.test.ts`
- Create: `web/src/portal-fx/scheduler.ts`, `scheduler.test.ts`
- Create: `web/src/portal-fx/PortalEffect.tsx`, `PortalEffect.test.tsx`

- [ ] **Step 1: write asset and particle tests first.**

Required asset assertions:

```text
manifest IDs exactly equal all 85 seed IDs
every manifest local_path is unique and resolves below web/public/planes
every file exists, is WebP, is at most 512×512 and at most 180 KiB
every entry contains source_url, credit, policy_url, sha256 and fallback boolean
runtime resolver never returns an http/https URL
unknown Plane ID returns the committed generic fallback
Art Credits renders source, artist/credit and policy links as safe anchors
```

Required pure particle assertions use injected seeded random:

```ts
const particle = spawnParticle({ center: { x: 88, y: 88 }, radius: 65, random });
expect(distance(particle.position, { x: 88, y: 88 })).toBeCloseTo(65, 5);
expect(dot(radialUnit(particle), particle.velocity)).toBeGreaterThan(0);
expect(Math.abs(cross(radialUnit(particle), particle.velocity))).toBeGreaterThan(0);
```

Also test high cap 170, low cap 42, one scheduler RAF, unregistration,
`visibilitychange`, IntersectionObserver pause and reduced motion.

- [ ] **Step 2: run and observe RED.**

```bash
npm --prefix web run test -- src/assets src/portal-fx src/components/art
```

Expected: FAIL because manifest, resolver and Canvas modules are absent.

- [ ] **Step 3: commit RED.**

```bash
git add web/src/assets web/src/portal-fx
git commit -m "test(stage15): RED local plane art and portal particle boundary"
```

- [ ] **Step 4: implement the one-time asset builder.**

`prepare-plane-art.mjs`:

1. reads `data/mtg_planes_expanded_seed.json`;
2. in `--discover` mode fetches Scryfall Plane cards with the required
   User-Agent, at least 100 ms between requests, and follows pagination;
3. normalizes Plane names/aliases and the subtype after `Plane —`;
4. chooses only exact normalized subtype matches, deterministically by Scryfall
   ID; unmatched Planes receive generated local fallback art;
5. uses `sharp` to cover-crop 512×512 WebP at quality 78;
6. writes source choices, manifest and sha256; no timestamp is written, so a
   repeated build is byte-stable;
7. rejects redirects or source MIME types that are not images and never embeds
   API credentials.

Run once with network:

```bash
npm --prefix web run assets:prepare -- --discover
```

Review `credit`, `source_url` and `fallback` fields, then run offline rebuild:

```bash
npm --prefix web run assets:prepare -- --offline
```

Expected: the second run makes no Git diff.

- [ ] **Step 5: implement the Canvas subsystem.**

Public API:

```ts
export type PortalEffectDensity = 'high' | 'low' | 'static';

export interface ParticleProfile {
  maxParticles: number;
  spawnPerSecond: number;
  tangentialSpeed: readonly [number, number];
  outwardSpeed: readonly [number, number];
}

export interface PortalEffectProps {
  portalId: number;
  planeId: number;
  planeName: string;
  density: PortalEffectDensity;
}
```

Color is a pure HSL hash of `portalId`. The Plane image remains an HTML element
below the Canvas; particles spawn on the circumference and move outside, with no
concentric/ripple texture inside. Canvas has `aria-hidden` and
`pointer-events:none`.

Implement `ArtCreditsDialog` from manifest metadata. External credit/policy
anchors use `target="_blank"` with `rel="noreferrer"`; source strings are never
inserted as HTML. The App Shell entry opens this dialog without changing route.

- [ ] **Step 6: run focused GREEN and asset verification.**

```bash
npm --prefix web run test -- src/assets src/portal-fx src/components/art
npm --prefix web run typecheck
npm --prefix web run build
```

Expected: PASS, no remote URL is used as an image/fetch target at runtime (credit
links in the manifest are allowed), and all 85 IDs resolve locally.

- [ ] **Step 7: commit GREEN.**

```bash
git add data/plane_image_sources.json data/plane_images_manifest.json \
  web/scripts web/public/planes web/src/assets web/src/portal-fx \
  web/src/components/art web/src/app/AppShell.tsx web/package.json web/package-lock.json
git commit -m "feat(stage15): GREEN local plane art and shared portal particles"
```

### Stage 15 closure

- [ ] Run the full per-stage commands from §1.
- [ ] Update traceability notes for UI-001..007 to identify the GREEN foundation
  subsets but retain PARTIAL/PLANNED status until their rendering stages.
- [ ] Record dependency versions, asset provenance counts, RED/GREEN hashes and
  verification output in `01_AI_WORKLOG_CURRENT.md`.
- [ ] Commit documentation:

```bash
git add docs/traceability.md 01_AI_WORKLOG_CURRENT.md
git commit -m "docs(stage15): close frontend foundation evidence"
```

Stage 15 не реализует Dashboard content, Details diagnostics, Extraction flow,
Event Log, Tutorial guidance или AI Worklog page body.

## 7. Stage 16 — Dashboard

### Checkpoint 16A — summary, Needs Attention и seven-slot rendering

**Files:**

- Create: `web/src/pages/DashboardPage.test.tsx`
- Create: `web/src/components/dashboard/LabSummary.tsx`
- Create: `web/src/components/dashboard/NeedsAttention.tsx`
- Create: `web/src/components/dashboard/PortalSlot.tsx`
- Create: `web/src/components/dashboard/EmptyPortalSlot.tsx`
- Create: corresponding CSS Modules
- Modify: `web/src/pages/DashboardPage.tsx`, `DashboardPage.module.css`
- Modify: `web/src/state/selectors.ts`

- [ ] **Step 1: write failing Dashboard rendering tests.**

Required test cases:

```text
renders every Final Spec summary value from one snapshot
renders exactly Slot 1..7 in slot_index order despite shuffled input
never renders Risk, Recommendation or History in a Slot
formats Portal Energy with one decimal and time as mm:ss
Needs Attention links to the matching Portal without moving it
Leyline Override banner shows active countdown deadline, not a client mutation
zero OPEN Portals keeps all seven Empty Slots and waiting message
occupied-to-empty snapshot preserves Slot geometry/control positions
```

The hidden-data guard inspects rendered text and component props:

```tsx
renderDashboard(snapshotBuilder().withRiskOnlyInDetails().build());
expect(screen.queryByText(/risk/i)).not.toBeInTheDocument();
expect(screen.queryByText(/recommendation/i)).not.toBeInTheDocument();
expect(screen.getAllByTestId('portal-slot')).toHaveLength(7);
```

- [ ] **Step 2: run and observe RED.**

```bash
npm --prefix web run test -- src/pages/DashboardPage.test.tsx
```

Expected: FAIL on absent summary/Slot components.

- [ ] **Step 3: commit RED.**

```bash
git add web/src/pages/DashboardPage.test.tsx web/src/test/builders.ts
git commit -m "test(stage16): RED dashboard summary and fixed portal slots"
```

- [ ] **Step 4: implement pure selectors and components.**

The selector rejects malformed Slot contracts instead of sorting silently:

```ts
export function sevenSlots(snapshot: StateSnapshot): readonly SlotDTO[] {
  const byIndex = new Map(snapshot.slots.map((slot) => [slot.slot_index, slot]));
  const result = Array.from({ length: 7 }, (_, i) => byIndex.get(i + 1));
  if (result.some((slot) => slot === undefined) || byIndex.size !== 7) {
    throw new Error('State snapshot must contain exactly Slot 1..7');
  }
  return result as readonly SlotDTO[];
}
```

Energy display uses `value.toFixed(1)`. Time clamps negative transport values to
`00:00` for presentation only; it never predicts a lifecycle transition.

- [ ] **Step 5: run focused GREEN.**

```bash
npm --prefix web run test -- src/pages/DashboardPage.test.tsx
```

Expected: PASS; snapshots with invalid Slot identities show the shared protocol
error state rather than a partial board.

- [ ] **Step 6: commit GREEN.**

```bash
git add web/src/pages web/src/components/dashboard web/src/state
git commit -m "feat(stage16): GREEN dashboard summary and fixed portal slots"
```

### Checkpoint 16B — quick actions, stable geometry и phone command-board

**Files:**

- Create: `web/src/features/portal-actions/PortalActions.test.tsx`
- Create: `web/src/features/portal-actions/PortalActions.tsx`
- Create: `web/src/features/portal-actions/usePortalCommand.ts`
- Create: `web/src/components/feedback/ToastRegion.tsx`
- Modify: `web/src/components/dashboard/PortalSlot.tsx` and CSS
- Modify: `web/src/pages/DashboardPage.module.css`
- Create: `web/e2e/dashboard.spec.ts`

- [ ] **Step 1: write failing interaction and layout tests.**

Required component tests:

```text
each occupied and Empty Slot renders four command buttons
DETAILS is a link and is not counted as a gameplay command
available action invokes only its exact API method
unavailable action is focusable, has aria-disabled=true, shows mapped reason
unavailable action sends no POST and creates no optimistic snapshot
busy state affects only the selected portal/action key
new snapshot switches availability without remounting the Slot position
PortalEffect receives high density only for Needs Attention, low otherwise
```

Representative unavailable proof:

```tsx
const user = userEvent.setup();
renderSlot(withUnavailableSend('PORTAL_CRITICAL_RISK'), api);
const send = screen.getByRole('button', { name: /send observer/i });
expect(send).toHaveAttribute('aria-disabled', 'true');
await user.click(send);
expect(await screen.findByRole('status')).toHaveTextContent(/critical/i);
expect(api.sendObserver).not.toHaveBeenCalled();
```

Playwright phone case uses viewport `{ width: 390, height: 760 }` and asserts:

```ts
const board = page.getByTestId('portal-board');
await expect(page.getByTestId('portal-slot')).toHaveCount(7);
const fits = await board.evaluate((node) => node.scrollHeight <= node.clientHeight);
expect(fits).toBe(true);
const pageFits = await page.evaluate(() => document.documentElement.scrollHeight <= innerHeight);
expect(pageFits).toBe(true);
```

- [ ] **Step 2: run unit tests and observe RED.**

```bash
npm --prefix web run test -- src/features/portal-actions src/pages/DashboardPage.test.tsx
```

Expected: FAIL on missing actions/toasts/layout behavior.

- [ ] **Step 3: commit RED.**

```bash
git add web/src/features/portal-actions web/src/pages/DashboardPage.test.tsx \
  web/e2e/dashboard.spec.ts
git commit -m "test(stage16): RED dashboard commands and responsive board"
```

- [ ] **Step 4: implement the four-command control.**

`PortalActions` accepts availability directly from DTO and one callback per
command. It renders `STABILIZE`, `CLOSE`, `SEND OBSERVER`, `RECALL OBSERVER` in
stable DOM order. `aria-disabled`, not native `disabled`, is used for unavailable
commands; an actual in-flight command uses native `disabled` to prevent duplicate
submission.

The Dashboard root uses `height:100dvh` on phone. After shell, two-row compact
summary and Needs Attention, `portal-board` gets remaining height with:

```css
grid-template-columns: repeat(2, minmax(0, 1fr));
grid-template-rows: repeat(4, minmax(0, 1fr));
overflow: hidden;
```

Slot 7 spans two columns but keeps one-column width and is centered. Portal
diameter/buttons/type use `clamp()` against both viewport width and height.

- [ ] **Step 5: configure and run real browser verification.**

`web/scripts/run-e2e-backend.sh` uses a `mktemp -d` SQLite directory, exports
`OMENPATH_ADDR=127.0.0.1:18080`, runs `go run ./cmd/server` from repository root
and cleans only that exact temporary directory on exit. `playwright.config.ts`
starts it plus Vite on `127.0.0.1:4173`.

```bash
test -x /home/asari/.cache/ms-playwright/chromium-1208/chrome-linux64/chrome
test -x /home/asari/.cache/ms-playwright/chromium_headless_shell-1208/chrome-headless-shell-linux64/chrome-headless-shell
npm --prefix web exec -- playwright --version
npm --prefix web run test:e2e -- dashboard.spec.ts
```

Expected: desktop 4+3 and phone 2×4 tests PASS; all seven Slots fit at the phone
baseline without page/board vertical scroll.

- [ ] **Step 6: run focused/full frontend GREEN and commit.**

```bash
npm --prefix web run test -- src/features/portal-actions src/pages/DashboardPage.test.tsx
npm --prefix web run test:e2e -- dashboard.spec.ts
git add web
git commit -m "feat(stage16): GREEN responsive dashboard and quick actions"
```

### Stage 16 closure

- [ ] Run the full per-stage commands from §1.
- [ ] Split the current aggregate `UI-001..006` traceability row into the six
  individual requirement rows from `docs/requirements.md`; set UI-001, UI-002,
  UI-006 and UI-007 to GREEN with exact frontend tests, and leave UI-003..005
  PLANNED.
- [ ] Append real Stage 16 RED/GREEN hashes, mobile viewport result and quality
  output to the worklog.
- [ ] Commit:

```bash
git add docs/traceability.md 01_AI_WORKLOG_CURRENT.md
git commit -m "docs(stage16): close dashboard traceability and evidence"
```

Stage 16 не отображает per-Portal Risk/Recommendation/History и не реализует
Extraction chooser.

## 8. Stage 17 — Portal Details

### Checkpoint 17A — live Details resource и required sections

**Files:**

- Create: `web/src/api/live-resource.ts`, `live-resource.test.ts`
- Create: `web/src/pages/PortalDetailsPage.test.tsx`
- Create: `web/src/components/portal/PortalFacts.tsx`
- Create: `web/src/components/portal/DestinationFacts.tsx`
- Create: `web/src/components/portal/Diagnostics.tsx`
- Modify: `web/src/pages/PortalDetailsPage.tsx` and CSS

- [ ] **Step 1: write failing resource and rendering tests.**

Required cases:

```text
route id must be a positive integer before API call
Details renders every Portal and Destination field from Final Spec §25
open Portal renders Risk enum and Recommendation enum
terminal Portal renders null Risk/Recommendation as Not applicable
How Risk Works explains bands without numeric score/hidden timestamp/decay
accepted snapshot coalesces a Details refresh
one in-flight GET allows exactly one trailing refresh
unmount aborts current GET and leaves no trailing work
404 renders Portal Not Found; 500/network uses retry state
```

Coalescing test must prove three snapshot edges during one GET produce two total
GETs, never four.

- [ ] **Step 2: run and observe RED.**

```bash
npm --prefix web run test -- src/api/live-resource.test.ts \
  src/pages/PortalDetailsPage.test.tsx
```

Expected: FAIL because live resource and section components are absent.

- [ ] **Step 3: commit RED.**

```bash
git add web/src/api/live-resource.test.ts web/src/pages/PortalDetailsPage.test.tsx
git commit -m "test(stage17): RED live portal details and diagnostics"
```

- [ ] **Step 4: implement the coalesced resource and page.**

Resource contract:

```ts
export interface LiveResource<T> {
  getSnapshot(): { data: T | null; loading: boolean; error: Error | null };
  subscribe(listener: () => void): () => void;
  refresh(): void;
  dispose(): void;
}
```

Page uses `PortalEffect` high density for the current Portal, displays Energy
with one decimal, time `mm:ss`, destination explored status and counts. `How Risk
Works` is a disclosure containing LOW/MEDIUM/HIGH/CRITICAL meanings and states
that Recommendation is guidance, not a restriction.

- [ ] **Step 5: run focused GREEN and commit.**

```bash
npm --prefix web run test -- src/api/live-resource.test.ts \
  src/pages/PortalDetailsPage.test.tsx
git add web/src/api web/src/pages web/src/components/portal
git commit -m "feat(stage17): GREEN live portal details and diagnostics"
```

### Checkpoint 17B — shared actions, history и explicit Details signal

**Files:**

- Create: `web/src/components/events/EventRow.tsx`, `EventList.tsx`
- Create: `web/src/components/events/EventList.test.tsx`
- Create: `web/src/features/tutorial/navigation-signal.ts`
- Create: `web/src/features/tutorial/navigation-signal.test.ts`
- Modify: `web/src/pages/PortalDetailsPage.tsx`, tests
- Create: `web/e2e/portal-details.spec.ts`

- [ ] **Step 1: write failing tests.**

Required cases:

```text
Portal History renders only API-provided rows in received chronological order
payload_json is collapsed by default and valid object is pretty-printed on open
Details reuses exactly the same four-command component as Dashboard
target DETAILS navigation emits one matching signal
wrong step, wrong Portal ID and direct route load emit no signal
React Strict Mode does not duplicate the signal
successful command snapshot updates shared store and refreshes Details once
```

Signal matcher truth table:

```ts
expect(matchingNavigationSignal(expectedDetails(42), { kind: 'portal', id: 42 }))
  .toEqual({ signal: 'PORTAL_DETAILS_OPENED', portal_id: 42 });
expect(matchingNavigationSignal(expectedDetails(42), { kind: 'portal', id: 43 }))
  .toBeNull();
```

- [ ] **Step 2: run and observe RED, then commit.**

```bash
npm --prefix web run test -- src/components/events \
  src/features/tutorial/navigation-signal.test.ts src/pages/PortalDetailsPage.test.tsx
git add web/src/components/events web/src/features/tutorial \
  web/src/pages/PortalDetailsPage.test.tsx web/e2e/portal-details.spec.ts
git commit -m "test(stage17): RED portal history actions and details signal"
```

- [ ] **Step 3: implement Event primitives and explicit navigation signal.**

The click that navigates from a matching target Slot carries an in-memory
navigation intent. After route render, one effect consumes that intent and calls
`tutorialSignal`. Reload/bookmark without intent never signals. API errors show a
toast but do not block Details navigation.

- [ ] **Step 4: run focused and browser GREEN.**

```bash
npm --prefix web run test -- src/components/events src/features/tutorial \
  src/pages/PortalDetailsPage.test.tsx
npm --prefix web run test:e2e -- portal-details.spec.ts
```

Expected: Tutorial Step 1 advances only after clicking matching `DETAILS`; GET
requests alone do not advance it.

- [ ] **Step 5: commit GREEN.**

```bash
git add web
git commit -m "feat(stage17): GREEN portal history actions and details signal"
```

### Stage 17 closure

- [ ] Run the full per-stage commands from §1.
- [ ] Set UI-003 and RECOMMENDATION-003 to GREEN; add frontend evidence to
  EVENT-004 and TUTORIAL-010 without changing their already-GREEN backend status.
- [ ] Record hashes and verification in worklog.
- [ ] Commit:

```bash
git add docs/traceability.md 01_AI_WORKLOG_CURRENT.md
git commit -m "docs(stage17): close portal details traceability and evidence"
```

## 9. Stage 18 — Extraction, confirmations and errors

### Checkpoint 18A — authoritative Plane presence additive pass

**Files:**

- Modify: `internal/transport/dto.go`, `dto_test.go`
- Modify: `web/src/api/types.ts`, `web/src/test/builders.ts`

- [ ] **Step 1: write failing transport tests.**

Cases must prove EXPLORING + WAITING_RETURN + RETURNING count as
`observers_in_plane`; OUTBOUND/AVAILABLE/LOST and another Plane do not;
`observers_waiting_return` is the exact subset; counts use the same resolved
snapshot timestamp and do not add persistence columns.

```go
view, err := transport.BuildStateSnapshot(snapshotWithPlanePresence(), now, cfg)
require.NoError(t, err)
require.Equal(t, 3, view.Planes[0].ObserversInPlane)
require.Equal(t, 1, view.Planes[0].ObserversWaitingReturn)
```

- [ ] **Step 2: run and observe RED, then commit.**

```bash
go test -count=1 ./internal/transport -run TestBuildStateSnapshot_PlaneObserverPresence
git add internal/transport/dto_test.go
git commit -m "test(stage18): RED authoritative plane observer presence"
```

- [ ] **Step 3: implement two additive derived fields only.**

Build counts in one observer pass and merge by Plane ID. Unknown referenced Plane
remains an invariant error. Do not alter DB schema or observer lifecycle.

- [ ] **Step 4: run focused/full backend GREEN and commit.**

```bash
gofmt -w internal/transport/dto.go internal/transport/dto_test.go
go test -count=1 ./internal/transport ./internal/httpapi ./internal/realtime
git add internal/transport web/src/api/types.ts web/src/test/builders.ts
git commit -m "feat(stage18): GREEN authoritative plane observer presence"
```

### Checkpoint 18B — Extraction chooser and local Plane artwork

**Files:**

- Create: `web/src/features/extraction/plane-filter.ts`, `plane-filter.test.ts`
- Create: `web/src/features/extraction/ExtractionDialog.tsx`, test and CSS
- Modify: `web/src/app/AppShell.tsx`
- Create: `web/e2e/extraction.spec.ts`

- [ ] **Step 1: write failing filter/dialog tests.**

Required cases:

```text
all 85 Plane cards use local artwork resolver
search matches case-insensitive name and aliases
ALL/UNEXPLORED/EXPLORED/OBSERVER PRESENT filters are mutually exclusive
OBSERVER PRESENT uses observers_in_plane > 0, not Events
card shows explored badge, in-plane count and waiting-return count
selection summary shows exactly 30 Lab Energy cost
submit sends selected positive plane_id once
success closes dialog and accepts returned snapshot
409/404/network error preserves selection and shows feedback
keyboard focus is trapped and Escape cancels without POST
```

Pure filter signature:

```ts
export type PlaneFilter = 'ALL' | 'UNEXPLORED' | 'EXPLORED' | 'OBSERVER_PRESENT';
export function filterPlanes(
  planes: readonly PlaneDTO[],
  query: string,
  filter: PlaneFilter,
): readonly PlaneDTO[];
```

- [ ] **Step 2: run and observe RED, then commit.**

```bash
npm --prefix web run test -- src/features/extraction
git add web/src/features/extraction web/e2e/extraction.spec.ts
git commit -m "test(stage18): RED extraction plane chooser"
```

- [ ] **Step 3: implement chooser without client eligibility rules.**

`OPEN EXTRACTION` always permits opening the chooser. The dialog explains current
Lab Energy and cost but does not predict backend acceptance from those values.
The API remains authoritative for slot, energy and selected Plane validity.

- [ ] **Step 4: run focused/browser GREEN and commit.**

```bash
npm --prefix web run test -- src/features/extraction
npm --prefix web run test:e2e -- extraction.spec.ts
git add web
git commit -m "feat(stage18): GREEN extraction plane chooser"
```

### Checkpoint 18C — confirmation modal and global error states

**Files:**

- Create: `web/src/components/feedback/ConfirmDialog.tsx`, test and CSS
- Create: `web/src/components/feedback/ConnectionState.tsx`, test
- Modify: `web/src/features/portal-actions/usePortalCommand.ts`, tests
- Modify: Dashboard/Details/NotFound pages and tests
- Extend: `web/e2e/dashboard.spec.ts`, `portal-details.spec.ts`

- [ ] **Step 1: write failing confirmation/error tests.**

Required cases:

```text
confirmable 409 opens dialog with action and Portal identity
confirm repeats same method/id once with confirm=true
cancel sends no second request
second non-confirmable failure closes busy state and shows toast/inline message
network failure retains last authoritative snapshot and exposes retry
offline/reconnecting/connected state is announced accessibly
unknown app route and unknown Portal have distinct recovery screens
focus returns to invoking command after modal closes
```

Exact retry proof:

```ts
api.close
  .mockRejectedValueOnce(new ApiError(409, 'CONFIRMATION_REQUIRED', true, 'confirmation required'))
  .mockResolvedValueOnce(snapshot);
await user.click(screen.getByRole('button', { name: /close/i }));
await user.click(screen.getByRole('button', { name: /confirm close/i }));
expect(api.close.mock.calls).toEqual([[42, false], [42, true]]);
```

- [ ] **Step 2: run and observe RED, then commit.**

```bash
npm --prefix web run test -- src/components/feedback src/features/portal-actions
git add web/src/components/feedback web/src/features/portal-actions \
  web/src/pages web/e2e
git commit -m "test(stage18): RED confirmations and resilient error states"
```

- [ ] **Step 3: implement minimal modal/toast/retry orchestration.**

There is one application modal at a time and an ordered toast queue. Error copy
uses server code/message; HTML from server is never rendered. Duplicate click
while in flight is ignored by native disabled state.

- [ ] **Step 4: run focused and browser GREEN.**

```bash
npm --prefix web run test -- src/components/feedback src/features/portal-actions \
  src/features/extraction src/pages
npm --prefix web run test:e2e -- dashboard.spec.ts portal-details.spec.ts \
  extraction.spec.ts
```

- [ ] **Step 5: commit GREEN.**

```bash
git add web
git commit -m "feat(stage18): GREEN confirmations extraction and error states"
```

### Stage 18 closure

- [ ] Run the full per-stage commands from §1.
- [ ] Add exact Extraction UI tests to EXTRACTION requirement rows and
  confirmation/error frontend tests to API-008, API-010, API-011.
- [ ] Keep UI-001..003 GREEN; record the two-field PlaneDTO additive contract,
  RED/GREEN hashes and verification in worklog.
- [ ] Commit:

```bash
git add docs/traceability.md 01_AI_WORKLOG_CURRENT.md
git commit -m "docs(stage18): close extraction and error evidence"
```

## 10. Stage 19 — Event Log

### Checkpoint 19A — shared chronological log and filters

**Files:**

- Create: `web/src/pages/EventLogPage.test.tsx`
- Create: `web/src/features/event-log/event-filter.ts`
- Create: `web/src/features/event-log/event-filter.test.ts`
- Create: `web/src/features/event-log/EventFilters.tsx`, test and CSS
- Modify: `web/src/pages/EventLogPage.tsx`, `EventLogPage.module.css`
- Modify: `web/src/components/events/EventList.tsx`, `EventRow.tsx`

- [ ] **Step 1: write failing filtering/rendering tests.**

Required cases:

```text
all API events render in received chronological order without pagination
all canonical event_type values have stable human labels
event type filter supports one or many selected types
Portal/Observer/Plane ID filters use exact positive IDs
combining filters is logical AND and never reorders rows
clearing filters restores the same row order
payload_json stays collapsed until requested and is rendered as text, never HTML
empty source and empty filter result have distinct messages
```

Pure filter contract:

```ts
export interface EventFilter {
  eventTypes: ReadonlySet<EventType>;
  portalId: number | null;
  observerId: number | null;
  planeId: number | null;
}

export function filterEvents(
  events: readonly EventDTO[],
  filter: EventFilter,
): readonly EventDTO[];
```

- [ ] **Step 2: run and observe RED, then commit.**

```bash
npm --prefix web run test -- src/features/event-log src/pages/EventLogPage.test.tsx
git add web/src/features/event-log web/src/pages/EventLogPage.test.tsx
git commit -m "test(stage19): RED global event log and filters"
```

- [ ] **Step 3: implement Event Log on the shared Event components.**

The page owns a coalesced `GET /api/events` live resource. It refreshes on an
accepted snapshot only while the route is mounted. Filtering is client-side over
the complete MVP source and does not request alternate ordering or pagination.

- [ ] **Step 4: run focused GREEN and commit.**

```bash
npm --prefix web run test -- src/components/events src/features/event-log \
  src/pages/EventLogPage.test.tsx
git add web/src/components/events web/src/features/event-log web/src/pages/EventLogPage.tsx \
  web/src/pages/EventLogPage.module.css
git commit -m "feat(stage19): GREEN global event log and filters"
```

### Checkpoint 19B — realtime refresh and explicit Event Log signal

**Files:**

- Extend: `web/src/pages/EventLogPage.test.tsx`
- Extend: `web/src/features/tutorial/navigation-signal.ts`, test
- Modify: `web/src/app/AppShell.tsx`
- Create: `web/e2e/events.spec.ts`

- [ ] **Step 1: write failing integration tests.**

Required cases:

```text
new accepted snapshot causes at most one trailing Event refresh
Event Log route unmount aborts and unsubscribes
navigation during matching Step 8 sends EVENT_LOG_OPENED once
navigation at any other step sends no Tutorial signal
Strict Mode and repeated render do not duplicate EVENT_LOG_OPENED
new events appear after realtime edge without resetting active filters
```

- [ ] **Step 2: observe RED and commit.**

```bash
npm --prefix web run test -- src/features/tutorial/navigation-signal.test.ts \
  src/pages/EventLogPage.test.tsx
git add web/src/features/tutorial/navigation-signal* web/src/pages/EventLogPage.test.tsx \
  web/e2e/events.spec.ts
git commit -m "test(stage19): RED realtime event log and tutorial signal"
```

- [ ] **Step 3: implement one-shot matching navigation intent.**

Event Log navigation carries an intent only when current authoritative
`expected_action` is `OPEN_EVENT_LOG`. The destination consumes it once; a direct
URL load remains read-only.

- [ ] **Step 4: run unit/browser GREEN and commit.**

```bash
npm --prefix web run test -- src/features/tutorial src/pages/EventLogPage.test.tsx
npm --prefix web run test:e2e -- events.spec.ts
git add web
git commit -m "feat(stage19): GREEN realtime event log and explicit signal"
```

### Stage 19 closure

- [ ] Run the full per-stage commands from §1.
- [ ] Set UI-004 GREEN and add frontend evidence to EVENT-005, EVENT-011 and
  TUTORIAL-010.
- [ ] Record RED/GREEN hashes and verification in worklog.
- [ ] Commit:

```bash
git add docs/traceability.md 01_AI_WORKLOG_CURRENT.md
git commit -m "docs(stage19): close event log traceability and evidence"
```

## 11. Stage 20 — Tutorial UI

### Checkpoint 20A — authoritative guidance copy and step presentation

**Files:**

- Create: `web/src/features/tutorial/tutorial-copy.ts`
- Create: `web/src/features/tutorial/tutorial-copy.test.ts`
- Create: `web/src/features/tutorial/TutorialPanel.tsx`, test and CSS
- Modify: `web/src/app/AppShell.tsx`
- Modify: `web/src/components/dashboard/PortalSlot.tsx`

- [ ] **Step 1: encode failing table tests for all authoritative step/phase
  combinations.**

`tutorial-copy.ts` returns structured copy rather than JSX:

```ts
export interface TutorialGuidance {
  title: string;
  explanation: readonly string[];
  instruction: string;
  cta: 'BEGIN_PRACTICE' | 'TRY_SEND' | 'START_LIVE' | null;
  waiting: boolean;
}

export function tutorialGuidance(app: AppDTO): TutorialGuidance | null;
```

Required English meanings by step:

| Step | Title | Required instruction/copy |
|---:|---|---|
| 0 | `Welcome to Omenpath Research Lab` | Goal 85/85; identify Dashboard, Lab Summary, Observers, seven Slots, Needs Attention, Details and Event Log; `BEGIN PRACTICE`; no prices or detailed Portal rules |
| 1 | `Read a Portal` | Portal Energy differs from Lab Energy; individual decay; Time does not guarantee Energy; Stability, Risk, Recommendation and History; open matching Details |
| 2 | `Wait for the corridor` | Creatures block SEND and leave one every 2 seconds; player waits; no rejected SEND required |
| 3 | `Send an Observer` | SEND costs 0; requires AVAILABLE Observer; transit takes 5–15 seconds; first movement fixes OUTBOUND |
| 4 | `Stabilize an Omenpath` | STABILIZE costs 20; Lab Energy is 0–100 and regenerates +1/sec; Portal must be UNSTABLE and ≤85%; adds 15 Portal Energy; resulting Risk becomes MEDIUM/LOW |
| 5 | `Respect CRITICAL risk` | CRITICAL blocks SEND/RECALL; CLOSE costs 5; CLOSED differs from COLLAPSED; Collapse drains Lab Energy and starts 20-second Override; `TRY SEND` |
| 6 | `Bring the Observer home` | Research takes 20 sec; RECALL costs 0; longest-waiting is selected; new Portal fixes INBOUND; phase-specific wait/send/recall instruction |
| 7 | `Survive the return` | Plane becomes EXPLORED only after successful return; terminal Portal during transit makes Observer LOST; wait |
| 8 | `Inspect the Event Log` | Global Log and Portal History share one source; open Event Log |
| 9 | `Training complete` | Natural Portals, Extraction cost 30, 5-second sync, first automatic return and Tutorial→Live continuity; `START LIVE` |

Tests assert forbidden Step 0 terms are absent: `SEND`, `RECALL`, `5 Energy`,
`20 Energy`, `30 Energy`, `decay`, `CRITICAL`, `LOST`, `OUTBOUND`.

- [ ] **Step 2: run and observe RED.**

```bash
npm --prefix web run test -- src/features/tutorial/tutorial-copy.test.ts \
  src/features/tutorial/TutorialPanel.test.tsx
```

Expected: FAIL because guidance table/panel are absent.

- [ ] **Step 3: commit RED.**

```bash
git add web/src/features/tutorial/tutorial-copy.test.ts \
  web/src/features/tutorial/TutorialPanel.test.tsx
git commit -m "test(stage20): RED contextual tutorial guidance"
```

- [ ] **Step 4: implement guidance as a persistent non-blocking panel.**

The panel renders current `tutorial_step`, `tutorial_phase`, target IDs and one
instruction. It can collapse on phone but its current instruction/CTA remains
visible. Target Slot receives a tutorial outline without reordering. Wait phases
show state-derived text, not a fake client countdown.

- [ ] **Step 5: run focused GREEN and commit.**

```bash
npm --prefix web run test -- src/features/tutorial/tutorial-copy.test.ts \
  src/features/tutorial/TutorialPanel.test.tsx src/pages/DashboardPage.test.tsx
git add web/src/features/tutorial web/src/app/AppShell.tsx \
  web/src/components/dashboard/PortalSlot.tsx
git commit -m "feat(stage20): GREEN contextual tutorial guidance"
```

### Checkpoint 20B — actions, retry, reset and Tutorial→Live journey

**Files:**

- Extend: `web/src/features/tutorial/TutorialPanel.test.tsx`
- Extend: `web/src/features/portal-actions/PortalActions.tsx`, tests
- Modify: `web/src/features/tutorial/navigation-signal.ts`
- Create: `web/e2e/tutorial.spec.ts`

- [ ] **Step 1: write failing Tutorial orchestration tests.**

Required component/integration cases:

```text
Step 0 BEGIN PRACTICE posts TUTORIAL_INTRO_COMPLETED once
Step 1 matching DETAILS and Step 8 EVENT LOG use explicit signals only
Step 2 has no action CTA and waits for authoritative progress
Steps 3/4/6 highlight exact target command but call ordinary command endpoints
Step 5 matching target force-enables only SEND as TRY SEND
Step 5 TRY SEND posts despite can_send_observer=false and expects critical 409
Step 5 critical 409 plus the subsequent authoritative WS snapshot advances guidance without generic failure toast
wrong step, wrong target or different unavailable reason never force-enables
recreated tutorial_portal_id moves highlight to fresh Slot
LOST retry follows SEND_REPLACEMENT/WAIT_RESEARCH/RECALL_READY phases
Reset requires confirmation, calls reset once and returns to Step 0
Step 9 START LIVE calls live endpoint and removes Tutorial panel only after LIVE snapshot
Strict Mode cannot duplicate intro, critical attempt, reset or Live POST
```

The only unavailable override is explicit:

```ts
export function isExpectedCriticalSend(app: AppDTO, portalId: number): boolean {
  return app.mode === 'TUTORIAL' &&
    app.expected_action === 'ATTEMPT_CRITICAL_SEND' &&
    app.tutorial_portal_id === portalId;
}
```

- [ ] **Step 2: run and observe RED, then commit.**

```bash
npm --prefix web run test -- src/features/tutorial src/features/portal-actions
git add web/src/features/tutorial web/src/features/portal-actions \
  web/e2e/tutorial.spec.ts
git commit -m "test(stage20): RED tutorial actions retry and live handoff"
```

- [ ] **Step 3: implement the minimal orchestration.**

Expected Step 5 `PORTAL_CRITICAL_RISK` is treated as successful learning feedback
only when the latest snapshot has already advanced beyond Step 5 or a subsequent
WebSocket snapshot does so. Any other 409 follows normal error handling. Frontend
does not advance the step locally.

Reset confirmation states that Energy, Observers, exploration and Tutorial Event
history are reset by backend. Live CTA does not claim success until returned/WS
snapshot mode is `LIVE`.

- [ ] **Step 4: run full Tutorial browser journey.**

```bash
npm --prefix web run test -- src/features/tutorial src/features/portal-actions
npm --prefix web run test:e2e -- tutorial.spec.ts
```

The Playwright journey starts from a fresh temporary DB, asserts Step 0 survives
ticks, completes Steps 0→9, permits real wait phases, reaches LIVE with 0/7 OPEN
Portals, preserved Energy/Event/Observer/Plane continuity and no broken primary
flow. Timeout is explicit and at least 180 seconds; no arbitrary `waitForTimeout`
is used—wait on UI/state conditions.

- [ ] **Step 5: commit GREEN.**

```bash
git add web
git commit -m "feat(stage20): GREEN complete tutorial user journey"
```

### Stage 20 closure

- [ ] Run the full per-stage commands from §1.
- [ ] Set TUTORIAL-016 and TUTORIAL-019 GREEN; attach frontend evidence to
  TUTORIAL-001..015, TUTORIAL-017 and TUTORIAL-018 without weakening backend
  evidence.
- [ ] Record actual journey duration, retries, RED/GREEN hashes and verification
  in worklog.
- [ ] Commit:

```bash
git add docs/traceability.md 01_AI_WORKLOG_CURRENT.md
git commit -m "docs(stage20): close tutorial UI traceability and evidence"
```

## 12. Stage 21 — AI Worklog UI

### Checkpoint 21A — single-source Markdown page

**Files:**

- Create: `web/src/pages/AIWorklogPage.test.tsx`
- Modify: `web/src/pages/AIWorklogPage.tsx`, `AIWorklogPage.module.css`
- Modify: `web/src/vite-env.d.ts`, `web/vite.config.ts`
- Modify: `01_AI_WORKLOG_CURRENT.md` only to fill honest Block D facts

- [ ] **Step 1: write failing Worklog source/rendering tests.**

Required cases:

```text
page source is imported from repository 01_AI_WORKLOG_CURRENT.md?raw
headings, lists, tables and code blocks render as semantic HTML
GFM tables do not overflow the page; their wrapper scrolls horizontally
raw HTML from Markdown is not executed
page exposes tools, development time, token availability, stages,
developer-vs-AI contribution, key prompts, decisions, AI mistakes,
manual rewrites, verification and future improvements
app build fails if the root Markdown import cannot resolve
```

The source import is explicit and unique:

```ts
import worklog from '../../../01_AI_WORKLOG_CURRENT.md?raw';

export function AIWorklogPage() {
  return <ReactMarkdown remarkPlugins={[remarkGfm]}>{worklog}</ReactMarkdown>;
}
```

Do not enable `rehype-raw`.

- [ ] **Step 2: run and observe RED, then commit.**

```bash
npm --prefix web run test -- src/pages/AIWorklogPage.test.tsx
git add web/src/pages/AIWorklogPage.test.tsx
git commit -m "test(stage21): RED single-source AI worklog page"
```

- [ ] **Step 3: update the Markdown honestly before rendering it.**

Add the actual Block D planning/implementation stages, human decisions, AI
mistakes/corrections, verification and timing. If aggregate token usage remains
unavailable, state that explicitly; never estimate a number. Record that the
developer selected Mission Control, local artwork, Canvas sparks, English UI,
backend reason codes and phone all-slots layout.

- [ ] **Step 4: implement safe Markdown rendering and responsive styles.**

Vite must allow this one root import without exposing arbitrary filesystem files.
The page adds a generated table of contents from parsed headings but does not
duplicate or rewrite the Markdown content.

- [ ] **Step 5: run focused GREEN and production build.**

```bash
npm --prefix web run test -- src/pages/AIWorklogPage.test.tsx
npm --prefix web run typecheck
npm --prefix web run build
```

Expected: PASS and built page contains current worklog headings.

- [ ] **Step 6: commit GREEN.**

```bash
git add 01_AI_WORKLOG_CURRENT.md web/src/pages/AIWorklogPage.tsx \
  web/src/pages/AIWorklogPage.module.css web/src/vite-env.d.ts web/vite.config.ts
git commit -m "feat(stage21): GREEN single-source AI worklog UI"
```

### Checkpoint 21B — cross-screen navigation and final frontend integration

**Files:**

- Extend: `web/src/app/App.test.tsx`
- Extend: all `web/e2e/*.spec.ts`
- Modify only if tests prove necessary: App Shell/navigation/shared styles

- [ ] **Step 1: write failing integration assertions for the complete shell.**

Required cases:

```text
keyboard navigation reaches Dashboard, Events and AI Worklog
browser back/forward preserves correct route and fresh authoritative state
phone bottom nav does not cover Slot controls
desktop sidebar does not change content width when WS status changes
all primary routes recover after WebSocket reconnect
art credits expose source/artist/policy and no remote image request occurs
no route logs React warnings, unhandled rejection or console error
```

- [ ] **Step 2: observe RED and commit.**

```bash
npm --prefix web run test -- src/app/App.test.tsx
npm --prefix web run test:e2e
git add web/src/app/App.test.tsx web/e2e
git commit -m "test(stage21): RED complete frontend navigation and integration"
```

- [ ] **Step 3: apply only test-proven integration fixes.**

Do not add Stage 22 quality features, deployment/static Go serving, auth,
localization, themes or new domain behavior. A failing existing backend contract
is handled as a separate RED/GREEN corrective commit with explicit evidence.

- [ ] **Step 4: run GREEN and commit.**

```bash
npm --prefix web run test
npm --prefix web run test:e2e
git add web
git commit -m "feat(stage21): GREEN complete frontend integration"
```

### Stage 21 closure

- [ ] Run the full per-stage commands from §1.
- [ ] Set UI-005 GREEN and confirm UI-001..007 plus TUTORIAL-019 remain GREEN.
- [ ] Update worklog with real Stage 21 hashes and commands, then commit:

```bash
git add docs/traceability.md 01_AI_WORKLOG_CURRENT.md
git commit -m "docs(stage21): close worklog UI and Block D stage evidence"
```

## 13. Stage-level verification and Git evidence

Для каждого checkpoint Stage 15A–D, 16A–B, 17A–B, 18A–C, 19A–B, 20A–B и
21A–B executor добавляет в worklog отдельную строку с RED hash, наблюдавшейся
причиной падения, GREEN hash и focused command. Запись делается сразу после
GREEN, поэтому к завершению блока в ней нет пропущенных hashes/output. Сам этот
planning-файл сохраняет unchecked boxes и не переписывается в ложный
completed-state заранее.

Порядок закрытия каждой стадии:

1. `git status --short` — проверить, что generated/unrelated files поняты.
2. `npm --prefix web run format`, затем `format:check`.
3. `npm --prefix web run lint`.
4. `npm --prefix web run typecheck`.
5. `npm --prefix web run test`.
6. `npm --prefix web run build`.
7. Для changed Go: `gofmt -w <exact changed .go files>`.
8. `gofmt -l .` — пустой output.
9. `go vet ./...`.
10. `go build ./...`.
11. `go test -count=1 ./...`.
12. Обновить traceability/worklog и сделать stage docs commit.

Если любой focused GREEN проходит только отдельно, но падает full suite, стадия
остаётся незавершённой. Flaky retry не является GREEN evidence; сначала найти и
устранить причину.

## 14. Block D final verification

После Stage 21 выполнить из repository root на чистом dependency install:

```bash
PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 npm_config_cache=/tmp/omenpath-npm-cache \
  npm --prefix web ci
npm --prefix web run format:check
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
npm --prefix web run test:e2e
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
go test -race -count=1 ./...
git diff --check
git status --short
```

Expected:

- `npm ci` uses committed lockfile without modifying it;
- no global package or new Playwright browser is installed;
- frontend unit/integration and all Playwright flows PASS;
- `web/dist` builds without remote-art/runtime-fetch assumptions;
- `gofmt -l .` is empty;
- vet/build/ordinary/race Go suites PASS;
- only intentional documentation changes remain before final commit;
- `.superpowers/` is ignored and does not enter a product commit.

Manual browser smoke after automated suite:

1. At 1440×900 confirm desktop Dashboard uses 4+3 Slots and no Slot shifts.
2. At 390×760 confirm all seven Slots/four commands fit with no vertical Portal
   board/page scroll.
3. Confirm selected/Needs Attention Portal has fast spark corona; background
   Portals remain low density; reduced motion is static.
4. Disable network temporarily: last snapshot remains visible and reconnect state
   is clear; restore network and observe fresh authoritative state.
5. Complete Tutorial through Live, including actual rejected Step 5 SEND.
6. Open Extraction and verify local art/search/all four filters and 30 Energy
   explanation.
7. Open Details, Event Log and AI Worklog; verify navigation and shared history.
8. Inspect browser network: no runtime request to Scryfall, Wizards or another
   artwork host.

### Final consistency audit

- [ ] Re-read Final Spec §§5–6, 9, 13, 23–29, 35–36, 39–41.
- [ ] Compare TypeScript transport fields/enums with `internal/transport/dto.go`.
- [ ] Confirm quick reasons are additive and command errors/events are unchanged.
- [ ] Confirm Plane presence fields are derived, not persisted.
- [ ] Confirm Tutorial copy/actions/system completion match §28.2 one step at a
  time.
- [ ] Confirm Dashboard contains no per-Slot Risk/Recommendation/History.
- [ ] Confirm all UI-001..007 and TUTORIAL-019 traceability rows are GREEN with
  named tests/implementation.
- [ ] Confirm no requirement moved from GREEN to PARTIAL/PLANNED.
- [ ] Confirm no Stage 22 file/feature was started.

Update `02_IMPLEMENTATION_ROADMAP_TDD.md` only after every check passes:

```text
Block D — GREEN; Stages 15–21 complete and boundary-verified.
Block E — PLANNED; next planning boundary, not started.
```

Append a Block D final section to worklog with:

- stage/checkpoint RED and GREEN hashes;
- dependency/tool versions actually used;
- asset totals, fallback count and provenance policy;
- ordinary and race suite outputs;
- Playwright desktop/mobile/Tutorial results;
- honest timing/token availability;
- deviations and corrective passes.

Final documentation commit:

```bash
git add 01_AI_WORKLOG_CURRENT.md 02_IMPLEMENTATION_ROADMAP_TDD.md \
  docs/requirements.md docs/traceability.md
git commit -m "docs(block-d): reconcile frontend verification and traceability"
```

After this commit: STOP and request the required user checkpoint. Do not draft or
implement Block E / Stage 22 in the same execution.

At the same checkpoint report the measured size of `web/node_modules` and
`/tmp/omenpath-npm-cache`, then offer their explicit cleanup. Do not delete
either path without the user's confirmation.

## 15. Definition of Done — Block D

Block D is complete only when all statements are true:

- [ ] React/TypeScript/Vite workspace has reproducible lockfile and green
  format/lint/typecheck/test/build.
- [ ] REST/WebSocket clients and one authoritative snapshot store tolerate
  ordering, reconnect and Strict Mode without duplicate effects.
- [ ] Quick action availability/reasons come from backend; ordinary unavailable
  clicks do not POST.
- [ ] Matching Tutorial Step 5 is the only forced rejected-action exception.
- [ ] Dashboard renders complete summary, Needs Attention, Override, Empty State
  and seven stable Slots with exactly four commands.
- [ ] Desktop uses 4+3; phone keeps all seven Slots simultaneously monitorable
  without Portal-list scrolling.
- [ ] Local Plane art/fallback resolves all 85 seed IDs; runtime has no external
  art dependency; credits/policy notice are visible.
- [ ] Canvas particles originate on the circumference, move tangentially/outward,
  keep the image center clean and obey caps/pause/reduced-motion.
- [ ] Portal Details contains every Portal/Destination/Diagnostics/History/Action
  section and no hidden risk inputs.
- [ ] Extraction chooser has local art, search, four filters, Observer presence,
  30 Energy copy and authoritative submission/errors.
- [ ] Confirmation repeats exact endpoint with `confirm:true`; errors and
  reconnect never invent optimistic state.
- [ ] Event Log and Portal History share formatting/source semantics and preserve
  chronology.
- [ ] Tutorial UI delivers contextual Steps 0–9, retry/recreation/LOST behavior,
  explicit signals and verified Live handoff.
- [ ] AI Worklog page renders the root Markdown as its single maintained source
  and exposes all Final Spec categories honestly.
- [ ] UI-001..007, TUTORIAL-019 and RECOMMENDATION-003 are GREEN in traceability;
  related backend rows retain their stronger evidence.
- [ ] Complete frontend suite, Playwright suite, Go format/vet/build/test and Go
  race suite pass from clean install.
- [ ] Worklog and roadmap describe actual completed state and hashes.
- [ ] Stage 22 has not begun.
