package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/hocicopollo02/sandbox/internal/config"
	"github.com/hocicopollo02/sandbox/internal/metadata"
	"github.com/hocicopollo02/sandbox/internal/model"
	"github.com/hocicopollo02/sandbox/internal/sandbox"
	"github.com/hocicopollo02/sandbox/internal/ui"
)

type lifecycleContainer struct {
	created []string
	stopped []string
	deleted []string
}

func (f *lifecycleContainer) Available() error { return nil }
func (f *lifecycleContainer) Create(_ context.Context, name, _, _ string) error {
	f.created = append(f.created, name)
	return nil
}
func (f *lifecycleContainer) Enter(context.Context, string) error          { return nil }
func (f *lifecycleContainer) Exec(context.Context, string, []string) error { return nil }
func (f *lifecycleContainer) Delete(_ context.Context, name string) error {
	f.deleted = append(f.deleted, name)
	return nil
}
func (f *lifecycleContainer) Stop(_ context.Context, name string) error {
	f.stopped = append(f.stopped, name)
	return nil
}

type lifecycleInspector struct{ status sandbox.Status }

func (f *lifecycleInspector) Available() error { return nil }
func (f *lifecycleInspector) Status(context.Context, string) (sandbox.Status, error) {
	return f.status, nil
}
func (f *lifecycleInspector) Statuses(context.Context, []string) (map[string]sandbox.Status, error) {
	return nil, nil
}

func newLifecycleTestApp(t *testing.T, status sandbox.Status) (*app, *bytes.Buffer, *metadata.Store, *lifecycleContainer) {
	t.Helper()
	out := &bytes.Buffer{}
	store := metadata.NewStore(metadata.PathsFor(t.TempDir()))
	container := &lifecycleContainer{}
	return &app{
		manager: sandbox.NewManager(store, container, &lifecycleInspector{status: status}),
		config:  config.Config{},
		ui:      ui.New(out, &bytes.Buffer{}),
		in:      bytes.NewBuffer(nil),
		out:     out,
		errOut:  &bytes.Buffer{},
	}, out, store, container
}

func TestStopJSONEmitsOneSuccessObject(t *testing.T) {
	appState, out, store, container := newLifecycleTestApp(t, sandbox.Running)
	if err := store.Save(model.Record{Name: "gentle-ai"}); err != nil {
		t.Fatal(err)
	}
	cmd := newStopCommand(appState)
	cmd.SetArgs([]string{"gentle-ai", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	decoder := json.NewDecoder(out)
	if err := decoder.Decode(&got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out.String())
	}
	if decoder.Decode(&struct{}{}) == nil {
		t.Fatalf("stdout contains more than one JSON value: %q", out.String())
	}
	if got["name"] != "gentle-ai" || got["result"] != "stopped" || len(got) != 2 {
		t.Fatalf("JSON = %#v, want name and stopped result", got)
	}
	if len(container.stopped) != 1 {
		t.Fatalf("stop calls = %v, want one", container.stopped)
	}
}

func TestDeleteJSONKeepsHomeAndEmitsOneSuccessObject(t *testing.T) {
	appState, out, store, container := newLifecycleTestApp(t, sandbox.Stopped)
	if err := os.MkdirAll(store.Home("gentle-ai"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(model.Record{Name: "gentle-ai", HomeMode: sandbox.IsolatedHome, HomePath: store.Home("gentle-ai")}); err != nil {
		t.Fatal(err)
	}
	cmd := newDeleteCommand(appState)
	cmd.SetArgs([]string{"gentle-ai", "--yes", "--keep-home", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Name         string  `json:"name"`
		Result       string  `json:"result"`
		RetainedHome *string `json:"retained_home"`
	}
	decoder := json.NewDecoder(out)
	if err := decoder.Decode(&got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out.String())
	}
	if decoder.Decode(&struct{}{}) == nil {
		t.Fatalf("stdout contains more than one JSON value: %q", out.String())
	}
	if got.Name != "gentle-ai" || got.Result != "deleted" || got.RetainedHome == nil || *got.RetainedHome != store.Home("gentle-ai") {
		t.Fatalf("JSON = %#v, want deleted result with retained home %q", got, store.Home("gentle-ai"))
	}
	if len(container.deleted) != 1 {
		t.Fatalf("delete calls = %v, want one", container.deleted)
	}
	if _, err := os.Stat(store.Home("gentle-ai")); err != nil {
		t.Fatalf("retained home is not present: %v", err)
	}
}

func TestCreateJSONWithCompleteFlagsEmitsOneSuccessObject(t *testing.T) {
	appState, out, _, container := newLifecycleTestApp(t, sandbox.Missing)
	cmd := newCreateCommand(appState)
	cmd.SetArgs([]string{"gentle-ai", "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	decoder := json.NewDecoder(out)
	if err := decoder.Decode(&got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out.String())
	}
	if decoder.Decode(&struct{}{}) == nil {
		t.Fatalf("stdout contains more than one JSON value: %q", out.String())
	}
	if got["name"] != "gentle-ai" || got["result"] != "created" || len(got) != 2 {
		t.Fatalf("JSON = %#v, want name and created result", got)
	}
	if len(container.created) != 1 {
		t.Fatalf("create calls = %v, want one", container.created)
	}
}

func TestLifecycleJSONReportsNoOpsAsUnchanged(t *testing.T) {
	t.Run("stop already stopped", func(t *testing.T) {
		appState, out, store, container := newLifecycleTestApp(t, sandbox.Stopped)
		if err := store.Save(model.Record{Name: "gentle-ai"}); err != nil {
			t.Fatal(err)
		}
		cmd := newStopCommand(appState)
		cmd.SetArgs([]string{"gentle-ai", "--json"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.String() != "{\"name\":\"gentle-ai\",\"result\":\"unchanged\"}\n" {
			t.Fatalf("stdout = %q", out.String())
		}
		if len(container.stopped) != 0 {
			t.Fatalf("stop calls = %v, want none", container.stopped)
		}
	})

	t.Run("delete missing with if-exists", func(t *testing.T) {
		appState, out, _, _ := newLifecycleTestApp(t, sandbox.Missing)
		cmd := newDeleteCommand(appState)
		cmd.SetArgs([]string{"gentle-ai", "--yes", "--if-exists", "--json"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.String() != "{\"name\":\"gentle-ai\",\"result\":\"unchanged\",\"retained_home\":null}\n" {
			t.Fatalf("stdout = %q", out.String())
		}
	})

	t.Run("create existing with if-not-exists", func(t *testing.T) {
		appState, out, store, container := newLifecycleTestApp(t, sandbox.Running)
		if err := store.Save(model.Record{Name: "gentle-ai"}); err != nil {
			t.Fatal(err)
		}
		cmd := newCreateCommand(appState)
		cmd.SetArgs([]string{"gentle-ai", "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes", "--if-not-exists", "--json"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.String() != "{\"name\":\"gentle-ai\",\"result\":\"unchanged\"}\n" {
			t.Fatalf("stdout = %q", out.String())
		}
		if len(container.created) != 0 {
			t.Fatalf("create calls = %v, want none", container.created)
		}
	})
}

func TestLifecycleJSONRejectsInteractiveOrUnconfirmedModes(t *testing.T) {
	t.Run("create cannot open the wizard", func(t *testing.T) {
		appState, out, _, _ := newLifecycleTestApp(t, sandbox.Missing)
		cmd := newCreateCommand(appState)
		cmd.SetArgs([]string{"gentle-ai", "--json"})
		if err := cmd.Execute(); err == nil {
			t.Fatal("create --json without explicit flags succeeded")
		}
		if out.Len() != 0 {
			t.Fatalf("create --json wrote interactive output: %q", out.String())
		}
	})

	t.Run("delete requires confirmation bypass", func(t *testing.T) {
		appState, out, _, _ := newLifecycleTestApp(t, sandbox.Missing)
		cmd := newDeleteCommand(appState)
		cmd.SetArgs([]string{"gentle-ai", "--json"})
		if err := cmd.Execute(); err == nil {
			t.Fatal("delete --json without --yes succeeded")
		}
		if out.Len() != 0 {
			t.Fatalf("delete --json wrote confirmation output: %q", out.String())
		}
	})
}
