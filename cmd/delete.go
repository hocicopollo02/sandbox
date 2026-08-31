package cmd

import (
	"errors"
	"fmt"

	"github.com/hocicopollo02/sandbox/internal/metadata"
	"github.com/hocicopollo02/sandbox/internal/sandbox"
	"github.com/hocicopollo02/sandbox/internal/ui"
	"github.com/spf13/cobra"
)

func newDeleteCommand(appState *app) *cobra.Command {
	var yes, keepHome, ifExists bool
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
			info, err := appState.manager.Lookup(cmd.Context(), args[0])
			if err != nil {
				if ifExists && errors.Is(err, metadata.ErrNotFound) {
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
	return cmd
}
