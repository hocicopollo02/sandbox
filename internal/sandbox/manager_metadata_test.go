package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hocicopollo02/sandbox/internal/metadata"
)

// TestManagerCorruptMetadataIsActionableAndNonDestructive asserts that List
// and Info surface the corrupt record by name and never touch the runtime.
func TestManagerCorruptMetadataIsActionableAndNonDestructive(t *testing.T) {
	container := &fakeContainer{}
	store := metadata.NewStore(metadata.PathsFor(t.TempDir()))
	manager := NewManager(store, container, &fakeInspector{})

	if err := os.MkdirAll(store.Paths.Metadata, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Paths.Metadata, "corrupt.json"), []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.List(context.Background()); err == nil {
		t.Fatal("List() on corrupt metadata unexpectedly succeeded")
	} else if !strings.Contains(err.Error(), "corrupt") {
		t.Fatalf("List() error = %v, want the corrupt sandbox to be identified", err)
	}

	if _, err := manager.Info(context.Background(), "corrupt"); err == nil {
		t.Fatal("Info() on corrupt metadata unexpectedly succeeded")
	} else if !strings.Contains(err.Error(), "invalid metadata") {
		t.Fatalf("Info() error = %v, want invalid metadata error", err)
	}

	if len(container.deleted) != 0 || len(container.created) != 0 || len(container.entered) != 0 {
		t.Fatalf("corrupt metadata must not touch the runtime: deleted=%v created=%v entered=%v",
			container.deleted, container.created, container.entered)
	}
}
