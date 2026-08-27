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

// retryContainer fails the first container deletion and then succeeds, so the
// delete retry contract can be exercised with a real store and home.
type retryContainer struct {
	failFirst bool
	deletes   []string
}

func (r *retryContainer) Available() error { return nil }
func (r *retryContainer) Create(context.Context, string, string, string) error {
	return nil
}
func (r *retryContainer) Enter(context.Context, string) error { return nil }
func (r *retryContainer) Stop(context.Context, string) error  { return nil }
func (r *retryContainer) Delete(_ context.Context, name string) error {
	r.deletes = append(r.deletes, name)
	if r.failFirst && len(r.deletes) == 1 {
		return errors.New("podman rm: device or resource busy")
	}
	return nil
}

// seqInspector returns each configured status in order, then Missing forever.
type seqInspector struct {
	statuses []Status
	calls    int
}

func (s *seqInspector) Available() error { return nil }
func (s *seqInspector) Status(_ context.Context, _ string) (Status, error) {
	if s.calls >= len(s.statuses) {
		return Missing, nil
	}
	status := s.statuses[s.calls]
	s.calls++
	return status, nil
}

func (s *seqInspector) Statuses(_ context.Context, names []string) (map[string]Status, error) {
	out := make(map[string]Status, len(names))
	for _, name := range names {
		out[name] = Missing
	}
	return out, nil
}

func newRetryManager(t *testing.T, container *retryContainer, inspector *seqInspector) (*Manager, *metadata.Store) {
	t.Helper()
	store := metadata.NewStore(metadata.PathsFor(t.TempDir()))
	return NewManager(store, container, inspector), store
}

func saveRetrySandbox(t *testing.T, store *metadata.Store, name string) string {
	t.Helper()
	record := Record{
		Name: name, Distribution: "arch", Image: "archlinux",
		Persistence: Persistent, HomeMode: IsolatedHome,
		HomePath: store.Home(name), CreatedAt: time.Now().In(time.Local),
	}
	if err := store.Save(record); err != nil {
		t.Fatal(err)
	}
	home := store.Home(name)
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(home, "data.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	return marker
}

func TestDeleteRetryRetainsMetadataAndHomeUntilCleanupSucceeds(t *testing.T) {
	container := &retryContainer{failFirst: true}
	inspector := &seqInspector{statuses: []Status{Stopped, Missing}}
	manager, store := newRetryManager(t, container, inspector)
	marker := saveRetrySandbox(t, store, "broken-retry")

	err := manager.Delete(context.Background(), "broken-retry", DeleteOptions{DeleteHome: true})
	if err == nil || !strings.Contains(err.Error(), "broken-retry") {
		t.Fatalf("first Delete() = %v, want error identifying the sandbox", err)
	}
	if _, err := store.Get("broken-retry"); err != nil {
		t.Fatalf("metadata must be retained after failed delete: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("home must be retained after failed delete: %v", err)
	}

	if err := manager.Delete(context.Background(), "broken-retry", DeleteOptions{DeleteHome: true}); err != nil {
		t.Fatalf("retry Delete() = %v, want success", err)
	}
	if _, err := store.Get("broken-retry"); !errors.Is(err, metadata.ErrNotFound) {
		t.Fatalf("retry did not remove metadata: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("retry did not remove home: %v", err)
	}
	if len(container.deletes) != 1 {
		t.Fatalf("container deletes = %v, want exactly one (retry must skip missing runtime)", container.deletes)
	}
}

func TestFailedDeleteLeavesOtherSandboxIntact(t *testing.T) {
	container := &retryContainer{failFirst: true}
	inspector := &seqInspector{statuses: []Status{Stopped, Missing}}
	manager, store := newRetryManager(t, container, inspector)
	brokenMarker := saveRetrySandbox(t, store, "broken-retry")
	healthyMarker := saveRetrySandbox(t, store, "healthy")

	if err := manager.Delete(context.Background(), "broken-retry", DeleteOptions{DeleteHome: true}); err == nil {
		t.Fatal("first Delete() unexpectedly succeeded")
	}
	if _, err := os.Stat(healthyMarker); err != nil {
		t.Fatalf("healthy sandbox home damaged by failed delete: %v", err)
	}
	if _, err := store.Get("healthy"); err != nil {
		t.Fatalf("healthy sandbox metadata damaged by failed delete: %v", err)
	}
	if _, err := os.Stat(brokenMarker); err != nil {
		t.Fatalf("broken sandbox home removed before retry: %v", err)
	}
}
