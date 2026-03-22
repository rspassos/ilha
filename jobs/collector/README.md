# Collector

Match stats collector job for QuakeWorld servers. It loads server definitions from YAML, fetches `lastscores` and `laststats`, merges matches by `demo`, persists them into PostgreSQL with idempotent `UPSERT`, and emits structured logs plus Prometheus metrics during execution.

## Local setup

1. Copy `.env.example` to `.env`.
2. Start the local stack:

```bash
docker compose up -d postgres mock-api collector
```

3. Validate bootstrap only:

```bash
docker compose exec collector go run ./cmd/collector --bootstrap-only --config ./config/collector.local.yaml
```

4. Run one full collection cycle against the fixture API:

```bash
docker compose exec collector go run ./cmd/collector --config ./config/collector.local.yaml
```

5. Inspect persisted rows:

```bash
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select server_key, demo_name, mode, map_name, has_bots from collector_matches order by played_at desc;"
```

The `mock-api` service serves the sample payloads from `docs/responses`, so local validation does not depend on the external upstream endpoints.

## Configuration

Supported CLI flags:

- `--config`: path to the YAML file
- `--env-file`: path to the optional `.env` file
- `--bootstrap-only`: validate config loading and exit before the collection cycle

Required environment variables:

- `DATABASE_URL`
- `HUBAPI_BASE_URL`

Optional environment variables:

- `APP_ENV`, default `development`
- `LOG_LEVEL`, default `info`
- `METRICS_ADDR`, default `:9090`

YAML example:

```yaml
servers:
  - key: qlash-br-1
    name: Qlash Brazil 1
    address: qw.qlash.com.br:28501
    enabled: true
    timeout_seconds: 5
```

Validation rules:

- at least one server must be configured
- each server must have unique `key`
- `HUBAPI_BASE_URL` must include scheme and host
- each server requires `name` and `address`
- `timeout_seconds` defaults to `5` when omitted

## Scheduler usage

The binary is designed for a single `run once` execution:

```bash
go run ./cmd/collector --config ./config/collector.yaml
```

Exit code `0` means every enabled server completed successfully. A non-zero exit code means at least one server failed, but the collector still attempted the remaining enabled servers.

## Metrics and logs

The collector writes JSON logs for startup, per-server execution, merge warnings, and failures.

Prometheus metrics are exposed on `METRICS_ADDR` during execution:

- `collector_runs_total`
- `collector_server_runs_total`
- `collector_matches_fetched_total`
- `collector_matches_upserted_total`
- `collector_merge_warnings_total`
- `collector_request_duration_seconds`

## Troubleshooting

- If bootstrap fails, verify `DATABASE_URL`, the YAML path, and PostgreSQL connectivity.
- If upstream fetches fail, inspect the JSON logs for `server_key`, endpoint context, and wrapped request errors.
- If no rows are persisted locally, verify the `mock-api` container is running and that `collector.local.yaml` is the active config.
- PostgreSQL-backed integration tests require `COLLECTOR_TEST_DATABASE_URL`.
