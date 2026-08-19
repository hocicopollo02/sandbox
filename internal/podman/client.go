package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

func isMissing(output string) bool {
	value := strings.ToLower(output)
	return strings.Contains(value, "no such container") ||
		strings.Contains(value, "no such object") ||
		strings.Contains(value, "does not exist") ||
		strings.Contains(value, "not found")
}
