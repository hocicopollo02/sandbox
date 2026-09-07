package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/hocicopollo02/sandbox/internal/config"
	"github.com/hocicopollo02/sandbox/internal/metadata"
	"github.com/hocicopollo02/sandbox/internal/model"
	"github.com/hocicopollo02/sandbox/internal/sandbox"
	"github.com/hocicopollo02/sandbox/internal/ui"
)

type blockingContainer struct {
	lifecycleContainer
}

func (f *blockingContainer) Exec(ctx context.Context, name string, command []string) error {
	<-ctx.Done()
	return ctx.Err()
}

func (f *blockingContainer) Stop(ctx context.Context, name string) error {
	<-ctx.Done()
	return ctx.Err()
}

func (f *blockingContainer) Create(ctx context.Context, name, _, _ string) error {
	<-ctx.Done()
	return ctx.Err()
}

func newBlockingTestApp(t *testing.T, status sandbox.Status) (*app, *bytes.Buffer, *metadata.Store, *blockingContainer) {
	t.Helper()
	out := &bytes.Buffer{}
	store := metadata.NewStore(metadata.PathsFor(t.TempDir()))
	container := &blockingContainer{}
	return &app{
		manager: sandbox.NewManager(store, container, &lifecycleInspector{status: status}),
		config:  config.Config{},
		ui:      ui.New(out, &bytes.Buffer{}),
		in:      bytes.NewBuffer(nil),
		out:     out,
		errOut:  &bytes.Buffer{},
	}, out, store, container
}

func TestExecTimeoutBoundsGuestCommand(t *testing.T) {
	appState, _, store, _ := newBlockingTestApp(t, sandbox.Running)
	if err := store.Save(model.Record{Name: "box"}); err != nil {
		t.Fatal(err)
	}
	cmd := newExecCommand(appState)
	cmd.SetArgs([]string{"--timeout", "1ms", "box", "--", "echo", "hi"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("exec --timeout did not bound the guest command")
	}
}

func TestStopTimeoutBoundsRuntimeCall(t *testing.T) {
	appState, _, store, _ := newBlockingTestApp(t, sandbox.Running)
	if err := store.Save(model.Record{Name: "box"}); err != nil {
		t.Fatal(err)
	}
	cmd := newStopCommand(appState)
	cmd.SetArgs([]string{"--timeout", "1ms", "box"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("stop --timeout did not bound the runtime call")
	}
}

func TestCreateTimeoutBoundsAutomatedCreate(t *testing.T) {
	appState, _, _, _ := newBlockingTestApp(t, sandbox.Missing)
	cmd := newCreateCommand(appState)
	cmd.SetArgs([]string{"box", "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes", "--timeout", "1ms"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("create --timeout did not bound the automated create")
	}
}
