package collector

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"io"
	"log/slog"
	"os"
	"pg-bash-exporter/internal/cache"
	"pg-bash-exporter/internal/config"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// mockExecutor is a mock implementation of the Executor interface for testing.
type mockExecutor struct {
	output string
	err    error
}

// ExecuteCommand returns mock output and error.
func (m *mockExecutor) ExecuteCommand(ctx context.Context, command string, timeout time.Duration) (string, error) {
	return m.output, m.err
}

func TestCollect(t *testing.T) {
	testCases := []struct {
		name           string
		config         *config.Config
		executor       *mockExecutor
		expectedMetric string
	}{
		{
			name: "simple gauge metric",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "simple_metric",
						Help:    "Simple metric.",
						Type:    "gauge",
						Command: "echo 123",
					},
				},
				Global: config.Global{
					Timeout: 0,
				},
			},
			executor: &mockExecutor{
				output: "123",
				err:    nil,
			},
			expectedMetric: `
# HELP simple_metric Simple metric.
# TYPE simple_metric gauge
simple_metric 123
`,
		},
		{
			name: "gauge metric with postfix-metrics",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "postfix_metric",
						Help:    "A metric with postfix-metrics.",
						Type:    "gauge",
						Command: "echo '10 20'",
						PostfixMetrics: []config.PostfixMetric{
							{
								Name:  "postfix_one",
								Help:  "First postfix-metric.",
								Type:  "gauge",
								Field: 0,
							},
							{
								Name:  "postfix_two",
								Help:  "Second postfix-metric.",
								Type:  "gauge",
								Field: 1,
							},
						},
					},
				},
				Global: config.Global{
					Timeout: 0,
				},
			},
			executor: &mockExecutor{
				output: "10 20",
				err:    nil,
			},
			expectedMetric: `
# HELP postfix_metric_postfix_one First postfix-metric.
# TYPE postfix_metric_postfix_one gauge
postfix_metric_postfix_one 10
# HELP postfix_metric_postfix_two Second postfix-metric.
# TYPE postfix_metric_postfix_two gauge
postfix_metric_postfix_two 20
`,
		},
		{
			name: "command execution error",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "error_metric",
						Help:    "A metric that fails.",
						Type:    "gauge",
						Command: "exit 1",
					},
				},
				Global: config.Global{
					Timeout: 0,
				},
			},
			executor: &mockExecutor{
				output: "",
				err:    errors.New("command failed"),
			},
			expectedMetric: `
`,
		},
		{
			name: "metric with dynamic labels",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "dynamic_labels_metric",
						Help:    "A metric with dynamic labels.",
						Type:    "gauge",
						Command: "echo 'label_val1 10' && echo 'label_val2 20'",
						PostfixMetrics: []config.PostfixMetric{
							{
								Name:  "dynamic_postfix_metric",
								Help:  "A postfix-metric with dynamic labels.",
								Type:  "gauge",
								Field: 1,
								DynamicLabels: []config.DynamicLabel{
									{Name: "my_label", Field: 0},
								},
							},
						},
					},
				},
				Global: config.Global{
					Timeout: 0,
				},
			},
			executor: &mockExecutor{
				output: "label_val1 10\nlabel_val2 20",
				err:    nil,
			},
			expectedMetric: `
# HELP dynamic_labels_metric_dynamic_postfix_metric A postfix-metric with dynamic labels.
# TYPE dynamic_labels_metric_dynamic_postfix_metric gauge
dynamic_labels_metric_dynamic_postfix_metric{my_label="label_val1"} 10
dynamic_labels_metric_dynamic_postfix_metric{my_label="label_val2"} 20
`,
		},
		{
			name: "postfix-metric with pattern match",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "matched_metric",
						Help:    "metric with matched postfix-metrics.",
						Command: "echo -e 'CPU label1 100\\MEM label2 200'",
						PostfixMetrics: []config.PostfixMetric{
							{
								Name:  "cpu",
								Help:  "Metric for cpu.",
								Type:  "gauge",
								Field: 2,
								Match: "^CPU",
								DynamicLabels: []config.DynamicLabel{
									{Name: "label_name", Field: 1},
								},
							},
							{
								Name:  "mem",
								Help:  "Metric for mem.",
								Type:  "gauge",
								Field: 2,
								Match: "^MEM",
								DynamicLabels: []config.DynamicLabel{
									{Name: "label_name", Field: 1},
								},
							},
						},
					},
				},
				Global: config.Global{
					Timeout: 0,
				},
			},
			executor: &mockExecutor{
				output: "CPU label1 100\nMEM label2 200",
				err:    nil,
			},
			expectedMetric: `
# HELP matched_metric_cpu Metric for cpu.
# TYPE matched_metric_cpu gauge
matched_metric_cpu{label_name="label1"} 100
# HELP matched_metric_mem Metric for mem.
# TYPE matched_metric_mem gauge
matched_metric_mem{label_name="label2"} 200
`,
		},
		{
			name: "command timeout",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "timeout_metric",
						Help:    "metric that times out.",
						Type:    "gauge",
						Command: "sleep 5",
						Timeout: 1 * time.Millisecond,
					},
				},
				Global: config.Global{
					Timeout: 1 * time.Second,
				},
			},
			executor: &mockExecutor{
				output: "",
				err:    context.DeadlineExceeded,
			},
			expectedMetric: ``,
		},
		{
			name: "simple metric with dynamic labels and multi-line output",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "connections",
						Help:    "number of connetions.",
						Type:    "gauge",
						Command: "echo -e 'tcp 150\nudp 25'",
						Field:   1,
						DynamicLabels: []config.DynamicLabel{
							{Name: "type", Field: 0},
						},
					},
				},
				Global: config.Global{
					Timeout: 0,
				},
			},
			executor: &mockExecutor{
				output: "tcp 150\nudp 25",
				err:    nil,
			},
			expectedMetric: `
# HELP connections number of connetions.
# TYPE connections gauge
connections{type="tcp"} 150
connections{type="udp"} 25
`,
		},
		{
			name: "value parsing failure",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "parse_fail_metric",
						Help:    "metric that fails to parse.",
						Type:    "gauge",
						Command: "echo 'not_number'",
					},
				},
			},
			executor: &mockExecutor{
				output: "not_number",
			},
			expectedMetric: ``,
		},
		{
			name: "field index out of range",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "index_metric",
						Help:    "metric where field index is out of range.",
						Type:    "gauge",
						Command: "echo 'one_value'",
						Field:   1,
					},
				},
			},
			executor: &mockExecutor{
				output: "one_value",
			},
			expectedMetric: ``,
		},
		{
			name: "postfix-metric with no matching pattern",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "filter_metric",
						Help:    "metric with a postfix-metric that filters lines.",
						Command: "echo -e 'matched_line 100\nunmatched_line 200'",
						PostfixMetrics: []config.PostfixMetric{
							{
								Name:  "filtered_sub",
								Help:  "created for matched lines.",
								Type:  "gauge",
								Field: 1,
								Match: "^matched_line",
							},
						},
					},
				},
			},
			executor: &mockExecutor{
				output: "matched_line 100\nunmatched_line 200",
			},
			expectedMetric: `
# HELP filter_metric_filtered_sub created for matched lines.
# TYPE filter_metric_filtered_sub gauge
filter_metric_filtered_sub 100
`,
		},
		{
			name: "blacklisted command",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "blacklisted_metric",
						Help:    "command should be blocked.",
						Type:    "gauge",
						Command: "rm -rf /",
					},
				},
				Global: config.Global{
					CommandBlacklist: []string{"rm"},
				},
			},
			executor:       &mockExecutor{},
			expectedMetric: ``,
		},
		{
			name: "blacklisted command with ignore flag",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:            "ignored_blacklist_metric",
						Help:            "command should be allowed.",
						Type:            "gauge",
						Command:         "rm -rf /safe",
						IgnoreBlacklist: true,
					},
				},
				Global: config.Global{
					CommandBlacklist: []string{"rm"},
				},
			},
			executor: &mockExecutor{
				output: "1",
			},
			expectedMetric: `
# HELP ignored_blacklist_metric command should be allowed.
# TYPE ignored_blacklist_metric gauge
ignored_blacklist_metric 1
`,
		},
		{
			name: "command with blacklisted word in command argmnts",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "not_blacklisted_metric",
						Help:    "command should be allowed.",
						Type:    "gauge",
						Command: "echo \"fake rm\"",
					},
				},
				Global: config.Global{
					CommandBlacklist: []string{"rm"},
				},
			},
			executor: &mockExecutor{
				output: "1",
			},
			expectedMetric: `
# HELP not_blacklisted_metric command should be allowed.
# TYPE not_blacklisted_metric gauge
not_blacklisted_metric 1
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			collector := NewCollector(tc.config, logger, tc.executor, cache.New(), "")
			reg := prometheus.NewRegistry()
			reg.MustRegister(collector)

			err := testutil.CollectAndCompare(reg, strings.NewReader(tc.expectedMetric))
			if err != nil {
				t.Errorf("unexpected collecting result:\n%s", err)
			}
		})
	}
}

func TestMergeLabels(t *testing.T) {
	testCases := []struct {
		name     string
		parent   map[string]string
		child    map[string]string
		expected map[string]string
	}{
		{
			name:     "both nil",
			parent:   nil,
			child:    nil,
			expected: nil,
		},
		{
			name:     "parent nil",
			parent:   nil,
			child:    map[string]string{"a": "1"},
			expected: map[string]string{"a": "1"},
		},
		{
			name:     "child nil",
			parent:   map[string]string{"b": "2"},
			child:    nil,
			expected: map[string]string{"b": "2"},
		},
		{
			name:     "no conflicts",
			parent:   map[string]string{"a": "1"},
			child:    map[string]string{"b": "2"},
			expected: map[string]string{"a": "1", "b": "2"},
		},
		{
			name:     "with conflicts",
			parent:   map[string]string{"a": "1", "b": "original"},
			child:    map[string]string{"b": "override", "c": "3"},
			expected: map[string]string{"a": "1", "b": "override", "c": "3"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			merged := mergeLabels(tc.parent, tc.child)
			if len(merged) != len(tc.expected) {
				t.Errorf("expected %d labels, but got %d", len(tc.expected), len(merged))
			}
			for k, v := range tc.expected {
				if merged[k] != v {
					t.Errorf("expected label %s with value %s, but got %s", k, v, merged[k])
				}
			}
		})
	}
}

func TestGetLabelNames(t *testing.T) {
	testCases := []struct {
		name     string
		labels   []config.DynamicLabel
		expected []string
	}{
		{
			name:     "nil slice",
			labels:   nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			labels:   []config.DynamicLabel{},
			expected: nil,
		},
		{
			name: "one label",
			labels: []config.DynamicLabel{
				{Name: "label1"},
			},
			expected: []string{"label1"},
		},
		{
			name: "multiple labels",
			labels: []config.DynamicLabel{
				{Name: "label1"},
				{Name: "label2"},
			},
			expected: []string{"label1", "label2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			names := getLabelNames(tc.labels)
			if len(names) != len(tc.expected) {
				t.Errorf("expected %d names, but got %d", len(tc.expected), len(names))
			}
			for i, name := range names {
				if name != tc.expected[i] {
					t.Errorf("expected name %s, but got %s", tc.expected[i], name)
				}
			}
		})
	}
}

func TestGetLabelValues(t *testing.T) {
	fields := []string{"field0", "field1", "field2"}

	testCases := []struct {
		name     string
		labels   []config.DynamicLabel
		expected []string
	}{
		{
			name:     "nil slice",
			labels:   nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			labels:   []config.DynamicLabel{},
			expected: nil,
		},
		{
			name: "valid labels",
			labels: []config.DynamicLabel{
				{Field: 0},
				{Field: 2},
			},
			expected: []string{"field0", "field2"},
		},
		{
			name: "field index out of range",
			labels: []config.DynamicLabel{
				{Field: 0},
				{Field: 5}, // out of range
			},
			expected: []string{"field0", ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values := getLabelValues(fields, tc.labels)
			if len(values) != len(tc.expected) {
				t.Errorf("expected %d values, but got %d", len(tc.expected), len(values))
			}
			for i, value := range values {
				if value != tc.expected[i] {
					t.Errorf("expected value %s, but got %s", tc.expected[i], value)
				}
			}
		})
	}
}

func TestToPrometheusValueType(t *testing.T) {
	testCases := []struct {
		name         string
		metricType   string
		expectedType prometheus.ValueType
		wantErr      bool
	}{
		{"gauge", "gauge", prometheus.GaugeValue, false},
		{"counter", "counter", prometheus.CounterValue, false},
		{"invalid type", "invalid", 0, true},
		{"empty type", "", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valueType, err := toPrometheusValueType(tc.metricType)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error, but got: %v", err)
				}
			}

			if valueType != tc.expectedType {
				t.Errorf("expected type %v, but got %v", tc.expectedType, valueType)
			}
		})
	}
}

func TestInternalMetrics(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cache := cache.New()

	cfg := &config.Config{
		Metrics: []config.Metric{
			{Name: "ok_metric", Command: "echo 1", CacheTTL: 1 * time.Minute},
			{Name: "err_metric", Command: "exit 1"},
		},
	}

	executor := &mockExecutor{
		err: errors.New("error"),
	}

	collector := NewCollector(cfg, logger, executor, cache, "")

	ch := make(chan prometheus.Metric, 10)
	go func() {
		for range ch {
		}
	}()

	checksBefore := testutil.ToFloat64(Checks)
	errorsOkBefore := testutil.ToFloat64(CommandErrors.WithLabelValues("ok_metric"))
	errorsErrBefore := testutil.ToFloat64(CommandErrors.WithLabelValues("err_metric"))
	hitsBefore := testutil.ToFloat64(CacheHits)
	missesBefore := testutil.ToFloat64(CacheMisses)

	collector.Collect(ch)
	collector.Collect(ch)

	close(ch)

	if val := testutil.ToFloat64(Checks) - checksBefore; val != 2 {
		t.Errorf("Checks: wanted 2, got %v", val)
	}

	if val := testutil.ToFloat64(CommandErrors.WithLabelValues("err_metric")) - errorsErrBefore; val != 1 {
		t.Errorf("CommandErrors for err_metric: wanted 1, got %v", val)
	}

	if val := testutil.ToFloat64(CommandErrors.WithLabelValues("ok_metric")) - errorsOkBefore; val != 1 {
		t.Errorf("ComandErrors for ok_metric: wanted 1, got %v", val)
	}

	if val := testutil.ToFloat64(CacheHits) - hitsBefore; val != 2 {
		t.Errorf("CacheHits: wanted 2, got %v", val)
	}

	if val := testutil.ToFloat64(CacheMisses) - missesBefore; val != 2 {
		t.Errorf("CacheMisses: wnted 2, got %v", val)
	}
}

func TestReloadConfig(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	configV1 := `
server:
  listen_address: ":8080"
  metrics_path: "/metrics"
logging:
  level: "info"
metrics:
  - name: "metric_v1"
    help: "help v1"
    type: "gauge"
    command: "echo 1"
`
	configV2 := `
server:
  listen_address: ":8080"
  metrics_path: "/metrics"
logging:
  level: "info"
metrics:
  - name: "metric_v2"
    help: "help v2"
    type: "counter"
    command: "echo 2"
`

	tmpfile, err := os.CreateTemp(t.TempDir(), "test-config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configV1)); err != nil {
		t.Fatalf("failed to write v1 config: %v", err)
	}

	var cfg config.Config
	if err := config.Load(tmpfile.Name(), &cfg); err != nil {
		t.Fatalf("failed to load v1 config: %v", err)
	}

	collector := NewCollector(&cfg, logger, &mockExecutor{}, cache.New(), tmpfile.Name())

	if collector.config.Metrics[0].Name != "metric_v1" {
		t.Fatalf("expected initial metric to be metric_v1, got %s", collector.config.Metrics[0].Name)
	}

	if err := os.WriteFile(tmpfile.Name(), []byte(configV2), 0644); err != nil {
		t.Fatalf("failed to write v2 config: %v", err)
	}

	if err := collector.ReloadConfig(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if len(collector.config.Metrics) != 1 {
		t.Fatalf("expected 1 metric after reload, got %d", len(collector.config.Metrics))
	}
	if collector.config.Metrics[0].Name != "metric_v2" {
		t.Errorf("expected metric_v2 after reload, got %s", collector.config.Metrics[0].Name)
	}
	if collector.config.Metrics[0].Type != "counter" {
		t.Errorf("expected counter type after reload, got %s", collector.config.Metrics[0].Type)
	}
}
