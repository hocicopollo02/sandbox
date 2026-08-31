package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strconv"

	"github.com/hocicopollo02/sandbox/internal/execx"
	"github.com/hocicopollo02/sandbox/internal/podman"
	"github.com/hocicopollo02/sandbox/internal/sandbox"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Name   string `json:"name"`
	Ok     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

func newDoctorCommand(appState *app) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose the local rootless container setup",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(appState, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

func runDoctor(appState *app, asJSON bool) error {
	checks, err := buildDoctorChecks(appState)
	if err != nil {
		return err
	}
	allGood := checksOK(checks)
	if asJSON {
		data, err := renderDoctorJSON(checks)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(appState.out, string(data)); err != nil {
			return err
		}
	} else {
		appState.ui.Header("Sandbox Doctor")
		for _, check := range checks {
			if check.Ok {
				appState.ui.Success(check.Name)
			} else {
				appState.ui.Failure(check.Name)
				if check.Detail != "" {
					_, _ = fmt.Fprintln(appState.errOut, check.Detail)
				}
			}
		}
		if allGood {
			_, _ = fmt.Fprintln(appState.out, "\nEverything looks good.")
		} else {
			_, _ = fmt.Fprintln(appState.out, "\nSome checks failed. Fix them and run sandbox doctor again.")
		}
	}
	if !allGood {
		return &execx.ExitError{Code: 1}
	}
	return nil
}

func buildDoctorChecks(appState *app) ([]doctorCheck, error) {
	var infoData []byte
	podmanErr := appState.podman.Available()
	podmanOK := podmanErr == nil

	runtimeOK := false
	var runtimeDetail string
	if podmanOK {
		var err error
		infoData, err = appState.podman.Info(context.Background())
		if err == nil {
			runtimeOK = true
		} else {
			runtimeDetail = err.Error()
		}
	}

	rootlessOK := false
	var rootlessDetail string
	if podmanOK && runtimeOK {
		switch {
		case os.Geteuid() == 0:
			rootlessDetail = "sandbox must run as a normal user; do not use sudo podman"
		default:
			rootless, err := podman.ParseRootless(infoData)
			if err != nil {
				rootlessDetail = err.Error()
			} else if !rootless {
				rootlessDetail = "Podman is configured for rootful mode; sandbox requires rootless Podman"
			} else {
				rootlessOK = true
			}
		}
	}

	nsOK := false
	var nsDetail string
	current, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("find current user: %w", err)
	}
	hasSubUID := sandbox.SubIDConfigured("/etc/subuid", current.Username, strconv.Itoa(os.Getuid()))
	hasSubGID := sandbox.SubIDConfigured("/etc/subgid", current.Username, strconv.Itoa(os.Getuid()))
	if hasSubUID && hasSubGID {
		nsOK = true
	} else {
		nsDetail = fmt.Sprintf("configure ranges for %s in /etc/subuid and /etc/subgid", current.Username)
	}

	return []doctorCheck{
		{Name: "Podman installed", Ok: podmanOK, Detail: errText(podmanErr)},
		{Name: "Podman runtime working", Ok: runtimeOK, Detail: runtimeDetail},
		{Name: "Podman rootless", Ok: rootlessOK, Detail: rootlessDetail},
		{Name: "User namespaces configured", Ok: nsOK, Detail: nsDetail},
	}, nil
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func checksOK(checks []doctorCheck) bool {
	for _, check := range checks {
		if !check.Ok {
			return false
		}
	}
	return true
}

func renderDoctorJSON(checks []doctorCheck) ([]byte, error) {
	output := struct {
		OK     bool          `json:"ok"`
		Checks []doctorCheck `json:"checks"`
	}{OK: checksOK(checks), Checks: checks}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode doctor JSON: %w", err)
	}
	return data, nil
}
