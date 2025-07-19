package config_test

import (
	"os"
	"pg-bash-exporter/internal/config"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	testCases := []struct {
		name          string
		yaml          string
		path          string
		wantErr       bool
		expectedError string
	}{
		{
			name: "valid config",
			yaml: `
logging:
  level: "info"
global:
  timeout: "1s"
  cache_ttl: "1s"
  max_concurrent: 1
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
`,
			wantErr: false,
		},
		{
			name:    "bad path",
			path:    "/tmp/this/file/does/not/exist.yaml",
			wantErr: true,
		},
		{
			name:    "bad yaml",
			yaml:    "server: 'bad",
			wantErr: true,
		},
		{
			name: "metric without help",
			yaml: `
logging:
  level: "info"
global:
  timeout: 1s
  cache_ttl: 1s
  max_concurrent: 1
metrics:
  - name: "my_metric"
    # help: "some help"
    type: "gauge"
    command: "echo 1"
`,
			wantErr:       true,
			expectedError: "help string is required",
		},
		{
			name: "metric with bad type",
			yaml: `
server:
  listen_address: ":8080"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: 1s
  cache_ttl: 1s
  max_concurrent: 1
metrics:
  - name: "my_metric"
    help: "some help"
    type: "bad type"
    command: "echo 1"
`,
			wantErr:       true,
			expectedError: "type is invalid. valid: gauge, counter",
		},
		{
			name: "no metrics defined",
			yaml: `
server:
  listen_address: ":8080"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: 1s
  cache_ttl: 1s
  max_concurrent: 1
metrics: []
`,
			wantErr:       true,
			expectedError: "at least one metric must be defined",
		},
		{
			name: "invalid metric name",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
metrics:
  - name: "my-metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
`,
			wantErr:       true,
			expectedError: "metric name is not valid",
		},
		{
			name: "invalid dynamic label name",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
    dynamic_labels:
      - name: "1_invalid_label"
        field: 0
`,
			wantErr:       true,
			expectedError: "dynamic_label name: 1_invalid_label is not valid",
		},

		{
			name: "invalid logging level",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "warning"
global:
  timeout: "1s"
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
`,
			wantErr:       true,
			expectedError: "is not valid. Valid levels: info, debug, error",
		},
		{
			name: "negative global timeout",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "-5s"
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
`,
			wantErr:       true,
			expectedError: "global.timeout must be > 0",
		},
		{
			name: "negative global cache_ttl",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
  cache_ttl: "-1m"
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
`,
			wantErr:       true,
			expectedError: "global.cache_ttl must be > 0",
		},
		{
			name: "negative global max_concurrent",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
  max_concurrent: -5
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
`,
			wantErr:       true,
			expectedError: "global.max_concurrent must be > 0",
		},
		{
			name: "metric with empty command",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: ""
`,
			wantErr:       true,
			expectedError: "command is required",
		},
		{
			name: "metric with invalid static label name",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
    labels:
      "invalid-label": "value"
`,
			wantErr:       true,
			expectedError: "label name invalid-label is not valid",
		},
		{
			name: "metric with empty dynamic label name",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
    dynamic_labels:
      - name: ""
        field: 0
`,
			wantErr:       true,
			expectedError: "dynamic_label name is required",
		},
		{
			name: "postfix-metric with invalid name",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
    postfix_metrics:
      - name: "invalid-postfix-metric"
        help: "help"
        type: "gauge"
        field: 0
`,
			wantErr:       true,
			expectedError: "postfix-metric name is not valid",
		},
		{
			name: "postfix-metric with empty help",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
    postfix_metrics:
      - name: "my_sub"
        help: ""
        type: "gauge"
        field: 0
`,
			wantErr:       true,
			expectedError: "help string is required",
		},
		{
			name: "postfix-metric with invalid type",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
    postfix_metrics:
      - name: "my_sub"
        help: "a sub metric"
        type: "bad_type"
        field: 0
`,
			wantErr:       true,
			expectedError: "type is invalid. valid: gauge, counter",
		},
		{
			name: "postfix-metric with negative field",
			yaml: `
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
logging:
  level: "info"
global:
  timeout: "1s"
metrics:
  - name: "my_metric"
    help: "help"
    type: "gauge"
    command: "echo 1"
    postfix_metrics:
      - name: "my_sub"
        help: "a sub metric"
        type: "gauge"
        field: -1
`,
			wantErr:       true,
			expectedError: "field must be >= 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.path
			if path == "" {
				tmpfile, err := os.CreateTemp("", "test-*.yaml")
				if err != nil {
					t.Fatalf("could not create temp file: %v", err)
				}
				defer os.Remove(tmpfile.Name())

				if _, err := tmpfile.Write([]byte(tc.yaml)); err != nil {
					t.Fatalf("could not write to temp file: %v", err)
				}
				tmpfile.Close()
				path = tmpfile.Name()
			}

			var cfg config.Config
			err := config.Load(path, &cfg)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Load() passed, but it should have failed")
				}
				if tc.expectedError != "" && !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("error message should contain '%s', but it was: '%s'", tc.expectedError, err.Error())
				}
			} else if err != nil {
				t.Fatalf("Load() failed, but it should have passed. error: %v", err)
			}
		})
	}
}
