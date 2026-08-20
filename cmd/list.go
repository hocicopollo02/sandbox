package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newListCommand(appState *app) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List persistent sandboxes",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := appState.manager.List(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				data, err := json.MarshalIndent(entries, "", "  ")
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(appState.out, string(data))
				return err
			}
			if len(entries) == 0 {
				_, err := fmt.Fprintln(appState.out, "No persistent sandboxes.")
				return err
			}
			writer := tabwriter.NewWriter(appState.out, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(writer, "NAME\tDISTRO\tTYPE\tHOME\tSTATUS")
			for _, entry := range entries {
				_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", entry.Name, entry.Distribution, entry.Persistence, entry.HomeMode, entry.Status)
			}
			return writer.Flush()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}
