package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(appState *app) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				return json.NewEncoder(appState.out).Encode(struct {
					Name      string `json:"name"`
					Version   string `json:"version"`
					Commit    string `json:"commit"`
					BuildDate string `json:"build_date"`
				}{
					Name:      sandboxBinary,
					Version:   Version,
					Commit:    Commit,
					BuildDate: BuildDate,
				})
			}
			_, err := fmt.Fprintf(appState.out, "sandbox %s\n", Version)
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}
