# ilha
Ilha Quake World Servers

## Collector

The match collector lives in [jobs/collector](/home/rspassos/projects/ilha/jobs/collector). Local validation uses `docker compose` with PostgreSQL plus a fixture-backed `mock-api`, then runs the collector in one-shot mode against [collector.local.yaml](/home/rspassos/projects/ilha/jobs/collector/config/collector.local.yaml).
