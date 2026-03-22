package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/rspassos/ilha/jobs/collector/internal/config"
	"github.com/rspassos/ilha/jobs/collector/internal/model"
)

const maxErrorBody = 256
const defaultUserAgent = "match-stats-collector/1.0"

type Client struct {
	hubAPIBaseURL string
	httpClient    *http.Client
}

func New(hubAPIBaseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Transport:     defaultTransport(),
			CheckRedirect: stopRedirects,
		}
	}
	return &Client{
		hubAPIBaseURL: hubAPIBaseURL,
		httpClient:    httpClient,
	}
}

func (c *Client) FetchLastScores(ctx context.Context, server config.ServerConfig) ([]model.ScoreMatch, error) {
	matches, err := fetchAndDecode[model.ScoreMatch](ctx, c.hubAPIBaseURL, c.httpClient, server, "lastscores")
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func (c *Client) FetchLastStats(ctx context.Context, server config.ServerConfig) ([]model.StatsMatch, error) {
	matches, err := fetchAndDecode[model.StatsMatch](ctx, c.hubAPIBaseURL, c.httpClient, server, "laststats")
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func fetchAndDecode[T any](ctx context.Context, hubAPIBaseURL string, httpClient *http.Client, server config.ServerConfig, endpointName string) ([]T, error) {
	requestURL, err := buildURL(hubAPIBaseURL, server.Address, endpointName)
	if err != nil {
		return nil, fmt.Errorf("server %q %s: build request url: %w", server.Key, endpointName, err)
	}

	requestCtx := ctx
	var cancel context.CancelFunc
	if server.TimeoutSeconds > 0 {
		requestCtx, cancel = context.WithTimeout(ctx, time.Duration(server.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("server %q %s: create request: %w", server.Key, endpointName, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		if errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("server %q %s: request timed out after %ds: %w", server.Key, endpointName, server.TimeoutSeconds, requestCtx.Err())
		}
		return nil, fmt.Errorf("server %q %s: send request: %w", server.Key, endpointName, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("server %q %s: request timed out after %ds: %w", server.Key, endpointName, server.TimeoutSeconds, requestCtx.Err())
		}
		return nil, fmt.Errorf("server %q %s: read response body: %w", server.Key, endpointName, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server %q %s: unexpected status %d: %s", server.Key, endpointName, resp.StatusCode, shortenBody(body))
	}

	var matches []T
	if err := json.Unmarshal(body, &matches); err != nil {
		return nil, fmt.Errorf("server %q %s: decode response: %w", server.Key, endpointName, err)
	}

	switch typed := any(matches).(type) {
	case []model.ScoreMatch:
		for index := range typed {
			if err := typed[index].Normalize(); err != nil {
				return nil, fmt.Errorf("server %q %s: normalize match %d: %w", server.Key, endpointName, index, err)
			}
		}
		matches = any(typed).([]T)
	case []model.StatsMatch:
		for index := range typed {
			if err := typed[index].Normalize(); err != nil {
				return nil, fmt.Errorf("server %q %s: normalize match %d: %w", server.Key, endpointName, index, err)
			}
		}
		matches = any(typed).([]T)
	}

	return matches, nil
}

func buildURL(baseURL string, address string, endpoint string) (string, error) {
	parsedBaseURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("parse hubapi_base_url: %w", err)
	}
	if parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" {
		return "", fmt.Errorf("hubapi_base_url %q must include scheme and host", baseURL)
	}
	trimmedAddress := strings.Trim(strings.TrimSpace(address), "/")
	if trimmedAddress == "" {
		return "", errors.New("address must not be empty")
	}
	parsedBaseURL.Path = path.Join(parsedBaseURL.Path, "v2", "servers", trimmedAddress, endpoint)
	return parsedBaseURL.String(), nil
}

func shortenBody(body []byte) string {
	value := strings.TrimSpace(string(body))
	if value == "" {
		return "<empty body>"
	}
	if len(value) <= maxErrorBody {
		return value
	}
	return value[:maxErrorBody] + "..."
}

func defaultTransport() http.RoundTripper {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	return transport
}

func stopRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
