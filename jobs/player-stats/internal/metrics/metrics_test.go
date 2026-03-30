package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCollectorRecordsExpectedMetrics(t *testing.T) {
	t.Parallel()

	collector := New()
	collector.RecordRun("success")
	collector.RecordMatchesScanned("processed", 10)
	collector.RecordPlayerRowsUpserted("inserted", 24)
	collector.RecordIdentityResolution("created", 3)
	collector.RecordSkippedMatch("missing_stats_payload")
	collector.ObserveStage("bootstrap", time.Now().Add(-500*time.Millisecond))

	if value := testutil.ToFloat64(collector.runsTotal.WithLabelValues("success")); value != 1 {
		t.Fatalf("player_stats_runs_total = %v, want 1", value)
	}
	if value := testutil.ToFloat64(collector.matchesScannedTotal.WithLabelValues("processed")); value != 10 {
		t.Fatalf("player_stats_matches_scanned_total = %v, want 10", value)
	}
	if value := testutil.ToFloat64(collector.playerRowsUpsertedTotal.WithLabelValues("inserted")); value != 24 {
		t.Fatalf("player_stats_player_rows_upserted_total = %v, want 24", value)
	}
	if value := testutil.ToFloat64(collector.identityResolutionsTotal.WithLabelValues("created")); value != 3 {
		t.Fatalf("player_stats_identity_resolutions_total = %v, want 3", value)
	}
	if value := testutil.ToFloat64(collector.skippedMatchesTotal.WithLabelValues("missing_stats_payload")); value != 1 {
		t.Fatalf("player_stats_skipped_matches_total = %v, want 1", value)
	}
}

func TestCollectorExposesPrometheusHandler(t *testing.T) {
	t.Parallel()

	collector := New()
	collector.RecordRun("success")

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()
	collector.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "player_stats_runs_total") {
		t.Fatalf("metrics output missing player_stats_runs_total:\n%s", body)
	}
}
