package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hocicopollo02/sandbox/internal/model"
	"github.com/hocicopollo02/sandbox/internal/sandbox"
	"github.com/spf13/cobra"
)

type infoJSONView struct {
	Name         string            `json:"name"`
	Distribution string            `json:"distribution"`
	Image        string            `json:"image"`
	Persistence  model.Persistence `json:"persistence"`
	HomeMode     model.HomeMode    `json:"home_mode"`
	HomePath     string            `json:"home_path"`
	CreatedAt    time.Time         `json:"created_at"`
	Status       model.Status      `json:"status"`
}

func marshalInfoJSON(info model.Info) ([]byte, error) {
	return json.MarshalIndent(infoJSONView{
		Name:         info.Name,
		Distribution: info.Distribution,
		Image:        info.Image,
		Persistence:  info.Persistence,
		HomeMode:     info.HomeMode,
		HomePath:     info.HomePath,
		CreatedAt:    info.CreatedAt,
		Status:       info.Status,
	}, "", "  ")
}

func newInfoCommand(appState *app) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "info NAME",
		Short: "Show sandbox metadata and runtime status",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("info requires a sandbox name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := appState.manager.Info(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if asJSON {
				data, err := marshalInfoJSON(info)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(appState.out, string(data))
				return err
			}
			created := info.CreatedAt.Local().Format(time.RFC3339)
			distribution := info.Distribution
			if distro, ok := sandbox.FindDistribution(info.Distribution); ok {
				distribution = distro.Name
			}
			homePath := "-"
			if info.HomePath != "" {
				homePath = info.HomePath
			}
			_, err = fmt.Fprintf(appState.out, "Name          %s\nDistribution  %s\nImage         %s\nType          %s\nStatus        %s\nHome          %s\nHome path     %s\nCreated       %s\nContainer     %s\n", info.Name, distribution, info.Image, info.Persistence, info.Status, info.HomeMode, filepath.Clean(homePath), created, info.Name)
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}
