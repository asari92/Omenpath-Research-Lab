# Omenpath deployment

Omenpath runs as one application container behind the existing HTTPS host
nginx. The container serves the compiled web application, REST API, WebSocket
and health endpoint on internal port `8080`; Compose publishes it only as
`127.0.0.1:8080` on the host.

## Production environment

Create `.env` from the committed example. The required values are:

```dotenv
OMENPATH_ENV=production
OMENPATH_COOKIE_SECURE=true
OMENPATH_ADDR=:8080
OMENPATH_DB_PATH=/var/lib/omenpath/omenpath.db
OMENPATH_WEB_ROOT=/app/web
```

The application has no session signing secret. It generates opaque random
session tokens and stores only their hashes in SQLite. The database, including
server-side sessions and all laboratories, persists through the bind mount
`/opt/omenpath/data:/var/lib/omenpath`. Do not commit `.env` or the database.

## First launch

Place or clone the repository at `/opt/omenpath`, then run:

```bash
cd /opt/omenpath
cp .env.example .env
sudo chown -R 10001:10001 /opt/omenpath/data
docker compose --env-file .env up -d --build
docker compose ps
curl -fsS http://127.0.0.1:8080/health
curl -fsSI http://127.0.0.1:8080/
```

The health response is `{"status":"ok"}`. `docker compose ps` should show a
healthy service published as `127.0.0.1:8080->8080/tcp`.

## Updating

This procedure recreates the application container without deleting the
persistent SQLite database:

```bash
cd /opt/omenpath
git pull --ff-only
docker compose --env-file .env build
docker compose --env-file .env up -d --remove-orphans
docker compose ps
curl -fsS http://127.0.0.1:8080/health
curl -fsSI http://127.0.0.1:8080/ui/lab-shell-background.png
```

The final request verifies that the reference-faithful local laboratory
background is present in the rebuilt SPA image; it must return `200`.

Back up `/opt/omenpath/data/omenpath.db` through the already prepared
`/opt/omenpath/backups` workflow when required. Do not remove the data directory
during an update.

## Existing host nginx

Inside the existing HTTPS `server` block for `omenpath.duckdns.org`, replace
the placeholder application location with these locations:

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

The only WebSocket endpoint is `/ws/lab`; its location requires the `Upgrade`
and `Connection` headers shown above. No nginx, TLS, DNS or firewall service is
part of the Compose project.

## Diagnostics

```bash
docker compose ps
docker compose logs --tail=100 omenpath
curl -fsS http://127.0.0.1:8080/health
```

Application data is under `/opt/omenpath/data`. Container filesystem changes
outside `/var/lib/omenpath` are disposable.
