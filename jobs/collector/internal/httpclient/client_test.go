package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rspassos/ilha/jobs/collector/internal/config"
)

func TestFetchLastScoresDecodesFixture(t *testing.T) {
	t.Parallel()

	server := newFixtureServer(t)
	client := New(server.URL, nil)

	matches, err := client.FetchLastScores(context.Background(), config.ServerConfig{
		Key:            "qlash-br-1",
		Address:        "qlash-br-1:27500",
		TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatalf("FetchLastScores() error = %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2", len(matches))
	}
	if matches[0].Demo != "duel_expert_vs_matuzah[aerowalk]20260319-2013.mvd" {
		t.Fatalf("matches[0].Demo = %q", matches[0].Demo)
	}
	if matches[0].PlayedAt.IsZero() {
		t.Fatal("matches[0].PlayedAt is zero")
	}
	if len(matches[0].Players) != 2 {
		t.Fatalf("len(matches[0].Players) = %d, want 2", len(matches[0].Players))
	}
	if len(matches[1].Teams) != 2 {
		t.Fatalf("len(matches[1].Teams) = %d, want 2", len(matches[1].Teams))
	}
	if matches[1].Teams[0].Players[0].Name != "MatuzaH" {
		t.Fatalf("matches[1].Teams[0].Players[0].Name = %q", matches[1].Teams[0].Players[0].Name)
	}
}

func TestFetchLastStatsDecodesFixture(t *testing.T) {
	t.Parallel()

	server := newFixtureServer(t)
	client := New(server.URL, nil)

	matches, err := client.FetchLastStats(context.Background(), config.ServerConfig{
		Key:            "qlash-br-1",
		Address:        "qlash-br-1:27500",
		TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatalf("FetchLastStats() error = %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2", len(matches))
	}
	if matches[0].Mode != "duel" {
		t.Fatalf("matches[0].Mode = %q, want duel", matches[0].Mode)
	}
	if matches[1].Mode != "team" {
		t.Fatalf("matches[1].Mode = %q, want team", matches[1].Mode)
	}
	if len(matches[1].Teams) != 2 {
		t.Fatalf("len(matches[1].Teams) = %d, want 2", len(matches[1].Teams))
	}
	if matches[1].Players[0].Damage["team"] != 267 {
		t.Fatalf("matches[1].Players[0].Damage[team] = %d, want 267", matches[1].Players[0].Damage["team"])
	}
	if len(matches[1].Players[0].Weapons) == 0 {
		t.Fatal("matches[1].Players[0].Weapons is empty")
	}
}

func TestFetchLastScoresReturnsDecodeError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"invalid":true}`))
	}))
	t.Cleanup(server.Close)

	client := New(server.URL, nil)
	_, err := client.FetchLastScores(context.Background(), config.ServerConfig{
		Key:            "bad-server",
		Address:        "bad-server:27500",
		TimeoutSeconds: 5,
	})
	if err == nil {
		t.Fatal("FetchLastScores() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `server "bad-server" lastscores: decode response`) {
		t.Fatalf("FetchLastScores() error = %v", err)
	}
}

func TestFetchLastStatsReturnsStatusError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream exploded", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)

	client := New(server.URL, nil)
	_, err := client.FetchLastStats(context.Background(), config.ServerConfig{
		Key:            "bad-gateway",
		Address:        "bad-gateway:27500",
		TimeoutSeconds: 5,
	})
	if err == nil {
		t.Fatal("FetchLastStats() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `unexpected status 502`) {
		t.Fatalf("FetchLastStats() error = %v", err)
	}
}

func TestFetchLastScoresHonorsPerRequestTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(1200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	client := New(server.URL, nil)
	_, err := client.FetchLastScores(context.Background(), config.ServerConfig{
		Key:            "slow-server",
		Address:        "slow-server:27500",
		TimeoutSeconds: 1,
	})
	if err == nil {
		t.Fatal("FetchLastScores() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "request timed out after 1s") {
		t.Fatalf("FetchLastScores() error = %v", err)
	}
}

func TestFetchLastStatsWrapsTransportErrors(t *testing.T) {
	t.Parallel()

	client := New("http://example.com", &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed")
		}),
	})

	_, err := client.FetchLastStats(context.Background(), config.ServerConfig{
		Key:            "offline-server",
		Address:        "offline-server:27500",
		TimeoutSeconds: 5,
	})
	if err == nil {
		t.Fatal("FetchLastStats() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `server "offline-server" laststats: send request:`) || !strings.Contains(err.Error(), "dial failed") {
		t.Fatalf("FetchLastStats() error = %v", err)
	}
}

func TestNewUsesDefaultTransportSettings(t *testing.T) {
	t.Parallel()

	client := New("http://example.com", nil)

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.httpClient.Transport type = %T, want *http.Transport", client.httpClient.Transport)
	}
	if transport.ForceAttemptHTTP2 != http.DefaultTransport.(*http.Transport).ForceAttemptHTTP2 {
		t.Fatalf(
			"client.httpClient.Transport.ForceAttemptHTTP2 = %v, want default %v",
			transport.ForceAttemptHTTP2,
			http.DefaultTransport.(*http.Transport).ForceAttemptHTTP2,
		)
	}
	if client.httpClient.CheckRedirect == nil {
		t.Fatal("client.httpClient.CheckRedirect = nil, want non-nil")
	}
}

func TestFetchLastScoresReturnsTimeoutWhenBodyReadExceedsDeadline(t *testing.T) {
	t.Parallel()

	client := New("http://example.com", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: &blockingReadCloser{
					ctx: req.Context(),
				},
			}, nil
		}),
	})

	_, err := client.FetchLastScores(context.Background(), config.ServerConfig{
		Key:            "slow-body-server",
		Address:        "slow-body-server:27500",
		TimeoutSeconds: 1,
	})
	if err == nil {
		t.Fatal("FetchLastScores() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "request timed out after 1s") {
		t.Fatalf("FetchLastScores() error = %v", err)
	}
}

func TestFetchLastScoresSetsRequestHeaders(t *testing.T) {
	t.Parallel()

	client := New("http://example.com", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("Accept"); got != "application/json" {
				t.Fatalf("Accept header = %q, want application/json", got)
			}
			if got := req.Header.Get("User-Agent"); got != defaultUserAgent {
				t.Fatalf("User-Agent header = %q, want %q", got, defaultUserAgent)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`[]`)),
			}, nil
		}),
	})

	_, err := client.FetchLastScores(context.Background(), config.ServerConfig{
		Key:            "header-server",
		Address:        "header-server:27500",
		TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatalf("FetchLastScores() error = %v", err)
	}
}

func TestBuildURLPreservesBasePath(t *testing.T) {
	t.Parallel()

	got, err := buildURL("https://hubapi.quakeworld.nu", "qw.qlash.com.br:28501", "lastscores")
	if err != nil {
		t.Fatalf("buildURL() error = %v", err)
	}

	want := "https://hubapi.quakeworld.nu/v2/servers/qw.qlash.com.br:28501/lastscores"
	if got != want {
		t.Fatalf("buildURL() = %q, want %q", got, want)
	}
}

func newFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()

	lastScores := mustReadFixture(t, "lastscores.json")
	lastStats := mustReadFixture(t, "laststats.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/servers/qlash-br-1:27500/lastscores":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(lastScores)
		case "/v2/servers/qlash-br-1:27500/laststats":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(lastStats)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return data
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type blockingReadCloser struct {
	ctx context.Context
}

func (r *blockingReadCloser) Read(_ []byte) (int, error) {
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

func (r *blockingReadCloser) Close() error {
	return nil
}

var _ io.ReadCloser = (*blockingReadCloser)(nil)
