package cache

import (
	"errors"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	testCases := []struct {
		name        string
		key         string
		value       string
		err         error
		ttl         time.Duration
		expectFound bool
		expectValue string
		expectErr   error
	}{
		{
			name:        "set get",
			key:         "key1",
			value:       "value1",
			err:         nil,
			ttl:         1 * time.Minute,
			expectFound: true,
			expectValue: "value1",
			expectErr:   nil,
		},
		{
			name:        "get no existing value",
			key:         "nonexistent",
			expectFound: false,
		},
		{
			name:        "errors caching",
			key:         "error key",
			value:       "",
			err:         errors.New("some error"),
			ttl:         1 * time.Minute,
			expectFound: true,
			expectValue: "",
			expectErr:   errors.New("some error"),
		},
		{
			name:        "expired value",
			key:         "expired key",
			value:       "some value",
			ttl:         1 * time.Millisecond,
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache := New()

			if tc.value != "" || tc.err != nil {
				cache.Set(tc.key, tc.value, tc.err, tc.ttl)
			}

			if tc.name == "expired value" {
				time.Sleep(10 * time.Millisecond)
			}

			val, err, found := cache.Get(tc.key)

			if found != tc.expectFound {
				t.Errorf("expected found to be %v, but got %v", tc.expectFound, found)
			}

			if val != tc.expectValue {
				t.Errorf("expected value to be '%s', but got '%s'", tc.expectValue, val)
			}

			if (err != nil && tc.expectErr == nil) || (err == nil && tc.expectErr != nil) || (err != nil && tc.expectErr != nil && err.Error() != tc.expectErr.Error()) {
				t.Errorf("expected error to be '%v', but got '%v'", tc.expectErr, err)
			}
		})
	}
}

func TestCache_UniqueKeys(t *testing.T) {
	cache := New()

	key1 := "metric1::echo hello"
	value1 := "output1"

	key2 := "metric2::echo hello"
	value2 := "output2"

	cache.Set(key1, value1, nil, 1*time.Minute)
	cache.Set(key2, value2, nil, 1*time.Minute)

	val, _, found := cache.Get(key1)
	if !found || val != value1 {
		t.Errorf("expected to find key '%s' with value '%s', but got '%s'", key1, value1, val)
	}

	val, _, found = cache.Get(key2)
	if !found || val != value2 {
		t.Errorf("expected to find key '%s' with value '%s', but got '%s'", key2, value2, val)
	}
}
