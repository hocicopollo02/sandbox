package cmd

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"strconv"

	"github.com/hocicopollo02/sandbox/internal/podman"
	"github.com/hocicopollo02/sandbox/internal/sandbox"
	"github.com/spf13/cobra"
)

func newDoctorCommand(appState *app) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose the local rootless container setup",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(appState)
		},
	}
}

func runDoctor(appState *app) error {
	appState.ui.Header("Sandbox Doctor")
	allGood := true

	podmanOK := true
	var infoData []byte
	if err := appState.podman.Available(); err != nil {
		appState.ui.Failure("Podman not installed")
		_, _ = fmt.Fprintln(appState.errOut, err)
		allGood = false
		podmanOK = false
	} else {
		appState.ui.Success("Podman installed")
	}
	if podmanOK {
		var err error
		infoData, err = appState.podman.Info(context.Background())
		if err != nil {
			appState.ui.Failure("Podman runtime working")
			_, _ = fmt.Fprintln(appState.errOut, err)
			allGood = false
		} else {
			appState.ui.Success("Podman runtime working")
		}
	}

	if !podmanOK {
		appState.ui.Failure("Podman runtime working")
	}
	if !podmanOK {
		appState.ui.Failure("Podman rootless")
	} else if os.Geteuid() == 0 {
		appState.ui.Failure("Podman rootless")
		_, _ = fmt.Fprintln(appState.errOut, "sandbox must run as a normal user; do not use sudo podman")
		allGood = false
	} else if rootless, err := podman.ParseRootless(infoData); err != nil || !rootless {
		appState.ui.Failure("Podman rootless")
		if err != nil {
			_, _ = fmt.Fprintln(appState.errOut, err)
		} else {
			_, _ = fmt.Fprintln(appState.errOut, "Podman is configured for rootful mode; sandbox requires rootless Podman")
		}
		allGood = false
	} else {
		appState.ui.Success("Podman rootless")
	}

	current, err := user.Current()
	if err != nil {
		return fmt.Errorf("find current user: %w", err)
	}
	hasSubUID := sandbox.SubIDConfigured("/etc/subuid", current.Username, strconv.Itoa(os.Getuid()))
	hasSubGID := sandbox.SubIDConfigured("/etc/subgid", current.Username, strconv.Itoa(os.Getuid()))
	if hasSubUID && hasSubGID {
		appState.ui.Success("User namespaces configured")
	} else {
		appState.ui.Failure("User namespaces configured")
		_, _ = fmt.Fprintf(appState.errOut, "configure ranges for %s in /etc/subuid and /etc/subgid\n", current.Username)
		allGood = false
	}

	if allGood {
		_, _ = fmt.Fprintln(appState.out, "\nEverything looks good.")
	} else {
		_, _ = fmt.Fprintln(appState.out, "\nSome checks failed. Fix them and run sandbox doctor again.")
	}
	return nil
}
