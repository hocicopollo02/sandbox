package cmd

import (
	"fmt"

	"github.com/hocicopollo02/sandbox/internal/ui"
	"github.com/spf13/cobra"
)

func newEnterCommand(appState *app) *cobra.Command {
	return &cobra.Command{
		Use:     "enter [name]",
		Aliases: []string{"shell"},
		Short:   "Enter a persistent sandbox",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("enter accepts at most one name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				entries, err := appState.manager.List(cmd.Context())
				if err != nil {
					return err
				}
				names := make([]string, 0, len(entries))
				for _, entry := range entries {
					names = append(names, entry.Name)
				}
				name, err = ui.SelectSandbox(names, appState.in, appState.out)
				if err != nil {
					return err
				}
			}
			return appState.manager.Enter(cmd.Context(), name)
		},
	}
}
