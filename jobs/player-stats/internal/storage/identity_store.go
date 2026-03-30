package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type txIdentityStore struct {
	tx               pgx.Tx
	collectorMatchID int64
}

func (s txIdentityStore) FindPlayerByLogin(ctx context.Context, login string) (string, error) {
	trimmed := strings.TrimSpace(login)
	if trimmed == "" {
		return "", nil
	}

	var playerID string
	err := s.tx.QueryRow(ctx, "SELECT id::text FROM player_canonical WHERE primary_login = $1", trimmed).Scan(&playerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}

	return playerID, nil
}

func (s txIdentityStore) FindPlayerByAlias(ctx context.Context, aliasName string, login string) (string, error) {
	trimmedAlias := strings.TrimSpace(aliasName)
	trimmedLogin := strings.TrimSpace(login)

	var playerID string
	err := s.tx.QueryRow(ctx, `
		SELECT player_id::text
		FROM player_aliases
		WHERE alias_name = $1
		  AND COALESCE(login, '') = COALESCE($2, '')
	`, trimmedAlias, trimmedLogin).Scan(&playerID)
	if err == nil {
		return playerID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	rows, err := s.tx.Query(ctx, `
		SELECT DISTINCT player_id::text
		FROM player_aliases
		WHERE alias_name = $1
		ORDER BY player_id::text
	`, trimmedAlias)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	playerIDs := make([]string, 0, 2)
	for rows.Next() {
		if err := rows.Scan(&playerID); err != nil {
			return "", err
		}
		playerIDs = append(playerIDs, playerID)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(playerIDs) == 0 {
		return "", nil
	}
	if len(playerIDs) > 1 {
		return "", fmt.Errorf("alias %q is ambiguous", trimmedAlias)
	}

	return playerIDs[0], nil
}

func (s txIdentityStore) CreateCanonicalPlayer(ctx context.Context, login string, displayName string, createdAt time.Time) (string, error) {
	var playerID string
	if err := s.tx.QueryRow(ctx, `
		INSERT INTO player_canonical (primary_login, display_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, nullIfEmpty(login), strings.TrimSpace(displayName), createdAt, createdAt).Scan(&playerID); err != nil {
		return "", err
	}

	return playerID, nil
}

func (s txIdentityStore) PromotePrimaryLogin(ctx context.Context, playerID string, login string, updatedAt time.Time) error {
	trimmed := strings.TrimSpace(login)
	if trimmed == "" {
		return nil
	}

	if _, err := s.tx.Exec(ctx, `
		UPDATE player_canonical
		SET primary_login = COALESCE(primary_login, $2),
		    updated_at = $3
		WHERE id = $1::uuid
	`, playerID, trimmed, updatedAt); err != nil {
		return err
	}

	return nil
}

func (s txIdentityStore) UpsertAlias(ctx context.Context, playerID string, aliasName string, login string, observedAt time.Time) (bool, error) {
	var inserted bool
	if err := s.tx.QueryRow(ctx, aliasUpsertSQL,
		playerID,
		strings.TrimSpace(aliasName),
		nullIfEmpty(login),
		observedAt,
		observedAt,
	).Scan(&inserted); err != nil {
		return false, fmt.Errorf("upsert alias for match %d: %w", s.collectorMatchID, err)
	}

	return inserted, nil
}
