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
	config     *config.Config
	logger     *slog.Logger
	executor   Executor
	cache      *cache.Cache
	configPath string

	mu sync.RWMutex
}

func NewCollector(cfg *config.Config, logger *slog.Logger, exec Executor, cache *cache.Cache, configPath string) *Collector {
	return &Collector{
		config:     cfg,
		logger:     logger,
		executor:   exec,
		cache:      cache,
		configPath: configPath,
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, metricConfig := range c.config.Metrics {
		if len(metricConfig.SubMetrics) == 0 {
			dynLblNames := getLabelNames(metricConfig.DynamicLabels)

			desc := prometheus.NewDesc(
				metricConfig.Name,
				metricConfig.Help,
				dynLblNames,
				metricConfig.Labels)

			ch <- desc
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
		}
	}
	c.logger.Debug("metric description reading ended")
}

func (c *Collector) ReloadConfig() error {
	var newCfg config.Config

	if err := config.Load(c.configPath, &newCfg); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.config = &newCfg

	c.logger.Info("config reloaded successfully")
	return nil
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	start := time.Now()

	defer func() {
		CheckDuration.Observe(time.Since(start).Seconds())
	}()

	Checks.Inc()

	c.logger.Info("Metrics collection started")

	wg := sync.WaitGroup{}

	maxConcurrent := c.config.Global.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = config.DefaultMaxConcurrent
	}
	smph := make(chan struct{}, maxConcurrent)

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
