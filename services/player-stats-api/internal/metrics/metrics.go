package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Collector struct {
	registry            *prometheus.Registry
	requestsTotal       *prometheus.CounterVec
	requestDuration     *prometheus.HistogramVec
	rankingRowsReturned prometheus.Counter
	invalidRequests     *prometheus.CounterVec
	dbQueriesTotal      *prometheus.CounterVec
}

func New() *Collector {
	registry := prometheus.NewRegistry()

	collector := &Collector{
		registry: registry,
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "player_stats_api_requests_total",
			Help: "Total number of HTTP requests handled by the player stats API.",
		}, []string{"endpoint", "status"}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "player_stats_api_request_duration_seconds",
			Help:    "Duration of HTTP requests handled by the player stats API.",
			Buckets: prometheus.DefBuckets,
		}, []string{"endpoint"}),
		rankingRowsReturned: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "player_stats_api_ranking_rows_returned_total",
			Help: "Total number of ranking rows returned by the player stats API.",
		}),
		invalidRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "player_stats_api_invalid_requests_total",
			Help: "Total number of invalid requests rejected by the player stats API.",
		}, []string{"reason"}),
		dbQueriesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "player_stats_api_db_queries_total",
			Help: "Total number of database queries executed by the player stats API.",
		}, []string{"query", "status"}),
	}

	registry.MustRegister(
		collector.requestsTotal,
		collector.requestDuration,
		collector.rankingRowsReturned,
		collector.invalidRequests,
		collector.dbQueriesTotal,
	)

	return collector
}

func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{})
}

func (c *Collector) RecordRequest(endpoint string, status string) {
	c.requestsTotal.WithLabelValues(endpoint, status).Inc()
}

func (c *Collector) ObserveRequest(endpoint string, startedAt time.Time) {
	c.requestDuration.WithLabelValues(endpoint).Observe(time.Since(startedAt).Seconds())
}

func (c *Collector) RecordRankingRowsReturned(count int) {
	c.rankingRowsReturned.Add(float64(count))
}

func (c *Collector) RecordInvalidRequest(reason string) {
	c.invalidRequests.WithLabelValues(reason).Inc()
}

func (c *Collector) RecordDBQuery(query string, status string) {
	c.dbQueriesTotal.WithLabelValues(query, status).Inc()
}
