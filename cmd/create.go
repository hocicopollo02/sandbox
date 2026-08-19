package cmd

import (
	"fmt"

	"github.com/pablo/sandbox/internal/sandbox"
	"github.com/pablo/sandbox/internal/ui"
	"github.com/spf13/cobra"
)

func newCreateCommand(appState *app) *cobra.Command {
	var distro string
	var persistent, disposable bool
	var isolatedHome, sharedHome bool
	var noEnter, yes bool

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
			if sharedHome {
				homeMode = sandbox.SharedHome
			}

			interactive := len(args) == 0 || !cmd.Flags().Changed("distro") || (!persistent && !disposable) || (!isolatedHome && !sharedHome)
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
			if isolatedHome && sharedHome {
				return fmt.Errorf("choose only one of --isolated-home or --shared-home")
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

			if homeMode == sandbox.SharedHome && !yes {
				confirmed, err := ui.ConfirmSharedHome(appState.in, appState.out)
				if err != nil {
					return err
				}
				if !confirmed {
					return fmt.Errorf("shared home cancelled")
				}
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

			appState.ui.Header("Creating " + name)
			disposableSession, err := appState.manager.Create(cmd.Context(), sandbox.CreateOptions{
				Name:         name,
				Distribution: distroDef,
				Persistence:  persistence,
				HomeMode:     homeMode,
				AutoEnter:    !noEnter,
			})
			if err != nil {
				return err
			}
			if disposableSession {
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
	cmd.Flags().BoolVar(&sharedHome, "shared-home", false, "use the host home directory")
	cmd.Flags().BoolVar(&noEnter, "no-enter", false, "create without entering (persistent only)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmations")
	return cmd
}
