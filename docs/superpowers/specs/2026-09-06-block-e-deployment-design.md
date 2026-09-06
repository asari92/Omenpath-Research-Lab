# Block E Deployment-First Design

**Status:** approved for planning on 2026-09-06

**Goal:** deploy the current Omenpath application behind the already configured
host nginx at `https://omenpath.duckdns.org/` with the smallest practical Docker
surface.

## Scope

Block E is reduced to the work required for the first public deployment:

1. package the existing backend and frontend into one production container;
2. persist SQLite data outside the container;
3. expose the application only on the VPS loopback interface;
4. document the exact first-run, update and host-nginx proxy commands;
5. perform only build/config validation and a basic health smoke check.

Balance simulation, exhaustive browser testing, extended final QA, CI/CD,
monitoring and infrastructure automation are deferred. This deployment pass
must not claim that those deferred audits were performed.

## Existing infrastructure boundary

The implementation assumes all of the following already exist and does not
modify them:

- VPS, Docker and Docker Compose;
- DuckDNS record for `omenpath.duckdns.org`;
- host nginx site configuration;
- Let's Encrypt certificate and Certbot renewal;
- HTTPS redirect;
- `/opt/omenpath`, `/opt/omenpath/data` and `/opt/omenpath/backups`.

There will be no nginx, Caddy or Traefik container, no DNS/firewall/SSH work,
and no deployment framework.

## Container architecture

A single multi-stage `Dockerfile` builds the application:

1. a Node build stage runs the existing frontend production build;
2. a Go build stage produces the server binary with `CGO_ENABLED=0`;
3. a minimal Alpine runtime contains only the binary, built frontend and the
   small tools needed by the container healthcheck.

The Go process serves all same-origin traffic on internal port `8080`:

- the compiled SPA and its client-side route fallback;
- `/api/*`;
- `/ws/lab`;
- `/health`.

Static files are read from `OMENPATH_WEB_ROOT`. `/health` is public and does not
create or refresh a laboratory session.

## Compose and persistence

The Compose file defines one service, `omenpath`, with:

- `127.0.0.1:8080:8080` as its only published port;
- `restart: unless-stopped`;
- `/opt/omenpath/data:/var/lib/omenpath` as a bind mount;
- `OMENPATH_DB_PATH=/var/lib/omenpath/omenpath.db`;
- an HTTP healthcheck against `http://127.0.0.1:8080/health`.

The SQLite database contains both gameplay state and hashed server-side session
tokens. Keeping the database under `/opt/omenpath/data` therefore preserves
games and anonymous sessions across restart and container recreation.

No session-signing secret is needed by the current architecture. Session tokens
are generated with cryptographic randomness, sent as cookies and stored only as
hashes in SQLite. No real secret or `.env` file is committed.

## Production environment

`.env.example` documents these required values:

```dotenv
OMENPATH_ENV=production
OMENPATH_COOKIE_SECURE=true
OMENPATH_ADDR=:8080
OMENPATH_DB_PATH=/var/lib/omenpath/omenpath.db
OMENPATH_WEB_ROOT=/app/web
```

`OMENPATH_COOKIE_SECURE=true` is mandatory because the public site is HTTPS.

## Existing nginx integration

Host nginx remains the only public endpoint. It proxies ordinary HTTP traffic
to `http://127.0.0.1:8080` and forwards WebSocket upgrade headers for
`/ws/lab`. The project will provide the minimal `location` snippets but will not
edit the server configuration.

The resulting path is:

```text
Internet -> existing nginx/HTTPS -> 127.0.0.1:8080 -> Docker Omenpath
```

## Minimal launch gate

This pass does not add or run an exhaustive QA matrix. Before handoff it checks
only that:

- the production image builds;
- the Compose file resolves successfully;
- the container becomes healthy;
- `http://127.0.0.1:8080/health` answers successfully;
- the root SPA is served;
- the existing `/ws/lab` path and required nginx headers are documented.

Any gameplay/UI review after deployment is manual and may produce a later
corrective pass.
