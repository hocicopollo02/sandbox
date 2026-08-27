package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hocicopollo02/sandbox/internal/model"
)

func TestStoreRoundTripAndAtomicFileMode(t *testing.T) {
	store := NewStore(PathsFor(t.TempDir()))
	record := model.Record{
		Name: "gentle-ai", Distribution: "arch", Image: "archlinux", Persistence: model.Persistent,
		HomeMode: model.IsolatedHome, HomePath: store.Home("gentle-ai"), CreatedAt: time.Now(),
	}
	if err := store.Save(record); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("gentle-ai")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != record.Name || got.Image != record.Image {
		t.Fatalf("round trip = %#v", got)
	}
	info, err := os.Stat(filepath.Join(store.Paths.Metadata, "gentle-ai.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("metadata mode = %o, want 600", info.Mode().Perm())
	}
}

func TestRemoveHomeUnwritableCache(t *testing.T) {
	store := NewStore(PathsFor(t.TempDir()))
	// Go's module cache keeps files read-only (0444) and directories 0555, which
	// blocks unlink from the host.
	dir := filepath.Join(store.Home("cached"), "mod", "cache", "workflows")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ci.yml"), []byte("x"), 0444); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Dir(dir), dir} {
		if err := os.Chmod(path, 0555); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.RemoveHome("cached"); err != nil {
		t.Fatalf("RemoveHome() = %v, want success after granting write permission", err)
	}
	if _, err := os.Stat(store.Home("cached")); !os.IsNotExist(err) {
		t.Fatalf("managed home still exists: %v", err)
	}
}

func TestRemoveHomeStaysInsideManagedRoot(t *testing.T) {
	store := NewStore(PathsFor(t.TempDir()))
	if err := os.MkdirAll(store.Home("gentle-ai"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Home("gentle-ai"), "data"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := store.RemoveHome("gentle-ai"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(store.Home("gentle-ai")); !os.IsNotExist(err) {
		t.Fatalf("managed home still exists: %v", err)
	}
}
