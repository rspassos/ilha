package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rspassos/ilha/jobs/player-stats/internal/model"
)

func TestResolvePlayerReusesByLogin(t *testing.T) {
	t.Parallel()

	repository := &stubRepository{
		playerByLogin: map[string]string{"alpha-login": "player-1"},
	}
	resolver := NewResolver(repository)

	identity, err := resolver.ResolvePlayer(context.Background(), model.ResolvePlayerInput{
		ObservedName:  "AlphaRenamed",
		ObservedLogin: "alpha-login",
		ObservedAt:    time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ResolvePlayer() error = %v", err)
	}
	if identity.PlayerID != "player-1" {
		t.Fatalf("PlayerID = %q, want player-1", identity.PlayerID)
	}
	if identity.Resolution != ResolutionByLogin {
		t.Fatalf("Resolution = %q, want %q", identity.Resolution, ResolutionByLogin)
	}
	if identity.AliasAction != AliasActionInserted {
		t.Fatalf("AliasAction = %q, want %q", identity.AliasAction, AliasActionInserted)
	}
	if len(repository.aliasUpserts) != 1 {
		t.Fatalf("len(aliasUpserts) = %d, want 1", len(repository.aliasUpserts))
	}
	if repository.aliasUpserts[0].aliasName != "AlphaRenamed" {
		t.Fatalf("alias name = %q, want AlphaRenamed", repository.aliasUpserts[0].aliasName)
	}
}

func TestResolvePlayerReusesByAlias(t *testing.T) {
	t.Parallel()

	repository := &stubRepository{
		playerByAlias: map[aliasKey]string{{aliasName: "Alpha", login: ""}: "player-2"},
	}
	resolver := NewResolver(repository)

	identity, err := resolver.ResolvePlayer(context.Background(), model.ResolvePlayerInput{
		ObservedName: "Alpha",
		ObservedAt:   time.Date(2026, 3, 26, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ResolvePlayer() error = %v", err)
	}
	if identity.PlayerID != "player-2" {
		t.Fatalf("PlayerID = %q, want player-2", identity.PlayerID)
	}
	if identity.Resolution != ResolutionByAlias {
		t.Fatalf("Resolution = %q, want %q", identity.Resolution, ResolutionByAlias)
	}
	if len(repository.promotions) != 1 {
		t.Fatalf("len(promotions) = %d, want 1", len(repository.promotions))
	}
	if repository.promotions[0].playerID != "player-2" {
		t.Fatalf("promoted player = %q, want player-2", repository.promotions[0].playerID)
	}
}

func TestResolvePlayerCreatesCanonicalAndAliasWhenUnknown(t *testing.T) {
	t.Parallel()

	repository := &stubRepository{createdPlayerID: "player-3"}
	resolver := NewResolver(repository)

	identity, err := resolver.ResolvePlayer(context.Background(), model.ResolvePlayerInput{
		ObservedName:  "Bravo",
		ObservedLogin: "bravo-login",
		ObservedAt:    time.Date(2026, 3, 26, 14, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ResolvePlayer() error = %v", err)
	}
	if identity.PlayerID != "player-3" {
		t.Fatalf("PlayerID = %q, want player-3", identity.PlayerID)
	}
	if identity.Resolution != ResolutionCreated {
		t.Fatalf("Resolution = %q, want %q", identity.Resolution, ResolutionCreated)
	}
	if len(repository.createdPlayers) != 1 {
		t.Fatalf("len(createdPlayers) = %d, want 1", len(repository.createdPlayers))
	}
	if repository.createdPlayers[0].displayName != "Bravo" {
		t.Fatalf("displayName = %q, want Bravo", repository.createdPlayers[0].displayName)
	}
}

func TestResolvePlayerUpdatesAliasWindowWhenReused(t *testing.T) {
	t.Parallel()

	inserted := false
	repository := &stubRepository{
		playerByAlias:       map[aliasKey]string{{aliasName: "Charlie", login: ""}: "player-4"},
		upsertAliasInserted: &inserted,
	}
	resolver := NewResolver(repository)

	observedAt := time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)
	identity, err := resolver.ResolvePlayer(context.Background(), model.ResolvePlayerInput{
		ObservedName: "Charlie",
		ObservedAt:   observedAt,
	})
	if err != nil {
		t.Fatalf("ResolvePlayer() error = %v", err)
	}
	if identity.AliasAction != AliasActionUpdated {
		t.Fatalf("AliasAction = %q, want %q", identity.AliasAction, AliasActionUpdated)
	}
	if len(repository.aliasUpserts) != 1 {
		t.Fatalf("len(aliasUpserts) = %d, want 1", len(repository.aliasUpserts))
	}
	if !repository.aliasUpserts[0].observedAt.Equal(observedAt) {
		t.Fatalf("observedAt = %s, want %s", repository.aliasUpserts[0].observedAt, observedAt)
	}
}

func TestResolvePlayerRequiresObservedName(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(&stubRepository{})
	_, err := resolver.ResolvePlayer(context.Background(), model.ResolvePlayerInput{})
	if err == nil || err.Error() != "observed name must not be empty" {
		t.Fatalf("ResolvePlayer() error = %v, want observed name must not be empty", err)
	}
}

func TestResolvePlayerPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(&stubRepository{findByLoginErr: errors.New("boom")})
	_, err := resolver.ResolvePlayer(context.Background(), model.ResolvePlayerInput{
		ObservedName:  "Alpha",
		ObservedLogin: "alpha-login",
	})
	if err == nil || err.Error() != "find player by login: boom" {
		t.Fatalf("ResolvePlayer() error = %v, want wrapped repository error", err)
	}
}

type aliasKey struct {
	aliasName string
	login     string
}

type aliasUpsert struct {
	playerID   string
	aliasName  string
	login      string
	observedAt time.Time
}

type promotion struct {
	playerID string
	login    string
}

type createdPlayer struct {
	login       string
	displayName string
	createdAt   time.Time
}

type stubRepository struct {
	playerByLogin       map[string]string
	playerByAlias       map[aliasKey]string
	findByLoginErr      error
	findByAliasErr      error
	createErr           error
	promoteErr          error
	upsertAliasErr      error
	createdPlayerID     string
	upsertAliasInserted *bool

	aliasUpserts   []aliasUpsert
	promotions     []promotion
	createdPlayers []createdPlayer
}

func (r *stubRepository) FindPlayerByLogin(_ context.Context, login string) (string, error) {
	if r.findByLoginErr != nil {
		return "", r.findByLoginErr
	}
	return r.playerByLogin[login], nil
}

func (r *stubRepository) FindPlayerByAlias(_ context.Context, aliasName string, login string) (string, error) {
	if r.findByAliasErr != nil {
		return "", r.findByAliasErr
	}
	return r.playerByAlias[aliasKey{aliasName: aliasName, login: login}], nil
}

func (r *stubRepository) CreateCanonicalPlayer(_ context.Context, login string, displayName string, createdAt time.Time) (string, error) {
	if r.createErr != nil {
		return "", r.createErr
	}
	r.createdPlayers = append(r.createdPlayers, createdPlayer{
		login:       login,
		displayName: displayName,
		createdAt:   createdAt,
	})
	return r.createdPlayerID, nil
}

func (r *stubRepository) PromotePrimaryLogin(_ context.Context, playerID string, login string, _ time.Time) error {
	if r.promoteErr != nil {
		return r.promoteErr
	}
	r.promotions = append(r.promotions, promotion{playerID: playerID, login: login})
	return nil
}

func (r *stubRepository) UpsertAlias(_ context.Context, playerID string, aliasName string, login string, observedAt time.Time) (bool, error) {
	if r.upsertAliasErr != nil {
		return false, r.upsertAliasErr
	}
	r.aliasUpserts = append(r.aliasUpserts, aliasUpsert{
		playerID:   playerID,
		aliasName:  aliasName,
		login:      login,
		observedAt: observedAt,
	})
	if r.upsertAliasInserted == nil {
		return true, nil
	}
	if !*r.upsertAliasInserted {
		return false, nil
	}
	return true, nil
}
