package distrobox

import (
	"context"
	"fmt"
	"strings"

	"github.com/pablo/sandbox/internal/execx"
)

type Client struct {
	Runner execx.Runner
}

func New(runner execx.Runner) *Client { return &Client{Runner: runner} }

func (c *Client) Available() error {
	if _, err := c.Runner.LookPath("distrobox"); err != nil {
		return fmt.Errorf("distrobox is required but was not found; on Arch/Omarchy run: sudo pacman -S distrobox")
	}
	return nil
}

func (c *Client) Create(ctx context.Context, name, image, home string) error {
	args := []string{"create", "--yes", "--name", name, "--image", image}
	if home != "" {
		args = append(args, "--home", home)
	}
	output, err := c.Runner.Run(ctx, "distrobox", args...)
	if err != nil {
		return commandError("create", name, output, err)
	}
	return nil
}

func (c *Client) Enter(ctx context.Context, name string) error {
	if err := c.Runner.Attach(ctx, "distrobox", "enter", name); err != nil {
		return fmt.Errorf("could not enter sandbox %q: %w", name, err)
	}
	return nil
}

func (c *Client) Stop(ctx context.Context, name string) error {
	output, err := c.Runner.Run(ctx, "distrobox", "stop", name)
	if err != nil {
		return commandError("stop", name, output, err)
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, name string) error {
	output, err := c.Runner.Run(ctx, "distrobox", "rm", "--force", name)
	if err != nil {
		return commandError("delete", name, output, err)
	}
	return nil
}

func commandError(action, name string, output []byte, err error) error {
	detail := strings.TrimSpace(string(output))
	if detail == "" {
		return fmt.Errorf("could not %s sandbox %q: %w; run with --verbose for details", action, name, err)
	}
	return fmt.Errorf("could not %s sandbox %q: %s", action, name, detail)
}
