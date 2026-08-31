package sandbox

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/hocicopollo02/sandbox/internal/metadata"
)

func TestCreateDuplicateWithIfNotExistsIsNoOp(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	opts := CreateOptions{Name: "dup", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome}
	if _, err := manager.Create(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	opts.IfNotExists = true
	removed, err := manager.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("Create with IfNotExists = %v, want nil error", err)
	}
	if removed {
		t.Fatal("Create with IfNotExists reported a disposable removal")
	}
	if len(container.created) != 1 {
		t.Fatalf("container creates = %v, want exactly one", container.created)
	}
	if _, err := store.Get("dup"); err != nil {
		t.Fatalf("existing metadata was damaged: %v", err)
	}
}

func TestCreateDuplicateWithoutIfNotExistsStillFails(t *testing.T) {
	container := &fakeContainer{}
	manager, _ := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	opts := CreateOptions{Name: "dup", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome}
	if _, err := manager.Create(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Create(context.Background(), opts); err == nil {
		t.Fatal("duplicate Create() returned nil without IfNotExists")
	}
	if len(container.created) != 1 {
		t.Fatalf("container creates = %v, want exactly one", container.created)
	}
}

func TestCreateRaceLoserWithIfNotExistsIsNoOp(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	// Simulate the winner reserving the name before the loser reaches SaveExclusive.
	record := Record{Name: "raced", Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome}
	if err := store.SaveExclusive(record); err != nil {
		t.Fatal(err)
	}
	// Give the fake inspector a Missing answer so the loser passes the runtime check.
	manager.Inspector = &fakeInspector{}
	_, err := manager.Create(context.Background(), CreateOptions{
		Name: "raced", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome, IfNotExists: true,
	})
	if err != nil {
		t.Fatalf("race loser with IfNotExists = %v, want nil", err)
	}
	if len(container.created) != 0 {
		t.Fatalf("loser created a container: %v", container.created)
	}
}

func TestDeleteMissingWithIfExistsIsNoOp(t *testing.T) {
	manager, _ := newTestManager(t, &fakeContainer{})
	if err := manager.Delete(context.Background(), "ghost", DeleteOptions{IfExists: true}); err != nil {
		t.Fatalf("Delete missing with IfExists = %v, want nil", err)
	}
}

func TestDeleteMissingWithoutIfExistsFails(t *testing.T) {
	manager, _ := newTestManager(t, &fakeContainer{})
	err := manager.Delete(context.Background(), "ghost", DeleteOptions{})
	if err == nil {
		t.Fatal("Delete missing returned nil without IfExists")
	}
}

func TestDeleteMissingWithIfExistsStillRemovesHomeWhenMetadataExists(t *testing.T) {
	manager, store := newTestManager(t, &fakeContainer{})
	distro, _ := FindDistribution("arch")
	record := Record{Name: "ghost", Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome, HomePath: store.Home("ghost")}
	if err := store.Save(record); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.Home("ghost"), 0700); err != nil {
		t.Fatal(err)
	}
	// Inspect via the real manager path: Delete sees Missing status (fake inspector) and removes metadata+home.
	if err := manager.Delete(context.Background(), "ghost", DeleteOptions{DeleteHome: true, IfExists: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("ghost"); !errors.Is(err, metadata.ErrNotFound) {
		t.Fatalf("metadata not removed: %v", err)
	}
	if _, err := os.Stat(store.Home("ghost")); !os.IsNotExist(err) {
		t.Fatalf("home not removed: %v", err)
	}
}
