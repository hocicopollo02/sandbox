package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/hocicopollo02/sandbox/internal/config"
	"github.com/hocicopollo02/sandbox/internal/execx"
	"github.com/hocicopollo02/sandbox/internal/metadata"
	"github.com/hocicopollo02/sandbox/internal/model"
	"github.com/hocicopollo02/sandbox/internal/podman"
	core "github.com/hocicopollo02/sandbox/internal/sandbox"
	"github.com/hocicopollo02/sandbox/internal/ui"
	"github.com/spf13/cobra"
)

var (
	Version   = "1.1.0"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// ErrorFormat selects the error output format for agent integrations.
var ErrorFormat = "text"

type app struct {
	manager *core.Manager
	config  config.Config
	runner  *execx.CommandRunner
	podman  *podman.Client
	ui      ui.Printer
	in      io.Reader
	out     io.Writer
	errOut  io.Writer
	verbose bool
}

func NewRootCommand(home string, in io.Reader, out, errOut io.Writer) (*cobra.Command, error) {
	cfg, err := config.Load(home)
	if err != nil {
		return nil, err
	}
	paths := metadata.PathsFor(home)
	runner := &execx.CommandRunner{Out: out, Err: errOut}
	appState := &app{
		config: cfg,
		runner: runner,
		ui:     ui.New(out, errOut),
		in:     in,
		out:    out,
		errOut: errOut,
	}
	appState.podman = podman.New(runner)
	appState.manager = core.NewManager(
		metadata.NewStore(paths),
		appState.podman,
		appState.podman,
	)

	root := &cobra.Command{
		Use:           "sandbox",
		Short:         "Create and manage local Linux sandboxes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.PersistentFlags().BoolVar(&appState.verbose, "verbose", false, "show external command details")
	root.PersistentFlags().StringVar(&ErrorFormat, "error-format", "text", "error output format: text or json")
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		runner.Verbose = appState.verbose
		if os.Geteuid() == 0 && cmd.Name() != "doctor" && cmd.Name() != "version" {
			return fmt.Errorf("sandbox must run as a normal user; do not use sudo sandbox or sudo podman")
		}
		return nil
	}

	root.AddCommand(
		newCreateCommand(appState),
		newListCommand(appState),
		newEnterCommand(appState),
		newExecCommand(appState),
		newStopCommand(appState),
		newDeleteCommand(appState),
		newInfoCommand(appState),
		newDoctorCommand(appState),
		newVersionCommand(appState),
	)
	return root, nil
}

func Execute() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find home directory: %w", err)
	}
	root, err := NewRootCommand(home, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	root.SetContext(ctx)
	return root.Execute()
}

// renderError returns the machine JSON line when format is json, or the plain
// message otherwise. The bool reports whether the caller should use the machine
// line instead of the human message.
func renderError(err error, format string) (string, bool) {
	if format != "json" {
		return "", false
	}
	payload := struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}
	payload.Error.Code = model.ErrorCode(err)
	payload.Error.Message = err.Error()
	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return "", false
	}
	return string(data), true
}

// RenderError exposes renderError for main's exit handling.
func RenderError(err error, format string) (string, bool) {
	return renderError(err, format)
}
