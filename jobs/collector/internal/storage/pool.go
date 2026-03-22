package storage

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool = pgxpool.Pool

func NewPool(ctx context.Context, databaseURL string) (*Pool, error) {
	return Open(ctx, databaseURL)
}
