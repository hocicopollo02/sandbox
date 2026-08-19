package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStopCommand(appState *app) *cobra.Command {
	return &cobra.Command{
		Use:   "stop NAME",
		Short: "Stop a sandbox without deleting it",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("stop requires a sandbox name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := appState.manager.Stop(cmd.Context(), args[0]); err != nil {
				return err
			}
			appState.ui.Success("Sandbox stopped")
			return nil
		},
	}
}
