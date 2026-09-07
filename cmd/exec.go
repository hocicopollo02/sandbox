package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newExecCommand(appState *app) *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
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
			if atDash != 1 {
				return fmt.Errorf("exec requires exactly one sandbox name before --")
			}
			if atDash >= len(args) {
				return fmt.Errorf("exec requires a command after --")
			}
			execCtx, cancel := withTimeout(cmd.Context(), timeout)
			defer cancel()
			return appState.manager.Exec(execCtx, args[0], args[atDash:])
		},
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "maximum duration for the guest command (example 5m); 0 means no limit")
	return cmd
}
