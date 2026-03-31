package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCollectorHandlerExposesPrometheusMetrics(t *testing.T) {
	t.Parallel()

	collector := New()
	collector.RecordRequest("/v1/rankings/players", "200")
	collector.ObserveRequest("/v1/rankings/players", time.Now().Add(-250*time.Millisecond))
	collector.RecordRankingRowsReturned(3)
	collector.RecordInvalidRequest("sort_by")
	collector.RecordDBQuery("list_player_ranking", "success")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()

	collector.Handler().ServeHTTP(recorder, req)

	if got := recorder.Code; got != http.StatusOK {
		t.Fatalf("status = %d, want %d", got, http.StatusOK)
	}

	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}

	output := string(body)
	for _, metricName := range []string{
		"player_stats_api_requests_total",
		"player_stats_api_request_duration_seconds",
		"player_stats_api_ranking_rows_returned_total",
		"player_stats_api_invalid_requests_total",
		"player_stats_api_db_queries_total",
	} {
		if !strings.Contains(output, metricName) {
			t.Fatalf("metrics output missing %q", metricName)
		}
	}
}
