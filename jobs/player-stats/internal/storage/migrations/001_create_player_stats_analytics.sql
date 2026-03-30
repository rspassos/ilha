CREATE TABLE IF NOT EXISTS player_stats_schema_migrations (
    version text PRIMARY KEY,
    applied_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS player_canonical (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    primary_login text NULL,
    display_name text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS player_canonical_primary_login_uidx
    ON player_canonical (primary_login)
    WHERE primary_login IS NOT NULL;

CREATE TABLE IF NOT EXISTS player_aliases (
    id bigserial PRIMARY KEY,
    player_id uuid NOT NULL REFERENCES player_canonical(id),
    alias_name text NOT NULL,
    login text NULL,
    first_seen_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS player_aliases_alias_login_uidx
    ON player_aliases (alias_name, COALESCE(login, ''));

CREATE INDEX IF NOT EXISTS player_aliases_player_id_idx
    ON player_aliases (player_id);

CREATE TABLE IF NOT EXISTS player_match_stats (
    id bigserial PRIMARY KEY,
    collector_match_id bigint NOT NULL REFERENCES collector_matches(id),
    player_id uuid NOT NULL REFERENCES player_canonical(id),
    server_key text NOT NULL,
    demo_name text NOT NULL,
    observed_name text NOT NULL,
    observed_login text NULL,
    team text NULL,
    map_name text NOT NULL,
    raw_mode text NULL,
    normalized_mode text NOT NULL,
    played_at timestamptz NOT NULL,
    has_bots boolean NOT NULL,
    excluded_from_analytics boolean NOT NULL,
    frags integer NOT NULL,
    deaths integer NOT NULL,
    kills integer NOT NULL,
    team_kills integer NOT NULL,
    suicides integer NOT NULL,
    damage_taken integer NOT NULL,
    damage_given integer NOT NULL,
    spree_max integer NOT NULL,
    spree_quad integer NOT NULL,
    rl_hits integer NOT NULL,
    rl_kills integer NOT NULL,
    lg_attacks integer NOT NULL,
    lg_hits integer NOT NULL,
    ga integer NOT NULL,
    ra integer NOT NULL,
    ya integer NOT NULL,
    health_100 integer NOT NULL,
    ping integer NOT NULL,
    efficiency numeric(5,2) NOT NULL,
    lg_accuracy numeric(5,2) NOT NULL,
    stats_snapshot jsonb NOT NULL,
    consolidated_at timestamptz NOT NULL,
    CONSTRAINT player_match_stats_collector_player_observed_key UNIQUE (collector_match_id, player_id, observed_name),
    CONSTRAINT player_match_stats_normalized_mode_check CHECK (normalized_mode IN ('1on1', '2on2', '3on3', '4on4', 'dmm4'))
);

CREATE INDEX IF NOT EXISTS player_match_stats_player_played_at_idx
    ON player_match_stats (player_id, played_at DESC);

CREATE INDEX IF NOT EXISTS player_match_stats_mode_played_at_idx
    ON player_match_stats (normalized_mode, played_at DESC);

CREATE INDEX IF NOT EXISTS player_match_stats_map_played_at_idx
    ON player_match_stats (map_name, played_at DESC);

CREATE INDEX IF NOT EXISTS player_match_stats_server_played_at_idx
    ON player_match_stats (server_key, played_at DESC);

CREATE INDEX IF NOT EXISTS player_match_stats_excluded_played_at_idx
    ON player_match_stats (excluded_from_analytics, played_at DESC);

CREATE TABLE IF NOT EXISTS player_stats_checkpoints (
    job_name text PRIMARY KEY,
    last_collector_match_id bigint NOT NULL,
    updated_at timestamptz NOT NULL
);
