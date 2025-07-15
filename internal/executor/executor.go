package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Executor interface {
	ExecuteCommand(ctx context.Context, command string) (string, error)
}

type BashExecutor struct{}

func (e *BashExecutor) ExecuteCommand(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() != nil {
		return "", fmt.Errorf("command execution failed: %w", ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("command execution failed: %w; stderr: %s", err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}
