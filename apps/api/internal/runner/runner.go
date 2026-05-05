package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type Result struct {
	Stdout string
	Stderr string
	JSON   map[string]any
}

type Runner struct {
	Timeout time.Duration
}

func New(timeout time.Duration) *Runner {
	return &Runner{Timeout: timeout}
}

func (r *Runner) Run(executable string, args []string, expectJSON bool) (Result, error) {
	timeout := r.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("module command timed out after %s", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("module command failed: %w: %s", err, result.Stderr)
	}
	if expectJSON && result.Stdout != "" {
		if err := json.Unmarshal(stdout.Bytes(), &result.JSON); err != nil {
			return result, fmt.Errorf("module command returned invalid JSON: %w", err)
		}
	}
	return result, nil
}
