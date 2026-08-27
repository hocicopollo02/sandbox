package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hocicopollo02/sandbox/internal/metadata"
)

// saveGhost writes a persistent record whose container no longer exists in
// the runtime. The default fakeInspector reports Missing, matching that state.
func saveGhost(t *testing.T, store *metadata.Store) {
	t.Helper()
	if err := os.MkdirAll(store.Home("ghost"), 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(store.Home("ghost"), "data.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	record := Record{
		Name: "ghost", Distribution: "arch", Image: "archlinux", Persistence: Persistent,
		HomeMode: IsolatedHome, HomePath: store.Home("ghost"), CreatedAt: time.Now().In(time.Local),
	}
	if err := store.Save(record); err != nil {
		t.Fatal(err)
	}
}

func TestStaleMetadataListReportsMissing(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	saveGhost(t, store)

	entries, err := manager.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "ghost" || entries[0].Status != Missing {
		t.Fatalf("list entries = %#v, want ghost with missing status", entries)
	}
}

func TestStaleMetadataInfoReportsMissing(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	saveGhost(t, store)

	info, err := manager.Info(context.Background(), "ghost")
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != Missing {
		t.Fatalf("Info() status = %q, want missing", info.Status)
	}
}

func TestStaleMetadataEnterFailsWithoutRecreating(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	saveGhost(t, store)

	err := manager.Enter(context.Background(), "ghost")
	if err == nil || !strings.Contains(err.Error(), "does not exist in the container runtime") {
		t.Fatalf("Enter() error = %v, want missing runtime error", err)
	}
	if len(container.created) != 0 || len(container.entered) != 0 || len(container.deleted) != 0 {
		t.Fatalf("Enter touched the runtime: created=%v entered=%v deleted=%v", container.created, container.entered, container.deleted)
	}
}

func TestStaleMetadataStopFailsWithoutRuntimeCalls(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	saveGhost(t, store)

	err := manager.Stop(context.Background(), "ghost")
	if err == nil || !strings.Contains(err.Error(), "does not exist in the container runtime") {
		t.Fatalf("Stop() error = %v, want missing runtime error", err)
	}
	if len(container.created) != 0 || len(container.entered) != 0 || len(container.deleted) != 0 {
		t.Fatalf("Stop touched the runtime: %v", container)
	}
}

func TestStaleMetadataDeleteRemovesMetadataAndHome(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	saveGhost(t, store)

	if err := manager.Delete(context.Background(), "ghost", DeleteOptions{DeleteHome: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("ghost"); !errors.Is(err, metadata.ErrNotFound) {
		t.Fatalf("metadata was not removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Home("ghost"), "data.txt")); !os.IsNotExist(err) {
		t.Fatalf("home was not removed: %v", err)
	}
	if len(container.deleted) != 0 {
		t.Fatalf("Delete issued a container removal for a missing container: %v", container.deleted)
	}
}
