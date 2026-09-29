# massmaker-be

Catalogue, publication, events, and the authenticated vendor list. Identity lives in `enmasse-be` on a **different VPS**.

This repo owns its Dockerfile, Compose files, and Caddyfile. There is no shared Docker network.

## Local

```bash
cp .env.example .env
docker compose up -d postgres
# Enmasse must be reachable at ENMASSE_INTERNAL_URL (default http://127.0.0.1:8081)
go run ./cmd/api
# GET http://localhost:8080/health
```

If this API runs in Docker and Enmasse is on the host, set `ENMASSE_INTERNAL_URL=http://host.docker.internal:8081`.

The browser never calls Enmasse. Massmaker proxies `/v1/auth/*` and `/v1/me/profile` to `ENMASSE_INTERNAL_URL`. `/v1/internal` is not mounted here.

`POST /v1/events` accepts one event or `{ "events": [ ... ] }` (max 40). A valid Bearer stamps `enmasse_user_id`; missing auth stays anonymous. Invalid Bearer is 401. Repeat `contact_click` for the same session and maker within 30 minutes is stored with `counted=false`.

`GET /v1/me/vendors` returns **503** when `ENMASSE_INTERNAL_URL` is unset or Enmasse is down, and **403** `profile_incomplete` when Enmasse says the artist is incomplete.

## VPS

```bash
docker compose -f docker-compose.prod.yml up --build -d
```

Caddy serves `api.$MASSMAKER_DOMAIN`. `/v1/admin` is 404 on the public host — use `ssh -L 8080:127.0.0.1:8080`.
