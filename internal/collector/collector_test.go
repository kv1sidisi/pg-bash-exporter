package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"pg-bash-exporter/internal/config"
	"testing"
)

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
				{Field: 5},
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
			valueType := toPrometheusValueType(tc.metricType)
			if valueType != tc.expected {
				t.Errorf("expected %v, but got %v", tc.expected, valueType)
			}
		})
	}
}
