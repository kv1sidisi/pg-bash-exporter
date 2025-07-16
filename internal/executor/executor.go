package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type BashExecutor struct{}

// ExecuteCommand executes shell command with timeout control.
// returns command stout or stderr on failure.
func (e *BashExecutor) ExecuteCommand(ctx context.Context, command string, timeout time.Duration) (string, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", fmt.Errorf("command timed out after %s: %w", timeout, ctx.Err())
	}
	if ctx.Err() != nil {
		return "", fmt.Errorf("command execution failed: %w", ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("command execution failed: %w; stderr: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
