package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hocicopollo02/sandbox/internal/execx"
	"github.com/spf13/cobra"
)

const (
	sandboxModule = "github.com/hocicopollo02/sandbox"
	sandboxBinary = "sandbox"
)

type upgradeResult struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	Result         string `json:"result"`
}

type moduleVersion struct {
	display string
	query   string
}

func newUpgradeCommand(appState *app) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Update sandbox to the latest release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(cmd.Context(), appState, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

func runUpgrade(ctx context.Context, appState *app, asJSON bool) error {
	latest, err := resolveLatestVersion(ctx, appState.runner)
	if err != nil {
		return err
	}

	result := upgradeResult{
		Name:           sandboxBinary,
		CurrentVersion: displayVersion(Version),
		LatestVersion:  latest.display,
		Result:         "unchanged",
	}
	if result.CurrentVersion == result.LatestVersion {
		return renderUpgradeResult(appState.out, result, asJSON)
	}

	executable, err := appExecutablePath(appState)
	if err != nil {
		return fmt.Errorf("could not locate the running sandbox executable: %w", err)
	}
	installDir := filepath.Dir(executable)
	runner, ok := appState.runner.(interface {
		RunWithEnv(context.Context, map[string]string, string, ...string) ([]byte, error)
	})
	if !ok {
		return fmt.Errorf("could not upgrade sandbox: command runner does not support environment overrides")
	}

	output, err := runner.RunWithEnv(ctx, map[string]string{"GOBIN": installDir}, "go", "install", sandboxModule+"@"+latest.query)
	if err != nil {
		return upgradeCommandError("upgrade sandbox", output, err)
	}
	result.Result = "upgraded"
	return renderUpgradeResult(appState.out, result, asJSON)
}

func resolveLatestVersion(ctx context.Context, runner execx.Runner) (moduleVersion, error) {
	output, err := runner.Run(ctx, "go", "list", "-m", "-f", "{{.Version}}", sandboxModule+"@latest")
	if err != nil {
		return moduleVersion{}, upgradeCommandError("determine the latest sandbox version", output, err)
	}
	version, err := parseModuleVersion(output)
	if err != nil {
		return moduleVersion{}, err
	}
	return version, nil
}

func parseModuleVersion(output []byte) (moduleVersion, error) {
	raw := strings.TrimSpace(string(output))
	if raw == "" || strings.ContainsAny(raw, " \t\r\n") {
		return moduleVersion{}, fmt.Errorf("go returned an invalid sandbox version %q", raw)
	}
	query := raw
	if !strings.HasPrefix(query, "v") {
		query = "v" + query
	}
	display := strings.TrimPrefix(query, "v")
	if display == "" {
		return moduleVersion{}, fmt.Errorf("go returned an invalid sandbox version %q", raw)
	}
	return moduleVersion{display: display, query: query}, nil
}

func appExecutablePath(appState *app) (string, error) {
	if appState.executablePath != nil {
		return appState.executablePath()
	}
	return os.Executable()
}

func displayVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func renderUpgradeResult(out io.Writer, result upgradeResult, asJSON bool) error {
	if asJSON {
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("encode upgrade JSON: %w", err)
		}
		_, err = fmt.Fprintln(out, string(data))
		return err
	}
	if result.Result == "unchanged" {
		_, err := fmt.Fprintf(out, "sandbox is already up to date (%s)\n", result.LatestVersion)
		return err
	}
	_, err := fmt.Fprintf(out, "sandbox upgraded from %s to %s\n", result.CurrentVersion, result.LatestVersion)
	return err
}

func upgradeCommandError(action string, output []byte, err error) error {
	if detail := strings.TrimSpace(string(output)); detail != "" {
		return fmt.Errorf("could not %s: %s", action, detail)
	}
	return fmt.Errorf("could not %s: %w", action, err)
}
