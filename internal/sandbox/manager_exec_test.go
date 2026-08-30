package sandbox

import (
	"context"
	"reflect"
	"testing"
)

func TestExecPassesCommandUnmodified(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	if err := store.Save(Record{Name: "box", Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome}); err != nil {
		t.Fatal(err)
	}
	inspector := &fakeInspector{status: Running}
	manager.Inspector = inspector
	command := []string{"git", "clone", "--depth", "1", "https://example.com/repo.git"}
	if err := manager.Exec(context.Background(), "box", command); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"box", "git", "clone", "--depth", "1", "https://example.com/repo.git"}}
	if !reflect.DeepEqual(container.execCalls, want) {
		t.Fatalf("exec calls = %#v, want %#v", container.execCalls, want)
	}
}

func TestExecStoppedSandboxStillRunsCommand(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	if err := store.Save(Record{Name: "box", Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome}); err != nil {
		t.Fatal(err)
	}
	manager.Inspector = &fakeInspector{status: Stopped}
	if err := manager.Exec(context.Background(), "box", []string{"echo", "hi"}); err != nil {
		t.Fatal(err)
	}
	if len(container.execCalls) != 1 {
		t.Fatalf("exec calls = %#v, want the command to reach the runtime", container.execCalls)
	}
}

func TestExecUnknownSandboxFails(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	if err := store.Save(Record{Name: "box", Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome}); err != nil {
		t.Fatal(err)
	}
	manager.Inspector = &fakeInspector{status: Unknown}
	if err := manager.Exec(context.Background(), "box", []string{"echo", "hi"}); err == nil {
		t.Fatal("Exec() succeeded on an unknown sandbox")
	}
	if len(container.execCalls) != 0 {
		t.Fatalf("runtime was touched on unknown sandbox: %#v", container.execCalls)
	}
}

func TestExecMissingMetadataFails(t *testing.T) {
	container := &fakeContainer{}
	manager, _ := newTestManager(t, container)
	if err := manager.Exec(context.Background(), "ghost", []string{"echo", "hi"}); err == nil {
		t.Fatal("Exec() succeeded on a sandbox without metadata")
	}
	if len(container.execCalls) != 0 {
		t.Fatalf("runtime was touched without metadata: %#v", container.execCalls)
	}
}

func TestExecMissingRuntimeFails(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	if err := store.Save(Record{Name: "box", Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome}); err != nil {
		t.Fatal(err)
	}
	manager.Inspector = &fakeInspector{status: Missing}
	if err := manager.Exec(context.Background(), "box", []string{"echo", "hi"}); err == nil {
		t.Fatal("Exec() succeeded on a sandbox missing from the runtime")
	}
	if len(container.execCalls) != 0 {
		t.Fatalf("runtime was touched on missing sandbox: %#v", container.execCalls)
	}
}
