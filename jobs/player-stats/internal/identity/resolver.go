package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rspassos/ilha/jobs/player-stats/internal/model"
)

const (
	ResolutionByLogin = "reused_login"
	ResolutionByAlias = "reused_alias"
	ResolutionCreated = "created"

	AliasActionInserted = "inserted"
	AliasActionUpdated  = "updated"
)

var ErrRepositoryNotInitialized = errors.New("identity repository is not initialized")

type repository interface {
	FindPlayerByLogin(context.Context, string) (string, error)
	FindPlayerByAlias(context.Context, string, string) (string, error)
	CreateCanonicalPlayer(context.Context, string, string, time.Time) (string, error)
	PromotePrimaryLogin(context.Context, string, string, time.Time) error
	UpsertAlias(context.Context, string, string, string, time.Time) (bool, error)
}

type Resolver struct {
	repository repository
	now        func() time.Time
}

func NewResolver(repository repository) *Resolver {
	return &Resolver{
		repository: repository,
		now:        time.Now().UTC,
	}
}

func (r *Resolver) ResolvePlayer(ctx context.Context, input model.ResolvePlayerInput) (model.PlayerIdentity, error) {
	if r == nil || r.repository == nil {
		return model.PlayerIdentity{}, ErrRepositoryNotInitialized
	}
	if err := ctx.Err(); err != nil {
		return model.PlayerIdentity{}, err
	}

	observedName := strings.TrimSpace(input.ObservedName)
	if observedName == "" {
		return model.PlayerIdentity{}, errors.New("observed name must not be empty")
	}

	observedLogin := strings.TrimSpace(input.ObservedLogin)
	observedAt := input.ObservedAt
	if observedAt.IsZero() {
		observedAt = r.now()
	}

	playerID, err := r.repository.FindPlayerByLogin(ctx, observedLogin)
	if err != nil {
		return model.PlayerIdentity{}, fmt.Errorf("find player by login: %w", err)
	}
	if playerID != "" {
		aliasAction, err := upsertAlias(ctx, r.repository, playerID, observedName, observedLogin, observedAt)
		if err != nil {
			return model.PlayerIdentity{}, err
		}
		return model.PlayerIdentity{
			PlayerID:    playerID,
			AliasName:   observedName,
			Login:       observedLogin,
			Resolution:  ResolutionByLogin,
			AliasAction: aliasAction,
		}, nil
	}

	playerID, err = r.repository.FindPlayerByAlias(ctx, observedName, observedLogin)
	if err != nil {
		return model.PlayerIdentity{}, fmt.Errorf("find player by alias: %w", err)
	}
	if playerID != "" {
		if err := r.repository.PromotePrimaryLogin(ctx, playerID, observedLogin, observedAt); err != nil {
			return model.PlayerIdentity{}, fmt.Errorf("promote primary login for player %s: %w", playerID, err)
		}
		aliasAction, err := upsertAlias(ctx, r.repository, playerID, observedName, observedLogin, observedAt)
		if err != nil {
			return model.PlayerIdentity{}, err
		}
		return model.PlayerIdentity{
			PlayerID:    playerID,
			AliasName:   observedName,
			Login:       observedLogin,
			Resolution:  ResolutionByAlias,
			AliasAction: aliasAction,
		}, nil
	}

	playerID, err = r.repository.CreateCanonicalPlayer(ctx, observedLogin, observedName, observedAt)
	if err != nil {
		return model.PlayerIdentity{}, fmt.Errorf("create canonical player: %w", err)
	}

	aliasAction, err := upsertAlias(ctx, r.repository, playerID, observedName, observedLogin, observedAt)
	if err != nil {
		return model.PlayerIdentity{}, err
	}

	return model.PlayerIdentity{
		PlayerID:    playerID,
		AliasName:   observedName,
		Login:       observedLogin,
		Resolution:  ResolutionCreated,
		AliasAction: aliasAction,
	}, nil
}

func upsertAlias(ctx context.Context, repository repository, playerID string, observedName string, observedLogin string, observedAt time.Time) (string, error) {
	inserted, err := repository.UpsertAlias(ctx, playerID, observedName, observedLogin, observedAt)
	if err != nil {
		return "", fmt.Errorf("upsert alias %q: %w", observedName, err)
	}
	if inserted {
		return AliasActionInserted, nil
	}
	return AliasActionUpdated, nil
}
