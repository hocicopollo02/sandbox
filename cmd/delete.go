package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hocicopollo02/sandbox/internal/metadata"
	"github.com/hocicopollo02/sandbox/internal/sandbox"
	"github.com/hocicopollo02/sandbox/internal/ui"
	"github.com/spf13/cobra"
)

func newDeleteCommand(appState *app) *cobra.Command {
	var yes, keepHome, ifExists, jsonOutput bool
	cmd := &cobra.Command{
		Use:     "delete NAME",
		Aliases: []string{"rm"},
		Short:   "Delete a sandbox and optionally its isolated home",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("delete requires a sandbox name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput && !yes {
				return fmt.Errorf("delete --json requires --yes")
			}
			info, err := appState.manager.Lookup(cmd.Context(), args[0])
			if err != nil {
				if ifExists && errors.Is(err, metadata.ErrNotFound) {
					if jsonOutput {
						return writeDeleteJSON(appState, args[0], "unchanged", nil)
					}
					_, _ = fmt.Fprintf(appState.out, "Sandbox %s does not exist; nothing to do\n", args[0])
					return nil
				}
				return err
			}
			if !yes {
				confirmed, err := ui.ConfirmDelete(info.Name, appState.in, appState.out)
				if err != nil {
					return err
				}
				if !confirmed {
					return fmt.Errorf("deletion cancelled")
				}
			}
			deleteHome := info.HomeMode == sandbox.IsolatedHome && !keepHome
			if deleteHome && !yes {
				deleteHome, err = ui.ConfirmDeleteHome(appState.in, appState.out)
				if err != nil {
					return err
				}
			}
			if err := appState.manager.Delete(cmd.Context(), info.Name, sandbox.DeleteOptions{DeleteHome: deleteHome, IfExists: ifExists}); err != nil {
				return err
			}
			if jsonOutput {
				var retainedHome *string
				if !deleteHome && info.HomeMode == sandbox.IsolatedHome {
					path := appState.manager.ManagedHome(info.Name)
					retainedHome = &path
				}
				return writeDeleteJSON(appState, info.Name, "deleted", retainedHome)
			}
			appState.ui.Success("Sandbox deleted")
			if !deleteHome && info.HomeMode == sandbox.IsolatedHome {
				if _, err := fmt.Fprintf(appState.out, "Retained isolated home: %s\n", appState.manager.ManagedHome(info.Name)); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmations")
	cmd.Flags().BoolVar(&keepHome, "keep-home", false, "keep an isolated home")
	cmd.Flags().BoolVar(&ifExists, "if-exists", false, "succeed as a no-op when the sandbox does not exist")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print successful result as JSON")
	return cmd
}

func writeDeleteJSON(appState *app, name, result string, retainedHome *string) error {
	return json.NewEncoder(appState.out).Encode(struct {
		Name         string  `json:"name"`
		Result       string  `json:"result"`
		RetainedHome *string `json:"retained_home"`
	}{Name: name, Result: result, RetainedHome: retainedHome})
}
