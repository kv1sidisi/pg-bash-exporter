package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"pg-bash-exporter/internal/config"
)

type Collector struct {
	config *config.Config
	logger *slog.Logger
}

func NewCollector(cfg *config.Config, logger *slog.Logger) *Collector {
	return &Collector{
		config: cfg,
		logger: logger,
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("base", "Exporter build info", nil, nil)
}
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.logger.Info("Metric collection started")
}
