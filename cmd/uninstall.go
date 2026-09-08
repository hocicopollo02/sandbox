package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hocicopollo02/sandbox/internal/ui"
	"github.com/spf13/cobra"
)

func newUninstallCommand(appState *app) *cobra.Command {
	var yes, asJSON bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the sandbox executable and keep its data",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON && !yes {
				return fmt.Errorf("uninstall --json requires --yes")
			}
			path, err := currentExecutablePath(appState)
			if err != nil {
				return err
			}
			if !yes {
				confirmed, err := ui.ConfirmUninstall(path, appState.in, appState.out)
				if err != nil {
					return err
				}
				if !confirmed {
					return fmt.Errorf("uninstallation cancelled")
				}
			}
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("could not uninstall sandbox: %w", err)
			}
			if asJSON {
				return json.NewEncoder(appState.out).Encode(struct {
					Name   string `json:"name"`
					Path   string `json:"path"`
					Result string `json:"result"`
				}{Name: sandboxBinary, Path: path, Result: "uninstalled"})
			}
			_, err = fmt.Fprintf(appState.out, "sandbox uninstalled from %s\n", path)
			return err
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print successful result as JSON")
	return cmd
}

func currentExecutablePath(appState *app) (string, error) {
	path, err := appExecutablePath(appState)
	if err != nil {
		return "", fmt.Errorf("could not locate the running sandbox executable: %w", err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("could not locate the running sandbox executable: %w", err)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("could not locate the running sandbox executable: %w", err)
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("could not locate the running sandbox executable: %s", path)
	}
	if err != nil {
		return "", fmt.Errorf("inspect the running sandbox executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("could not uninstall sandbox: executable path is not a regular file: %s", path)
	}
	return path, nil
}
