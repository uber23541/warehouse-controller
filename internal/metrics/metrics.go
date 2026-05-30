package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests processed, partitioned by method, path and status code.",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds, partitioned by method, path and status code.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	HTTPRequestsErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_errors_total",
		Help: "Total number of HTTP requests that resulted in a 5xx response.",
	}, []string{"method", "path", "status"})

	HTTPRequestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Number of HTTP requests currently being served.",
	})

	CacheHitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "Total number of cache hits, partitioned by operation.",
	}, []string{"operation"})

	CacheMissesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "Total number of cache misses, partitioned by operation.",
	}, []string{"operation"})

	DBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "Duration of database queries in seconds, partitioned by operation.",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})
)

func CacheHit(operation string) {
	CacheHitsTotal.WithLabelValues(operation).Inc()
}

func CacheMiss(operation string) {
	CacheMissesTotal.WithLabelValues(operation).Inc()
}

func ObserveDBQuery(operation string) func() {
	start := time.Now()
	return func() {
		DBQueryDuration.WithLabelValues(operation).Observe(time.Since(start).Seconds())
	}
}
