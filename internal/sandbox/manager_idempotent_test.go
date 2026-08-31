package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hocicopollo02/sandbox/internal/metadata"
)

type unavailableContainer struct {
	fakeContainer
	err error
}

func (c *unavailableContainer) Available() error { return c.err }

type unavailableInspector struct {
	fakeInspector
	err error
}

func (i *unavailableInspector) Status(_ context.Context, _ string) (Status, error) {
	return Missing, i.err
}

func TestCreateDuplicateWithIfNotExistsIsNoOp(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	opts := CreateOptions{Name: "dup", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome}
	if _, err := manager.Create(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	manager.Inspector.(*fakeInspector).status = Running
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

func TestCreateStaleReservationWithIfNotExistsIsRecreated(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	// Simulate the winner reserving the name before the loser reaches SaveExclusive.
	record := Record{Name: "raced", Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome}
	if err := store.SaveExclusive(record); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.Home("raced"), 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(store.Home("raced"), "orphan")
	if err := os.WriteFile(marker, []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	manager.Inspector = &fakeInspector{}
	_, err := manager.Create(context.Background(), CreateOptions{
		Name: "raced", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome, IfNotExists: true,
	})
	if err != nil {
		t.Fatalf("stale reservation with IfNotExists = %v, want nil", err)
	}
	if len(container.created) != 1 {
		t.Fatalf("recreated containers = %v, want one", container.created)
	}
	if _, err := store.Get("raced"); err != nil {
		t.Fatalf("recreated metadata missing: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("stale home contents remain: %v", err)
	}
}

func TestCreateStaleReservationDoesNotDeleteNewReservation(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	old := Record{Name: "raced", Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome}
	if err := store.SaveExclusive(old); err != nil {
		t.Fatal(err)
	}
	newRecord := old
	newRecord.CreatedAt = time.Now().Add(time.Second)
	manager.Inspector = &racingInspector{during: func() {
		if err := store.Delete(old.Name); err != nil {
			t.Errorf("old reservation removal failed: %v", err)
		}
		if err := store.SaveExclusive(newRecord); err != nil {
			t.Errorf("new reservation failed: %v", err)
		}
	}}
	_, err := manager.Create(context.Background(), CreateOptions{
		Name: old.Name, Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome, IfNotExists: true,
	})
	if err != nil {
		t.Fatalf("stale reservation race = %v, want nil", err)
	}
	if len(container.created) != 0 {
		t.Fatalf("created containers = %v, want none", container.created)
	}
	if record, err := store.Get(old.Name); err != nil || !record.CreatedAt.Equal(newRecord.CreatedAt) {
		t.Fatalf("reservation after race = %#v, %v; want new reservation", record, err)
	}
}

func TestCreateExistingWithIfNotExistsIgnoresRuntimeUnavailable(t *testing.T) {
	container := &unavailableContainer{err: fmt.Errorf("container runtime unavailable")}
	manager, store := newTestManager(t, &container.fakeContainer)
	manager.Container = container
	manager.Inspector = &unavailableInspector{err: fmt.Errorf("inspection unavailable")}
	distro, _ := FindDistribution("arch")
	if err := store.Save(Record{Name: "existing", Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Create(context.Background(), CreateOptions{
		Name: "existing", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome, IfNotExists: true,
	}); err != nil {
		t.Fatalf("existing sandbox with unavailable runtime = %v, want nil", err)
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
