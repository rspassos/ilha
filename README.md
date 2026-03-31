# ilha
Ilha Quake World Servers

## Jobs

- [jobs/collector](/home/rspassos/projects/ilha/jobs/collector): one-shot job that loads server config, fetches `lastscores` and `laststats`, merges matches by `demo`, and persists rows into `collector_matches` with idempotent `UPSERT`.
- [jobs/player-stats](/home/rspassos/projects/ilha/jobs/player-stats): one-shot job that reads `collector_matches`, resolves player identities and aliases, and consolidates analytical rows into PostgreSQL with idempotent `UPSERT`.
- [services/player-stats-api](/home/rspassos/projects/ilha/services/player-stats-api): always-on HTTP service that exposes public player rankings plus Prometheus metrics for local validation.

The local development flow uses `docker compose` with PostgreSQL, a fixture-backed `mock-api`, both job containers mounted with the repository workspace, and the `player-stats-api` service running on `:8080` with metrics on `:9092`.

## Local setup

1. Copy the environment file:

```bash
cp .env.example .env
```

2. Start the local stack for the jobs and API:

```bash
docker compose up -d postgres mock-api collector player-stats player-stats-api
```

3. Validate bootstrap for the jobs and API:

```bash
docker compose exec collector go run ./cmd/collector --bootstrap-only --config ./config/collector.local.yaml
docker compose exec player-stats go run ./cmd/player-stats --bootstrap-only
docker compose exec player-stats-api go run ./cmd/player-stats-api --bootstrap-only
```

## Run locally

Run the collector first to populate `collector_matches` from the fixture-backed API:

```bash
docker compose exec collector go run ./cmd/collector --config ./config/collector.local.yaml
```

Then run the player stats job to consolidate analytics from the collected matches:

```bash
docker compose exec player-stats go run ./cmd/player-stats
```

Re-run the jobs as needed to validate idempotency:

```bash
docker compose exec collector go run ./cmd/collector --config ./config/collector.local.yaml
docker compose exec player-stats go run ./cmd/player-stats
```

The API container starts automatically with `docker compose up`. If you need to restart it after env changes:

```bash
docker compose restart player-stats-api
```

## Operational checks

Inspect collected matches:

```bash
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select server_key, demo_name, mode, map_name, has_bots from collector_matches order by played_at desc;"
```

Inspect consolidated player stats:

```bash
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select collector_match_id, demo_name, observed_name, normalized_mode, has_bots, excluded_from_analytics from player_match_stats order by played_at desc, observed_name asc;"
```

Query the public ranking endpoint:

```bash
curl "http://127.0.0.1:8080/v1/rankings/players?mode=2on2&limit=5"
```

Inspect Prometheus metrics exposed by the API:

```bash
curl "http://127.0.0.1:9092/metrics"
```

Inspect structured logs from the API service:

```bash
docker compose logs player-stats-api --tail=50
```

The `mock-api` service serves local payload fixtures from `docs/responses`, so local validation does not depend on the external upstream endpoints.

## Stop local stack

```bash
docker compose down
```
