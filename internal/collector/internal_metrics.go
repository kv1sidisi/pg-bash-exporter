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

	// ConfigReloads shows number of successful config reloads.
	ConfigReloads prometheus.Counter

	// ConfigReloadErrors shows number of failed config reloads.
	ConfigReloadErrors prometheus.Counter

	// CommandDuration shows duration of each command execution.
	CommandDuration *prometheus.HistogramVec

	// ConcurrentCommands shows number of concurrently running commands.
	ConcurrentCommands prometheus.Gauge
)

func init() {
	Checks = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pg_bash_exporter_checks_total",
		Help: "Number of metrics checks.",
	})

	CheckDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "pg_bash_exporter_check_duration_seconds",
		Help: "How much time tom take metrics.",
	})

	CommandErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pg_bash_exporter_command_errors_total",
		Help: "Number of command errors.",
	}, []string{"metric_name"})

	CacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pg_bash_exporter_cache_hits_total",
		Help: "Number of cache hits.",
	})

	CacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pg_bash_exporter_cache_misses_total",
		Help: "Number of cache misses.",
	})

	ConfigReloads = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pg_bash_exporter_config_reloads_total",
		Help: "Number of successful config reloads.",
	})

	ConfigReloadErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pg_bash_exporter_config_reload_errors_total",
		Help: "Number of failed config reloads.",
	})

	CommandDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "pg_bash_exporter_command_duration_seconds",
		Help: "Duration of each command execution.",
	}, []string{"metric_name"})

	ConcurrentCommands = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pg_bash_exporter_concurrent_commands",
		Help: "Number of concurrently running commands.",
	})
}
