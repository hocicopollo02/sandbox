package podman

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pablo/sandbox/internal/execx"
	"github.com/pablo/sandbox/internal/sandbox"
)

type Client struct {
	Runner execx.Runner
}

func New(runner execx.Runner) *Client { return &Client{Runner: runner} }

func (c *Client) Available() error {
	if _, err := c.Runner.LookPath("podman"); err != nil {
		return fmt.Errorf("podman is required but was not found; on Arch/Omarchy run: sudo pacman -S podman")
	}
	return nil
}

func (c *Client) Create(ctx context.Context, name, image, home string) error {
	args := []string{
		"create", "--pull=missing", "--name", name, "--hostname", name,
		"--workdir", "/home/sandbox", "--env", "HOME=/home/sandbox",
	}
	if home != "" {
		args = append(args, "--volume", home+":/home/sandbox")
	}
	args = append(args, image, "sleep", "infinity")
	output, err := c.Runner.Run(ctx, "podman", args...)
	if err != nil {
		return commandError("create", name, output, err)
	}
	output, err = c.Runner.Run(ctx, "podman", "start", name)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cleanupOutput, cleanupErr := c.Runner.Run(cleanupCtx, "podman", "rm", "--force", name)
		startErr := commandError("start", name, output, err)
		if cleanupErr == nil {
			return startErr
		}
		cleanupDetail := strings.TrimSpace(string(cleanupOutput))
		if cleanupDetail == "" {
			return errors.Join(startErr, fmt.Errorf("could not clean up sandbox %q after failed start: %w", name, cleanupErr))
		}
		return errors.Join(startErr, fmt.Errorf("could not clean up sandbox %q after failed start: %s", name, cleanupDetail))
	}
	return nil
}

func (c *Client) Enter(ctx context.Context, name string) error {
	status, err := c.Status(ctx, name)
	if err != nil {
		return err
	}
	if status == sandbox.Missing {
		return fmt.Errorf("sandbox %q does not exist in the container runtime", name)
	}
	if status == sandbox.Unknown {
		return fmt.Errorf("could not determine the status of sandbox %q", name)
	}
	if status == sandbox.Stopped {
		output, err := c.Runner.Run(ctx, "podman", "start", name)
		if err != nil {
			return commandError("start", name, output, err)
		}
	}
	if err := c.Runner.Attach(ctx, "podman", "exec", "--interactive", "--tty", name, "/bin/bash"); err != nil {
		return fmt.Errorf("could not enter sandbox %q: %w", name, err)
	}
	return nil
}

func (c *Client) Stop(ctx context.Context, name string) error {
	output, err := c.Runner.Run(ctx, "podman", "stop", "--time", "10", name)
	if err != nil {
		return commandError("stop", name, output, err)
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, name string) error {
	output, err := c.Runner.Run(ctx, "podman", "rm", "--force", name)
	if err != nil {
		return commandError("delete", name, output, err)
	}
	return nil
}

func (c *Client) Status(ctx context.Context, name string) (sandbox.Status, error) {
	output, err := c.Runner.Run(ctx, "podman", "inspect", "--format", "{{.State.Status}}", name)
	if err != nil {
		if isMissing(string(output)) {
			return sandbox.Missing, nil
		}
		return sandbox.Unknown, fmt.Errorf("could not inspect sandbox %q: %s", name, strings.TrimSpace(string(output)))
	}
	switch strings.TrimSpace(string(output)) {
	case "running":
		return sandbox.Running, nil
	case "":
		return sandbox.Unknown, fmt.Errorf("podman returned no status for sandbox %q", name)
	default:
		return sandbox.Stopped, nil
	}
}

func (c *Client) Info(ctx context.Context) ([]byte, error) {
	output, err := c.Runner.Run(ctx, "podman", "info", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("podman is not working: %s", strings.TrimSpace(string(output)))
	}
	return output, nil
}

func (c *Client) Rootless(ctx context.Context) (bool, error) {
	output, err := c.Runner.Run(ctx, "podman", "info", "--format", "{{.Host.Security.Rootless}}")
	if err != nil {
		return false, fmt.Errorf("could not determine whether Podman is rootless: %s", strings.TrimSpace(string(output)))
	}
	value := strings.TrimSpace(string(output))
	if strings.EqualFold(value, "true") {
		return true, nil
	}
	if strings.EqualFold(value, "false") {
		return false, nil
	}
	var info struct {
		Host struct {
			Security struct {
				Rootless bool `json:"rootless"`
			} `json:"security"`
		} `json:"host"`
	}
	if err := json.Unmarshal(output, &info); err != nil {
		return false, fmt.Errorf("unexpected Podman rootless response %q", value)
	}
	return info.Host.Security.Rootless, nil
}

func commandError(action, name string, output []byte, err error) error {
	detail := strings.TrimSpace(string(output))
	if detail == "" {
		return fmt.Errorf("could not %s sandbox %q: %w; run with --verbose for details", action, name, err)
	}
	return fmt.Errorf("could not %s sandbox %q: %s", action, name, detail)
}

func isMissing(output string) bool {
	value := strings.ToLower(output)
	return strings.Contains(value, "no such container") ||
		strings.Contains(value, "no such object") ||
		strings.Contains(value, "does not exist") ||
		strings.Contains(value, "not found")
}
