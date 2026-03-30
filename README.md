# ilha
Ilha Quake World Servers

## Jobs

- [jobs/collector](/home/rspassos/projects/ilha/jobs/collector): one-shot job that loads server config, fetches `lastscores` and `laststats`, merges matches by `demo`, and persists rows into `collector_matches` with idempotent `UPSERT`.
- [jobs/player-stats](/home/rspassos/projects/ilha/jobs/player-stats): one-shot job that reads `collector_matches`, resolves player identities and aliases, and consolidates analytical rows into PostgreSQL with idempotent `UPSERT`.

The local development flow uses `docker compose` with PostgreSQL, a fixture-backed `mock-api`, and both job containers mounted with the repository workspace.

## Local setup

1. Copy the environment file:

```bash
cp .env.example .env
```

2. Start the local stack for both jobs:

```bash
docker compose up -d postgres mock-api collector player-stats
```

3. Validate bootstrap for both jobs:

```bash
docker compose exec collector go run ./cmd/collector --bootstrap-only --config ./config/collector.local.yaml
docker compose exec player-stats go run ./cmd/player-stats --bootstrap-only
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

## Operational checks

Inspect collected matches:

```bash
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select server_key, demo_name, mode, map_name, has_bots from collector_matches order by played_at desc;"
```

Inspect consolidated player stats:

```bash
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select collector_match_id, demo_name, observed_name, normalized_mode, has_bots, excluded_from_analytics from player_match_stats order by played_at desc, observed_name asc;"
```

The `mock-api` service serves local payload fixtures from `docs/responses`, so local validation does not depend on the external upstream endpoints.

## Stop local stack

```bash
docker compose down
```
