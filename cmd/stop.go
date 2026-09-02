package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/hocicopollo02/sandbox/internal/sandbox"
	"github.com/spf13/cobra"
)

func newStopCommand(appState *app) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "stop NAME",
		Short: "Stop a sandbox without deleting it",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("stop requires a sandbox name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := appState.manager.StopWithResult(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if jsonOutput {
				return json.NewEncoder(appState.out).Encode(struct {
					Name   string             `json:"name"`
					Result sandbox.StopResult `json:"result"`
				}{Name: args[0], Result: result})
			}
			appState.ui.Success("Sandbox stopped")
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print successful result as JSON")
	return cmd
}
