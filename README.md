# Omenpath Research Lab

Web application for the keeper of a magical laboratory. Temporary Omenpaths
open into 85 worlds; the player evaluates their risk, stabilizes or closes
them, and sends Observers on expeditions. The simulation continues in realtime
and an anonymous browser session preserves an isolated laboratory for 30 days.

- Live application: <https://omenpath.duckdns.org/>
- Repository: <https://github.com/asari92/Omenpath-Research-Lab>
- AI-assisted development history: `/ai-worklog` in the application or
  [`01_AI_WORKLOG_CURRENT.md`](01_AI_WORKLOG_CURRENT.md)

## Quick start

Requirements: Go 1.26.4, Node.js 22 and npm. SQLite is embedded; no separate
database server is required.

```bash
git clone https://github.com/asari92/Omenpath-Research-Lab.git
cd Omenpath-Research-Lab
npm ci --prefix web
npm --prefix web run build
OMENPATH_WEB_ROOT=web/dist OMENPATH_DB_PATH=./omenpath.db go run ./cmd/server
```

Open <http://localhost:8080/>. The database is created automatically. For the
production Docker deployment, environment variables, persistence and nginx
configuration, see [`docs/deployment.md`](docs/deployment.md).

## What to verify

The Dashboard always shows seven Portal Slots and the laboratory summary.
Select an occupied card to open its Details with Risk, Recommendation and
History. Portal commands are available both on the Dashboard and Details:

- **Stabilize** reinforces an eligible unstable Portal;
- **Close** terminates an open Portal, with confirmation when creatures or an
  Observer transit make the action dangerous;
- **Send** dispatches an available Observer unless the corridor is blocked or
  Risk is CRITICAL;
- **Recall** returns the longest-waiting Observer from the destination Plane.

Unavailable commands explain why they cannot be performed. Every accepted
action and every rejected domain action is recorded in Event Log. `Needs
Attention` identifies the highest-priority open Portal.

### Risk formula

For an OPEN Portal:

```text
energy_lifetime    = current_portal_energy / energy_decay_per_second
effective_lifetime = min(scheduled_time_remaining, energy_lifetime)
base_risk          = max(0, (45 - effective_lifetime) / 45 * 100)
risk_score         = min(100, base_risk + (20 if UNSTABLE else 0))
```

The interface exposes the understandable level, not the internal score:
`LOW` is `0..25`, `MEDIUM` is `>25..50`, `HIGH` is `>50..75`, and `CRITICAL`
is `>75..100`. The hidden random instability deadline is deliberately excluded
from the formula. Therefore Risk is useful guidance, not a promise that an
unstable Portal is safe.

Scheduled or manual termination produces **CLOSED**. Energy depletion or a
hidden instability failure produces **COLLAPSED**; a Collapse drains Lab Energy
and activates Leyline Override. These are intentionally different outcomes.

## Tests

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
go test -race -count=1 ./...

npm --prefix web run format:check
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web test
npm --prefix web run build
npm --prefix web run test:e2e
```

### Pre-submission checklist

- [x] Empty laboratory renders seven fixed empty Slots and waits for a Portal.
- [x] A CRITICAL Portal rejects Observer dispatch and records the rejection.
- [x] Stabilizing the prepared unstable Portal changes HIGH/CRITICAL Risk to
      MEDIUM/LOW.
- [x] Closing with creatures or active transit requests confirmation; terminal
      Portals reject further actions.
- [x] Several operations appear chronologically in Event Log and the matching
      Portal History.
- [x] Reload continues the same laboratory, while a separate browser profile
      receives an isolated one.

The repository contains substantially more than the required three tests:
domain boundary tests, persistence and API integration tests, frontend unit
tests, responsive browser scenarios and race/concurrency coverage.

## Architecture

One Go process owns the simulation, persists state and events in SQLite, serves
REST and `/ws/lab`, and hosts the compiled React application. The production
container is exposed only through `127.0.0.1:8080` behind the existing HTTPS
nginx. No secret or raw session token is stored in source code or in SQLite.
