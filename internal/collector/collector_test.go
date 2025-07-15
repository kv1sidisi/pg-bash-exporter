package collector

import (
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
