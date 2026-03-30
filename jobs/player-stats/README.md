# Player Stats

Batch job that reads `collector_matches`, resolves player identities and aliases, and persists analytical rows into PostgreSQL with idempotent `UPSERT`.

## Local setup

1. Copy `.env.example` to `.env`.
2. Start the local stack:

```bash
docker compose up -d postgres mock-api collector player-stats
```

3. Bootstrap the collector and player-stats jobs:

```bash
docker compose exec collector go run ./cmd/collector --bootstrap-only --config ./config/collector.local.yaml
docker compose exec player-stats go run ./cmd/player-stats --bootstrap-only
```

4. Load fixture-backed matches into `collector_matches`:

```bash
docker compose exec collector go run ./cmd/collector --config ./config/collector.local.yaml
```

5. Consolidate player stats:

```bash
docker compose exec player-stats go run ./cmd/player-stats
```

6. Re-run the job to confirm idempotency:

```bash
docker compose exec player-stats go run ./cmd/player-stats
```

## Operational verification

Inspect the consolidated rows:

```bash
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select collector_match_id, demo_name, observed_name, normalized_mode, has_bots, excluded_from_analytics from player_match_stats order by played_at desc, observed_name asc;"
```

Check that bot matches remain consolidated but excluded from default analytics:

```bash
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select demo_name, observed_name, has_bots, excluded_from_analytics from player_match_stats where has_bots = true order by demo_name, observed_name;"
```

Check idempotency directly in PostgreSQL:

```bash
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select count(*) as total_rows, count(distinct (collector_match_id, player_id, observed_name)) as unique_rows from player_match_stats;"
```

Create a realistic alias evolution scenario by copying one collected match and renaming a player while keeping the same login:

```bash
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "with base as (select * from collector_matches where stats_payload is not null order by played_at desc, id desc limit 1), renamed_payload as (select jsonb_set(stats_payload, '{players,0,name}', to_jsonb('AliasRenamed'::text), false) as stats_payload from base) insert into collector_matches (server_key, server_name, demo_name, match_key, mode, map_name, participants, played_at, duration_seconds, hostname, has_bots, score_payload, stats_payload, merged_payload, ingested_at, updated_at) select base.server_key, base.server_name, base.demo_name || '-alias-check', base.match_key || '-alias-check', base.mode, base.map_name, base.participants, base.played_at + interval '1 second', base.duration_seconds, base.hostname, base.has_bots, base.score_payload, renamed_payload.stats_payload, base.merged_payload, now(), now() from base cross join renamed_payload;"
docker compose exec player-stats go run ./cmd/player-stats
```

Inspect the alias registry after the reprocess:

```bash
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select pc.display_name, pa.alias_name, coalesce(pa.login, '') as login, pa.first_seen_at, pa.last_seen_at from player_aliases pa join player_canonical pc on pc.id = pa.player_id order by pc.display_name, pa.alias_name;"
```

## Tests

Package tests:

```bash
cd jobs/player-stats
GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod-cache go test ./...
```

PostgreSQL-backed integration tests:

```bash
cd jobs/player-stats
COLLECTOR_TEST_DATABASE_URL='postgres://ilha:ilha@127.0.0.1:5432/ilha?sslmode=disable' \
GOCACHE=/tmp/go-build \
GOMODCACHE=/tmp/go-mod-cache \
go test ./...
```

The integration suite covers idempotent reprocessing, bot exclusion flags, and alias evolution with a real PostgreSQL instance.
