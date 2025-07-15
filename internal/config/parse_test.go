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
server:
  listen_address: ":1234"
  metrics_path: "/metrics"
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
			name: "missing server address",
			yaml: `
server:
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
    type: "gauge"
    command: "echo 1"
`,
			wantErr:       true,
			expectedError: "server.listen_address is required",
		},
		{
			name: "metric without help",
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
