# AGENTS

## Project status

The `match-stats-collector` MVP under `jobs/collector` is implemented through tasks 1 to 8 in [tasks/prd-match-stats-collector/tasks.md](/home/rspassos/projects/ilha/tasks/prd-match-stats-collector/tasks.md).

Current flow:

1. Load YAML server config plus env vars.
2. Fetch `lastscores` and `laststats` per enabled server.
3. Merge matches by `demo`.
4. Persist into PostgreSQL with idempotent `UPSERT`.
5. Emit JSON logs and Prometheus metrics during a single `run once`.

## Important paths

- [jobs/collector/cmd/collector/main.go](/home/rspassos/projects/ilha/jobs/collector/cmd/collector/main.go): binary entrypoint.
- [jobs/collector/internal/bootstrap/bootstrap.go](/home/rspassos/projects/ilha/jobs/collector/internal/bootstrap/bootstrap.go): app wiring, migrations, metrics server, run-once startup.
- [jobs/collector/internal/config/config.go](/home/rspassos/projects/ilha/jobs/collector/internal/config/config.go): YAML and env loading.
- [jobs/collector/internal/httpclient/client.go](/home/rspassos/projects/ilha/jobs/collector/internal/httpclient/client.go): upstream HTTP fetch and decode.
- [jobs/collector/internal/merge/merge.go](/home/rspassos/projects/ilha/jobs/collector/internal/merge/merge.go): correlate `lastscores` and `laststats` by `demo`.
- [jobs/collector/internal/storage/postgres.go](/home/rspassos/projects/ilha/jobs/collector/internal/storage/postgres.go): migrations and PostgreSQL `UPSERT`.
- [jobs/collector/internal/metrics/metrics.go](/home/rspassos/projects/ilha/jobs/collector/internal/metrics/metrics.go): Prometheus collectors.
- [jobs/collector/internal/model/](/home/rspassos/projects/ilha/jobs/collector/internal/model): internal score, stats, merged-record models.
- [jobs/collector/config/collector.local.yaml](/home/rspassos/projects/ilha/jobs/collector/config/collector.local.yaml): local fixture-backed config.
- [compose.yml](/home/rspassos/projects/ilha/compose.yml): local `collector`, `postgres`, and `mock-api` services.

## Storage contract

Main table:

- `collector_matches`

Defined in:

- [001_create_collector_matches.sql](/home/rspassos/projects/ilha/jobs/collector/internal/storage/migrations/001_create_collector_matches.sql)

Key rules:

- unique match identity is `(server_key, demo_name)`
- raw payloads are stored as `jsonb`
- `has_bots`, `mode`, `server_key`, and `played_at` are indexed for filtering

## Runtime behavior

- The collector is intentionally single-shot, suitable for an external scheduler.
- `--bootstrap-only` validates config loading and exits before collection.
- Non-zero process exit means at least one server failed, but other enabled servers were still attempted.
- Merge warnings such as `mode_mismatch` are expected with the current fixtures for team games because `lastscores` uses `2on2` while `laststats` uses `team`.

## Local development

Setup:

```bash
cp .env.example .env
docker compose up -d postgres mock-api collector
```

Bootstrap check:

```bash
docker compose exec collector go run ./cmd/collector --bootstrap-only --config ./config/collector.local.yaml
```

Full local run:

```bash
docker compose exec collector go run ./cmd/collector --config ./config/collector.local.yaml
```

Verify persisted rows:

```bash
docker compose exec postgres psql -U ilha -d ilha -c "select server_key, demo_name, mode, map_name, has_bots from collector_matches order by played_at desc;"
```

Stop local stack:

```bash
docker compose down
```

## Testing

Default package test run:

```bash
cd jobs/collector
GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod-cache go test ./...
```

Run PostgreSQL-backed integration tests:

```bash
cd jobs/collector
COLLECTOR_TEST_DATABASE_URL='postgres://ilha:ilha@127.0.0.1:5432/ilha?sslmode=disable' \
GOCACHE=/tmp/go-build \
GOMODCACHE=/tmp/go-mod-cache \
go test ./...
```

Notes:

- `internal/storage/postgres_test.go` and `internal/collector/integration_test.go` use a real PostgreSQL instance when `COLLECTOR_TEST_DATABASE_URL` is set.
- `internal/httpclient/client_test.go` uses `httptest`.
- Local end-to-end validation uses `mock-api`, which serves files from `docs/responses`.

## Metrics exposed

Implemented metrics:

- `collector_runs_total`
- `collector_server_runs_total`
- `collector_matches_fetched_total`
- `collector_matches_upserted_total`
- `collector_merge_warnings_total`
- `collector_request_duration_seconds`

## Guidance for future changes

- Preserve `demo` as the external correlation key unless the upstream contract changes.
- Keep merge logic simple and explicit; avoid normalizing away upstream differences too early.
- If changing persistence fields, update both the SQL migration strategy and `model.MatchRecord`.
- If adding new external dependencies, expect to run Go commands with writable `GOCACHE` and `GOMODCACHE`.
- Prefer extending the existing local validation flow in `compose.yml` instead of inventing a separate ad hoc setup.
