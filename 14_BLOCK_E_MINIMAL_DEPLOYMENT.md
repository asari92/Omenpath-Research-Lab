# Block E Minimal Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Do not dispatch subagents. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** package and launch Omenpath as one Docker application behind the existing HTTPS host nginx at `omenpath.duckdns.org`.

**Architecture:** A multi-stage image builds the React SPA and Go server, then runs one non-root Go process that serves SPA, REST, WebSocket and health traffic on port `8080`. Docker Compose binds that port only to host loopback and bind-mounts `/opt/omenpath/data` for the SQLite database. Existing host nginx remains the sole public entrypoint.

**Tech Stack:** Go 1.26.4, React/Vite, Docker multi-stage build, Docker Compose, SQLite, existing host nginx/Let's Encrypt.

**Approved design:** [`docs/superpowers/specs/2026-09-06-block-e-deployment-design.md`](docs/superpowers/specs/2026-09-06-block-e-deployment-design.md)

---

## Scope and stop rule

This is a deployment-first Block E slice. It includes only production HTTP
serving, containerization, persistent storage, environment documentation,
deployment commands and a basic launch smoke.

It does not install or configure nginx, Certbot, DNS, firewall, SSH, CI/CD,
monitoring or orchestration. Balance simulation, exhaustive E2E, extended final
QA and consistency audit stay in the backlog. Do not claim those checks ran.

The implementation runs inline and stops after the deployment handoff. It does
not begin another product stage.

## Fixed deployment contract

| Item | Value |
|---|---|
| Public domain | `https://omenpath.duckdns.org/` |
| Container application port | `8080` |
| Host binding | `127.0.0.1:8080:8080` |
| Persistent host path | `/opt/omenpath/data` |
| Persistent container path | `/var/lib/omenpath` |
| SQLite path | `/var/lib/omenpath/omenpath.db` |
| Health endpoint | `/health` |
| WebSocket path | `/ws/lab` |
| Restart policy | `unless-stopped` |

## Target files

```text
Dockerfile                         production multi-stage image
.dockerignore                      small and secret-safe build context
compose.yml                        one loopback-bound application service
.env.example                       production environment contract
internal/httpapi/production.go     health + SPA/static production wrapper
internal/httpapi/production_test.go focused serving contract
cmd/server/main.go                 OMENPATH_WEB_ROOT composition
cmd/server/main_test.go            environment parsing coverage
docs/deployment.md                 exact VPS/nginx/run/update instructions
README.md                          short link to deployment guide
02_IMPLEMENTATION_ROADMAP_TDD.md  deployment-first Block E status
01_AI_WORKLOG_CURRENT.md          implementation and launch evidence
docs/traceability.md               health/container delivery evidence
```

## Stage 22 — Production HTTP surface

### Task 22.1 — Add health and SPA serving

**Files:**

- Create: `internal/httpapi/production.go`
- Create: `internal/httpapi/production_test.go`
- Modify: `cmd/server/main.go`
- Modify: `cmd/server/main_test.go`

- [ ] Add focused tests for this exact public wrapper contract:

```go
func TestProductionHandler_HealthBypassesSession(t *testing.T)
func TestProductionHandler_ServesAssetsAndSPAFallback(t *testing.T)
func TestProductionHandler_DelegatesAPIAndWebSocket(t *testing.T)
func TestProductionHandler_RejectsMissingStaticAsset(t *testing.T)
func TestServerConfig_WebRootEnvironment(t *testing.T)
```

The health test uses an API handler that fails the test when called, proving
that `GET /health` does not create or refresh an anonymous laboratory.

- [ ] Implement this production boundary:

```go
func NewProductionHandler(api http.Handler, webRoot string) (http.Handler, error)
```

Its routing order is fixed:

1. exact `GET /health` returns `200`, `Content-Type: application/json` and
   `{"status":"ok"}`;
2. `/api/*` and exact `/ws/lab` delegate unchanged to the existing router;
3. an existing frontend file is served from `webRoot`;
4. a non-file `GET` or `HEAD` route serves `index.html` for React routing;
5. a missing `/assets/*` file returns `404`, not `index.html`;
6. other methods delegate to the existing router and retain its normal errors.

Use `http.Dir`/`http.FileServer` and cleaned URL paths. Do not build a custom
filesystem path by concatenating unchecked request text.

- [ ] Extend `serverConfig` with `webRoot`, populated only from
  `OMENPATH_WEB_ROOT`. When non-empty, `run` wraps the existing API router with
  `NewProductionHandler`; when empty, local API-only startup remains valid.

- [ ] Run only the focused serving check:

```bash
go test -count=1 ./internal/httpapi ./cmd/server
```

- [ ] Commit:

```bash
git add internal/httpapi/production.go internal/httpapi/production_test.go cmd/server/main.go cmd/server/main_test.go
git commit -m "feat(block-e): serve production web and health"
```

## Stage 23 — Container and persistent Compose service

### Task 23.1 — Add the production image

**Files:**

- Create: `Dockerfile`
- Create: `.dockerignore`

- [ ] Create a three-stage `Dockerfile` with this structure:

```dockerfile
FROM node:22-alpine AS web-build
WORKDIR /src
COPY web/package.json web/package-lock.json ./web/
RUN npm ci --prefix web
COPY web ./web
COPY 01_AI_WORKLOG_CURRENT.md ./
RUN npm --prefix web run build

FROM golang:1.26.4-alpine AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY data ./data
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/omenpath ./cmd/server

FROM alpine:3.22
RUN addgroup -S -g 10001 omenpath && adduser -S -D -H -u 10001 -G omenpath omenpath
WORKDIR /app
COPY --from=go-build /out/omenpath /app/omenpath
COPY --from=web-build /src/web/dist /app/web
RUN mkdir -p /var/lib/omenpath && chown -R 10001:10001 /var/lib/omenpath
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/app/omenpath"]
```

- [ ] Keep the Docker context small and exclude local/runtime state:

```dockerignore
.git
.worktrees
.env
omenpath.db
omenpath.db-*
web/node_modules
web/dist
web/coverage
web/playwright-report
web/test-results
coverage.*
*.out
```

### Task 23.2 — Add Compose and environment contract

**Files:**

- Create: `compose.yml`
- Create: `.env.example`

- [ ] Create exactly one Compose service:

```yaml
services:
  omenpath:
    build:
      context: .
      dockerfile: Dockerfile
    image: omenpath:local
    env_file:
      - .env
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - type: bind
        source: /opt/omenpath/data
        target: /var/lib/omenpath
    restart: unless-stopped
    init: true
```

Do not add a proxy, database, certificate or utility service.

- [ ] Document only real application variables:

```dotenv
OMENPATH_ENV=production
OMENPATH_COOKIE_SECURE=true
OMENPATH_ADDR=:8080
OMENPATH_DB_PATH=/var/lib/omenpath/omenpath.db
OMENPATH_WEB_ROOT=/app/web
```

There is no session/signing secret in the current architecture. Opaque random
session tokens are hashed into the persistent SQLite database.

- [ ] Validate only what is required to launch:

```bash
cp .env.example .env
docker compose config
docker build -t omenpath:local .
```

- [ ] Commit:

```bash
git add Dockerfile .dockerignore compose.yml .env.example
git commit -m "feat(block-e): containerize persistent application"
```

## Stage 24 — Deployment handoff

### Task 24.1 — Write exact operator instructions

**Files:**

- Create: `docs/deployment.md`
- Modify: `README.md`

- [ ] Document the first launch:

```bash
cd /opt/omenpath
cp .env.example .env
sudo chown -R 10001:10001 /opt/omenpath/data
docker compose --env-file .env up -d --build
docker compose ps
curl -fsS http://127.0.0.1:8080/health
curl -fsSI http://127.0.0.1:8080/
```

- [ ] Document an update without deleting the persistent database:

```bash
cd /opt/omenpath
git pull --ff-only
docker compose --env-file .env build
docker compose --env-file .env up -d --remove-orphans
docker compose ps
curl -fsS http://127.0.0.1:8080/health
```

- [ ] Document the minimal snippets to place inside the already existing HTTPS
  `server` block. They replace the placeholder location; the project does not
  execute nginx commands.

```nginx
location /ws/lab {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 3600s;
}

location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

- [ ] Add a short README link to `docs/deployment.md`; do not duplicate the
  whole runbook.

- [ ] Commit:

```bash
git add README.md docs/deployment.md
git commit -m "docs(block-e): add existing-nginx deployment runbook"
```

## Stage 25 — Minimal launch gate and documentation close

### Task 25.1 — Run the essential local container smoke

No balance simulation, exhaustive browser matrix or manual gameplay audit is
part of this gate.

- [ ] Preserve the repository's inexpensive compile/test baseline:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
npm --prefix web run typecheck
npm --prefix web run build
```

- [ ] Launch the exact Compose service and check its public surfaces locally:

```bash
cp .env.example .env
docker compose --env-file .env up -d --build
docker compose ps
curl -fsS http://127.0.0.1:8080/health
curl -fsSI http://127.0.0.1:8080/
docker compose logs --tail=100 omenpath
```

Expected essentials:

- Compose service is `healthy`;
- `/health` returns `{"status":"ok"}`;
- `/` returns the built SPA;
- the host publication is exactly `127.0.0.1:8080->8080/tcp`;
- SQLite files appear only under `/opt/omenpath/data` on the host.

- [ ] Do not run full Playwright E2E, balance simulation or extended QA in this
  deployment pass.

### Task 25.2 — Close the deployment slice honestly

**Files:**

- Modify: `01_AI_WORKLOG_CURRENT.md`
- Modify: `02_IMPLEMENTATION_ROADMAP_TDD.md`
- Modify: `docs/traceability.md`

- [ ] Record image/Compose/smoke results and exact commits. Mark only the minimal
  deployment slice complete. Keep balance, external manual QA and exhaustive
  audit explicitly deferred.

- [ ] Verify tracked-file hygiene without touching the user's local database:

```bash
git diff --check
git status --short
```

Expected: only the user's untracked `omenpath.db` remains.

- [ ] Commit:

```bash
git add 01_AI_WORKLOG_CURRENT.md 02_IMPLEMENTATION_ROADMAP_TDD.md docs/traceability.md
git commit -m "docs(block-e): record minimal deployment readiness"
```

## Final handoff checklist

The final response must state exactly:

- internal application port: `8080`;
- host binding: `127.0.0.1:8080`;
- Dockerfile and Compose paths;
- production environment variables;
- persistent host/container/database paths;
- health endpoint: `/health`;
- WebSocket path: `/ws/lab`;
- first-launch commands;
- update commands;
- host-nginx `location` snippets;
- which automated/manual checks were deliberately skipped;
- confirmation that no VPS/nginx/Certbot/DNS/firewall/SSH/CI/CD changes were made.
