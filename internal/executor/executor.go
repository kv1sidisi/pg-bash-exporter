package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type CommandExecutor struct{}

// ExecuteCommand executes a shell command with timeout control.
// It takes the shell (e.g., "bash", "powershell"), the command to execute, and a timeout.
// It returns the command's stdout or an error on failure.
func (e *CommandExecutor) ExecuteCommand(ctx context.Context, shell, command string, timeout time.Duration) (string, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, shell, "-c", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("command execution failed due to context: %w", ctx.Err())
		}
		return "", fmt.Errorf("command execution failed: %w; stderr: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
