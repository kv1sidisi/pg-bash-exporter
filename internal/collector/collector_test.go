package collector

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"io"
	"log/slog"
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
			name: "gauge metric with sub-metrics",
			config: &config.Config{
				Metrics: []config.Metric{
					{
						Name:    "sub_metric",
						Help:    "A metric with sub-metrics.",
						Type:    "gauge",
						Command: "echo '10 20'",
						SubMetrics: []config.SubMetric{
							{
								Name:  "sub_one",
								Help:  "First sub-metric.",
								Type:  "gauge",
								Field: 0,
							},
							{
								Name:  "sub_two",
								Help:  "Second sub-metric.",
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
# HELP sub_metric_sub_one First sub-metric.
# TYPE sub_metric_sub_one gauge
sub_metric_sub_one 10
# HELP sub_metric_sub_two Second sub-metric.
# TYPE sub_metric_sub_two gauge
sub_metric_sub_two 20
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
						SubMetrics: []config.SubMetric{
							{
								Name:  "dynamic_sub_metric",
								Help:  "A sub-metric with dynamic labels.",
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
# HELP dynamic_labels_metric_dynamic_sub_metric A sub-metric with dynamic labels.
# TYPE dynamic_labels_metric_dynamic_sub_metric gauge
dynamic_labels_metric_dynamic_sub_metric{my_label="label_val1"} 10
dynamic_labels_metric_dynamic_sub_metric{my_label="label_val2"} 20
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			collector := NewCollector(tc.config, logger, tc.executor)
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
		name       string
		metricType string
		expected   prometheus.ValueType
	}{
		{"gauge", "gauge", prometheus.GaugeValue},
		{"counter", "counter", prometheus.CounterValue},
		{"invalid type", "invalid", 0},
		{"empty type", "", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valueType, err := toPrometheusValueType(tc.metricType)
			if err != nil {
				t.Errorf("expected %v, but got error: %v", tc.expected, err)
			}
			if valueType != tc.expected {
				t.Errorf("expected %v, but got %v", tc.expected, valueType)
			}
		})
	}
}
