package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Collector struct {
	registry             *prometheus.Registry
	runsTotal            *prometheus.CounterVec
	serverRunsTotal      *prometheus.CounterVec
	matchesFetchedTotal  *prometheus.CounterVec
	matchesUpsertedTotal *prometheus.CounterVec
	mergeWarningsTotal   *prometheus.CounterVec
	requestDuration      *prometheus.HistogramVec
}

func New() *Collector {
	registry := prometheus.NewRegistry()

	collector := &Collector{
		registry: registry,
		runsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "collector_runs_total",
			Help: "Total number of collector run-once executions.",
		}, []string{"status"}),
		serverRunsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "collector_server_runs_total",
			Help: "Total number of collector server executions.",
		}, []string{"server_key", "status"}),
		matchesFetchedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "collector_matches_fetched_total",
			Help: "Total number of matches fetched from upstream endpoints.",
		}, []string{"server_key", "source"}),
		matchesUpsertedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "collector_matches_upserted_total",
			Help: "Total number of matches inserted or updated in storage.",
		}, []string{"server_key", "result"}),
		mergeWarningsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "collector_merge_warnings_total",
			Help: "Total number of merge warnings emitted by reason.",
		}, []string{"server_key", "reason"}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "collector_request_duration_seconds",
			Help:    "Duration of upstream HTTP requests per server and endpoint.",
			Buckets: prometheus.DefBuckets,
		}, []string{"server_key", "endpoint"}),
	}

	registry.MustRegister(
		collector.runsTotal,
		collector.serverRunsTotal,
		collector.matchesFetchedTotal,
		collector.matchesUpsertedTotal,
		collector.mergeWarningsTotal,
		collector.requestDuration,
	)

	return collector
}

func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{})
}

func (c *Collector) Registry() *prometheus.Registry {
	return c.registry
}

func (c *Collector) RecordRun(status string) {
	c.runsTotal.WithLabelValues(status).Inc()
}

func (c *Collector) RecordServerRun(serverKey string, status string) {
	c.serverRunsTotal.WithLabelValues(serverKey, status).Inc()
}

func (c *Collector) RecordMatchesFetched(serverKey string, source string, count int) {
	c.matchesFetchedTotal.WithLabelValues(serverKey, source).Add(float64(count))
}

func (c *Collector) RecordMatchesUpserted(serverKey string, result string, count int) {
	c.matchesUpsertedTotal.WithLabelValues(serverKey, result).Add(float64(count))
}

func (c *Collector) RecordMergeWarning(serverKey string, reason string) {
	c.mergeWarningsTotal.WithLabelValues(serverKey, reason).Inc()
}

func (c *Collector) ObserveRequest(serverKey string, endpoint string, startedAt time.Time) {
	c.requestDuration.WithLabelValues(serverKey, endpoint).Observe(time.Since(startedAt).Seconds())
}
