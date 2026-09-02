package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/hocicopollo02/sandbox/internal/sandbox"
	"github.com/hocicopollo02/sandbox/internal/ui"
	"github.com/spf13/cobra"
)

func newCreateCommand(appState *app) *cobra.Command {
	var distro string
	var persistent, disposable bool
	var isolatedHome, sharedHome bool
	var noEnter, yes, ifNotExists, jsonOutput bool

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a disposable or persistent sandbox",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("create accepts at most one name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput && (len(args) != 1 || !cmd.Flags().Changed("distro") || !persistent || !isolatedHome || !noEnter) {
				return fmt.Errorf("create --json requires NAME, --distro, --persistent, --isolated-home, and --no-enter")
			}
			if sharedHome {
				return fmt.Errorf("shared home is disabled: sandbox never mounts the host home")
			}
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			persistence := appState.config.DefaultPersistence
			if persistent {
				persistence = sandbox.Persistent
			}
			if disposable {
				persistence = sandbox.Disposable
			}
			homeMode := appState.config.DefaultHome
			if isolatedHome {
				homeMode = sandbox.IsolatedHome
			}

			interactive := len(args) == 0 || !cmd.Flags().Changed("distro") || (!persistent && !disposable) || !isolatedHome
			if interactive {
				answers, err := ui.PromptCreate(appState.config, appState.in, appState.out)
				if err != nil {
					return err
				}
				if name == "" {
					name = answers.Name
				}
				if !cmd.Flags().Changed("distro") {
					distro = answers.Distribution
				}
				if !persistent && !disposable {
					persistence = sandbox.Persistence(answers.Persistence)
				}
				if !isolatedHome && !sharedHome {
					homeMode = sandbox.HomeMode(answers.HomeMode)
				}
				if !cmd.Flags().Changed("no-enter") {
					noEnter = !answers.AutoEnter
				}
			}

			if persistent && disposable {
				return fmt.Errorf("choose only one of --persistent or --disposable")
			}
			if noEnter {
				if persistence == sandbox.Disposable {
					return fmt.Errorf("disposable sandboxes cannot use --no-enter")
				}
			}
			name, err := sandbox.ValidateName(name)
			if err != nil {
				return err
			}
			distroDef, ok := sandbox.FindDistribution(distro)
			if !ok {
				return fmt.Errorf("unknown distribution %q; choose arch, ubuntu, fedora or debian", distro)
			}
			if interactive && !yes {
				confirmed, err := ui.ConfirmCreate(name, appState.in, appState.out)
				if err != nil {
					return err
				}
				if !confirmed {
					return fmt.Errorf("sandbox creation cancelled")
				}
			}

			if !jsonOutput {
				appState.ui.Header("Creating " + name)
			}
			result, err := appState.manager.CreateWithResult(cmd.Context(), sandbox.CreateOptions{
				Name:         name,
				Distribution: distroDef,
				Persistence:  persistence,
				HomeMode:     homeMode,
				AutoEnter:    !noEnter,
				IfNotExists:  ifNotExists,
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				return json.NewEncoder(appState.out).Encode(struct {
					Name   string               `json:"name"`
					Result sandbox.CreateResult `json:"result"`
				}{Name: name, Result: result})
			}
			if result == sandbox.CreateResultRemoved {
				appState.ui.Success("Sandbox removed")
				return nil
			}
			appState.ui.Success("Sandbox ready")
			return nil
		},
	}
	cmd.Flags().StringVar(&distro, "distro", appState.config.DefaultDistro, "distribution: arch, ubuntu, fedora or debian")
	cmd.Flags().BoolVar(&persistent, "persistent", false, "keep the sandbox after exit")
	cmd.Flags().BoolVar(&disposable, "disposable", false, "remove the sandbox after exit")
	cmd.Flags().BoolVar(&isolatedHome, "isolated-home", false, "use a managed isolated home")
	cmd.Flags().BoolVar(&sharedHome, "shared-home", false, "deprecated: shared host home is disabled")
	cmd.Flags().BoolVar(&noEnter, "no-enter", false, "create without entering (persistent only)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmations")
	cmd.Flags().BoolVar(&ifNotExists, "if-not-exists", false, "succeed as a no-op when the sandbox already exists")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print successful result as JSON")
	return cmd
}
