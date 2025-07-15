package executor

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExecuteCommand(t *testing.T) {
	testCases := []struct {
		name        string
		command     string
		timeout     time.Duration
		wantOutput  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "successful execution",
			command:    "echo 'hello world'",
			timeout:    5 * time.Second,
			wantOutput: "hello world",
			wantErr:    false,
		},
		{
			name:        "command fails with stderr",
			command:     "echo 'error message' >&2; exit 1",
			timeout:     5 * time.Second,
			wantErr:     true,
			errContains: "error message",
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			gotOutput, err := ExecuteCommand(ctx, tc.command)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error, but got none")
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tc.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect an error, but got: %v", err)
				}
			}

			if gotOutput != tc.wantOutput {
				t.Errorf("output = %q, want %q", gotOutput, tc.wantOutput)
			}
		})
	}
}
