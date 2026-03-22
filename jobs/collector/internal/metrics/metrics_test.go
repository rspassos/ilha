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
	collector.RecordServerRun("qlash-br-1", "success")
	collector.RecordMatchesFetched("qlash-br-1", "lastscores", 10)
	collector.RecordMatchesUpserted("qlash-br-1", "inserted", 7)
	collector.RecordMergeWarning("qlash-br-1", "mode_mismatch")
	collector.ObserveRequest("qlash-br-1", "lastscores", time.Now().Add(-500*time.Millisecond))

	if value := testutil.ToFloat64(collector.runsTotal.WithLabelValues("success")); value != 1 {
		t.Fatalf("collector_runs_total = %v, want 1", value)
	}
	if value := testutil.ToFloat64(collector.serverRunsTotal.WithLabelValues("qlash-br-1", "success")); value != 1 {
		t.Fatalf("collector_server_runs_total = %v, want 1", value)
	}
	if value := testutil.ToFloat64(collector.matchesFetchedTotal.WithLabelValues("qlash-br-1", "lastscores")); value != 10 {
		t.Fatalf("collector_matches_fetched_total = %v, want 10", value)
	}
	if value := testutil.ToFloat64(collector.matchesUpsertedTotal.WithLabelValues("qlash-br-1", "inserted")); value != 7 {
		t.Fatalf("collector_matches_upserted_total = %v, want 7", value)
	}
	if value := testutil.ToFloat64(collector.mergeWarningsTotal.WithLabelValues("qlash-br-1", "mode_mismatch")); value != 1 {
		t.Fatalf("collector_merge_warnings_total = %v, want 1", value)
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
	if !strings.Contains(body, "collector_runs_total") {
		t.Fatalf("metrics output missing collector_runs_total:\n%s", body)
	}
}
