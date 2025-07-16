package collector

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"pg-bash-exporter/internal/cache"
	"pg-bash-exporter/internal/config"
	"sync"
	"time"
)

type Executor interface {
	ExecuteCommand(ctx context.Context, command string, timeout time.Duration) (string, error)
}

type Collector struct {
	config   *config.Config
	logger   *slog.Logger
	executor Executor
	cache    *cache.Cache
}

func NewCollector(cfg *config.Config, logger *slog.Logger, exec Executor, cache *cache.Cache) *Collector {
	return &Collector{
		config:   cfg,
		logger:   logger,
		executor: exec,
		cache:    cache,
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

	wg := sync.WaitGroup{}

	smph := make(chan struct{}, c.config.Global.MaxConcurrent)

	for _, metricConfig := range c.config.Metrics {
		wg.Add(1)

		go func(mc config.Metric) {
			defer wg.Done()

			smph <- struct{}{}
			defer func() {
				<-smph
			}()

			if len(mc.SubMetrics) == 0 {
				c.collectSimpleMetric(ch, mc)
			} else {
				c.collectComplicatedMetric(ch, mc)
			}
		}(metricConfig)
	}

	wg.Wait()

	c.logger.Info("Metrics collection finished")
}
