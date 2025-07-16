package executor

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExecuteCommand(t *testing.T) {
	testCases := []struct {
		name          string
		command       string
		timeout       time.Duration
		context       context.Context
		cancelContext bool
		wantOutput    string
		wantErr       bool
		errContains   string
	}{
		{
			name:       "successful execution",
			command:    "echo 'hello world'",
			timeout:    5 * time.Second,
			context:    context.Background(),
			wantOutput: "hello world",
			wantErr:    false,
		},
		{
			name:        "command fails with stderr",
			command:     "echo 'error message' >&2; exit 1",
			timeout:     5 * time.Second,
			context:     context.Background(),
			wantErr:     true,
			errContains: "error message",
		},
		{
			name:    "context deadline exceeded",
			command: "sleep 2",
			timeout: 0,
			context: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), 300*time.Millisecond)
				return ctx
			}(),
			cancelContext: false,
			wantErr:       true,
			errContains:   "context deadline exceeded",
		},
		{
			name:        "internal timeout exceeded",
			command:     "sleep 1",
			timeout:     50 * time.Millisecond,
			context:     context.Background(),
			wantErr:     true,
			errContains: "command timed out",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.context
			if tc.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
			}

			executor := &BashExecutor{}
			gotOutput, err := executor.ExecuteCommand(ctx, tc.command, tc.timeout)

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
