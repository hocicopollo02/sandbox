package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(appState *app) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(appState.out, "sandbox %s\n", Version)
			return err
		},
	}
}
