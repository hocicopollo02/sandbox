package metadata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hocicopollo02/sandbox/internal/model"
)

// TestGetCorruptMetadataIsActionable asserts that an invalid JSON record fails
// with an error that names the affected sandbox.
func TestGetCorruptMetadataIsActionable(t *testing.T) {
	store := NewStore(PathsFor(t.TempDir()))
	if err := os.MkdirAll(store.Paths.Metadata, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Paths.Metadata, "corrupt.json"), []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := store.Get("corrupt")
	if err == nil {
		t.Fatal("Get() on corrupt metadata unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "invalid metadata") {
		t.Fatalf("Get() error = %v, want invalid metadata error", err)
	}
	if !strings.Contains(err.Error(), "corrupt") {
		t.Fatalf("Get() error = %v, want the sandbox name to be identified", err)
	}
}

// TestListIdentifiesCorruptRecord asserts that List fails with an actionable
// error naming the corrupt sandbox while the valid record stays untouched.
func TestListIdentifiesCorruptRecord(t *testing.T) {
	store := NewStore(PathsFor(t.TempDir()))
	if err := os.MkdirAll(store.Paths.Metadata, 0700); err != nil {
		t.Fatal(err)
	}
	valid := model.Record{
		Name: "healthy", Distribution: "arch", Image: "archlinux", Persistence: model.Persistent,
		HomeMode: model.IsolatedHome, CreatedAt: time.Now(),
	}
	if err := store.Save(valid); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Paths.Metadata, "corrupt.json"), []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := store.List()
	if err == nil {
		t.Fatal("List() on corrupt metadata unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "corrupt") {
		t.Fatalf("List() error = %v, want the corrupt sandbox to be identified", err)
	}
	if _, err := store.Get("healthy"); err != nil {
		t.Fatalf("valid record was affected by the corrupt one: %v", err)
	}
}
