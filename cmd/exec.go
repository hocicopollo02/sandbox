package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newExecCommand(appState *app) *cobra.Command {
	return &cobra.Command{
		Use:   "exec NAME -- COMMAND [ARG...]",
		Short: "Run a command in a sandbox without a TTY",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("exec requires a sandbox name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			atDash := cmd.ArgsLenAtDash()
			if atDash < 0 || atDash >= len(args) {
				return fmt.Errorf("exec requires a command after --")
			}
			return appState.manager.Exec(cmd.Context(), args[0], args[atDash:])
		},
	}
}
