package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Collector struct {
	registry                  *prometheus.Registry
	runsTotal                 *prometheus.CounterVec
	matchesScannedTotal       *prometheus.CounterVec
	playerRowsUpsertedTotal   *prometheus.CounterVec
	identityResolutionsTotal  *prometheus.CounterVec
	processingDurationSeconds *prometheus.HistogramVec
	skippedMatchesTotal       *prometheus.CounterVec
}

func New() *Collector {
	registry := prometheus.NewRegistry()

	collector := &Collector{
		registry: registry,
		runsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "player_stats_runs_total",
			Help: "Total number of player stats run-once executions.",
		}, []string{"status"}),
		matchesScannedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "player_stats_matches_scanned_total",
			Help: "Total number of matches scanned by result.",
		}, []string{"result"}),
		playerRowsUpsertedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "player_stats_player_rows_upserted_total",
			Help: "Total number of player rows upserted by result.",
		}, []string{"result"}),
		identityResolutionsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "player_stats_identity_resolutions_total",
			Help: "Total number of player identity resolutions by result.",
		}, []string{"result"}),
		processingDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "player_stats_processing_duration_seconds",
			Help:    "Duration of player stats processing by stage.",
			Buckets: prometheus.DefBuckets,
		}, []string{"stage"}),
		skippedMatchesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "player_stats_skipped_matches_total",
			Help: "Total number of skipped matches by reason.",
		}, []string{"reason"}),
	}

	registry.MustRegister(
		collector.runsTotal,
		collector.matchesScannedTotal,
		collector.playerRowsUpsertedTotal,
		collector.identityResolutionsTotal,
		collector.processingDurationSeconds,
		collector.skippedMatchesTotal,
	)

	return collector
}

func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{})
}

func (c *Collector) RecordRun(status string) {
	c.runsTotal.WithLabelValues(status).Inc()
}

func (c *Collector) RecordMatchesScanned(result string, count int) {
	c.matchesScannedTotal.WithLabelValues(result).Add(float64(count))
}

func (c *Collector) RecordPlayerRowsUpserted(result string, count int) {
	c.playerRowsUpsertedTotal.WithLabelValues(result).Add(float64(count))
}

func (c *Collector) RecordIdentityResolution(result string, count int) {
	c.identityResolutionsTotal.WithLabelValues(result).Add(float64(count))
}

func (c *Collector) ObserveStage(stage string, startedAt time.Time) {
	c.processingDurationSeconds.WithLabelValues(stage).Observe(time.Since(startedAt).Seconds())
}

func (c *Collector) RecordSkippedMatch(reason string) {
	c.skippedMatchesTotal.WithLabelValues(reason).Inc()
}
