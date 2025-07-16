package collector

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"pg-bash-exporter/internal/config"
	"time"
)

type Executor interface {
	ExecuteCommand(ctx context.Context, command string, timeout time.Duration) (string, error)
}

type cache struct {
	output    string
	timestamp time.Time
}

type Collector struct {
	config   *config.Config
	logger   *slog.Logger
	executor Executor
	cache    map[string]cache
}

func NewCollector(cfg *config.Config, logger *slog.Logger, exec Executor) *Collector {
	return &Collector{
		config:   cfg,
		logger:   logger,
		executor: exec,
		cache:    make(map[string]cache),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, metricConfig := range c.config.Metrics {
		if len(metricConfig.SubMetrics) == 0 {
			dynLblNames := getLabelNames(metricConfig.DynamicLabels)

			desc := prometheus.NewDesc(
				metricConfig.Name,
				metricConfig.Help,
				dynLblNames,
				metricConfig.Labels)

			ch <- desc
			c.logger.Debug("simple metric description added", "metric", metricConfig.Name)
			continue
		}
		for _, subMetric := range metricConfig.SubMetrics {
			labels := mergeLabels(metricConfig.Labels, subMetric.Labels)

			dynLblNames := getLabelNames(subMetric.DynamicLabels)

			fullName := metricConfig.Name + "_" + subMetric.Name

			desc := prometheus.NewDesc(
				fullName,
				subMetric.Help,
				dynLblNames,
				labels,
			)
			ch <- desc
			c.logger.Debug("sub-metric description added", "sub-metric", subMetric.Name)
		}
	}

}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.logger.Info("Metrics collection started")

	for _, metricConfig := range c.config.Metrics {
		if len(metricConfig.SubMetrics) == 0 {
			c.collectSimpleMetric(ch, metricConfig)
		} else {
			c.collectComplicatedMetric(ch, metricConfig)
		}
	}

	c.logger.Info("Metrics collection finished")
}
