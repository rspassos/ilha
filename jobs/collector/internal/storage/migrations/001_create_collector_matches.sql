CREATE TABLE IF NOT EXISTS collector_matches (
    id bigserial PRIMARY KEY,
    server_key text NOT NULL,
    server_name text NOT NULL,
    demo_name text NOT NULL,
    match_key text NOT NULL,
    mode text NOT NULL,
    map_name text NOT NULL,
    participants text NULL,
    played_at timestamptz NOT NULL,
    duration_seconds integer NULL,
    hostname text NULL,
    has_bots boolean NOT NULL DEFAULT false,
    score_payload jsonb NOT NULL,
    stats_payload jsonb NULL,
    merged_payload jsonb NOT NULL,
    ingested_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT collector_matches_server_demo_key UNIQUE (server_key, demo_name)
);

CREATE INDEX IF NOT EXISTS collector_matches_mode_played_at_idx
    ON collector_matches (mode, played_at DESC);

CREATE INDEX IF NOT EXISTS collector_matches_server_played_at_idx
    ON collector_matches (server_key, played_at DESC);

CREATE INDEX IF NOT EXISTS collector_matches_has_bots_played_at_idx
    ON collector_matches (has_bots, played_at DESC);
