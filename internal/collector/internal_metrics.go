package collector

import "github.com/prometheus/client_golang/prometheus"

var (
	// Checks shows how many times Prometheus checked metrics.
	Checks prometheus.Counter

	// CheckDuration shows time to check metrics
	CheckDuration prometheus.Histogram

	// CommandErrors shows number of errors in every metric.
	CommandErrors *prometheus.CounterVec

	// CacheHits shows number of times cache was used.
	CacheHits prometheus.Counter

	// CacheMisses shows number of times cache was not used.
	CacheMisses prometheus.Counter
)

func init() {
	Checks = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pg_bash_exporter_checks.",
		Help: "Number of metrics checks.",
	})

	CheckDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "pg_bash_exporter_check_duration_seconds",
		Help: "How much time tom take metrics.",
	})

	CommandErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pg_bash_exporter_command_errors.",
		Help: "Number of command errors.",
	}, []string{"metric_name"})

	CacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pg_bash_exporter_cache_hits",
		Help: "Number of cache hits.",
	})

	CacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pg_bash_exporter_cache_misses",
		Help: "Number of cache misses.",
	})
}
