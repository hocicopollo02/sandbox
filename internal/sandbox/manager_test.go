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

type fakeContainer struct {
	createErr error
	enterErr  error
	deleteErr error
	created   []string
	entered   []string
	deleted   []string
}

func (f *fakeContainer) Available() error { return nil }
func (f *fakeContainer) Create(_ context.Context, name, image, home string) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, name+"|"+image+"|"+home)
	return nil
}
func (f *fakeContainer) Enter(_ context.Context, name string) error {
	f.entered = append(f.entered, name)
	return f.enterErr
}
func (f *fakeContainer) Stop(_ context.Context, name string) error { return nil }
func (f *fakeContainer) Delete(_ context.Context, name string) error {
	f.deleted = append(f.deleted, name)
	return f.deleteErr
}

type fakeInspector struct {
	status    Status
	statuses  map[string]Status
	batches   int
	lastBatch []string
}

func (f *fakeInspector) Available() error { return nil }
func (f *fakeInspector) Status(_ context.Context, name string) (Status, error) {
	if f.status == "" {
		return Missing, nil
	}
	return f.status, nil
}
func (f *fakeInspector) Statuses(_ context.Context, names []string) (map[string]Status, error) {
	f.batches++
	f.lastBatch = append([]string(nil), names...)
	out := make(map[string]Status, len(names))
	for _, name := range names {
		if status, ok := f.statuses[name]; ok {
			out[name] = status
		} else {
			out[name] = Missing
		}
	}
	return out, nil
}

func newTestManager(t *testing.T, container *fakeContainer) (*Manager, *metadata.Store) {
	t.Helper()
	store := metadata.NewStore(metadata.PathsFor(t.TempDir()))
	return NewManager(store, container, &fakeInspector{}), store
}

func TestCreatePersistentCreatesIsolatedHomeAndMetadata(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	if _, err := manager.Create(context.Background(), CreateOptions{
		Name: "gentle-ai", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome,
	}); err != nil {
		t.Fatal(err)
	}
	if len(container.created) != 1 || container.created[0] != "gentle-ai|docker.io/library/archlinux:latest|"+store.Home("gentle-ai") {
		t.Fatalf("create calls = %#v", container.created)
	}
	if _, err := store.Get("gentle-ai"); err != nil {
		t.Fatalf("metadata was not saved: %v", err)
	}
	if _, err := os.Stat(store.Home("gentle-ai")); err != nil {
		t.Fatalf("isolated home was not created: %v", err)
	}
}

func TestCreateRefusesSymlinkedManagedHomes(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	external := t.TempDir()
	if err := os.MkdirAll(store.Paths.DataRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, store.Paths.Homes); err != nil {
		t.Fatal(err)
	}
	distro, _ := FindDistribution("arch")
	if _, err := manager.Create(context.Background(), CreateOptions{
		Name: "unsafe-home", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome,
	}); err == nil || !strings.Contains(err.Error(), "symlinked managed homes") {
		t.Fatalf("Create() error = %v, want symlink rejection", err)
	}
	if len(container.created) != 0 {
		t.Fatalf("container was created despite unsafe home: %v", container.created)
	}
	if _, err := os.Stat(filepath.Join(external, "unsafe-home")); !os.IsNotExist(err) {
		t.Fatalf("external directory was modified: %v", err)
	}
}

func TestCreateRefusesSymlinkedManagedHomeAncestor(t *testing.T) {
	container := &fakeContainer{}
	hostRoot := t.TempDir()
	external := t.TempDir()
	paths := metadata.PathsFor(hostRoot)
	if err := os.Symlink(external, filepath.Join(hostRoot, ".local")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.Homes, 0700); err != nil {
		t.Fatal(err)
	}
	store := metadata.NewStore(paths)
	manager := NewManager(store, container, &fakeInspector{})
	distro, _ := FindDistribution("arch")
	if _, err := manager.Create(context.Background(), CreateOptions{
		Name: "unsafe-ancestor", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome,
	}); err == nil || !strings.Contains(err.Error(), "symlinked managed homes") {
		t.Fatalf("Create() error = %v, want ancestor symlink rejection", err)
	}
	if len(container.created) != 0 {
		t.Fatalf("container was created despite unsafe home ancestor: %v", container.created)
	}
}

func TestCreateCleansHomeWhenContainerCreationFails(t *testing.T) {
	container := &fakeContainer{createErr: errors.New("pull failed")}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	if _, err := manager.Create(context.Background(), CreateOptions{
		Name: "broken", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome,
	}); err == nil {
		t.Fatal("failed container creation returned nil")
	}
	if _, err := os.Stat(store.Home("broken")); !os.IsNotExist(err) {
		t.Fatalf("home was not cleaned: %v", err)
	}
}

func TestDisposableDeletesContainerAndHomeAfterExit(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	removed, err := manager.Create(context.Background(), CreateOptions{
		Name: "temporary", Distribution: distro, Persistence: Disposable, HomeMode: IsolatedHome, AutoEnter: true,
	})
	if err != nil || !removed {
		t.Fatalf("Create() = %v, %v", removed, err)
	}
	if len(container.entered) != 1 || len(container.deleted) != 1 {
		t.Fatalf("lifecycle calls: entered=%v deleted=%v", container.entered, container.deleted)
	}
	if _, err := os.Stat(store.Home("temporary")); !os.IsNotExist(err) {
		t.Fatalf("temporary home was not cleaned: %v", err)
	}
}

func TestListQueriesRuntimeOnceForAllSandboxes(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	for _, name := range []string{"box-a", "box-b", "box-c"} {
		if err := store.Save(Record{Name: name, Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome, CreatedAt: time.Now().In(time.Local)}); err != nil {
			t.Fatal(err)
		}
	}
	inspector := &fakeInspector{statuses: map[string]Status{"box-b": Running}}
	manager.Inspector = inspector
	entries, err := manager.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if inspector.batches != 1 || len(inspector.lastBatch) != 3 {
		t.Fatalf("inspector batches = %d for %v, want one batch of 3", inspector.batches, inspector.lastBatch)
	}
	want := map[string]Status{"box-a": Missing, "box-b": Running, "box-c": Missing}
	for _, entry := range entries {
		if entry.Status != want[entry.Name] {
			t.Errorf("entry %q status = %q, want %q", entry.Name, entry.Status, want[entry.Name])
		}
	}
}

func TestLookupReturnsMetadataWithoutRuntimeQuery(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	record := Record{Name: "quiet", Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome, CreatedAt: time.Now().In(time.Local)}
	if err := store.Save(record); err != nil {
		t.Fatal(err)
	}
	inspector := &fakeInspector{}
	manager.Inspector = inspector
	info, err := manager.Lookup(context.Background(), "quiet")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "quiet" || info.HomeMode != IsolatedHome {
		t.Fatalf("Lookup() = %+v, want metadata-only info", info)
	}
	if info.Status != "" {
		t.Fatalf("Lookup() status = %q, want empty without runtime query", info.Status)
	}
	if inspector.batches != 0 {
		t.Fatal("Lookup must not query the container runtime")
	}
}

func TestCreateRollsBackEverythingWhenPersistentEnterFails(t *testing.T) {
	container := &fakeContainer{enterErr: errors.New("pty failed")}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	_, err := manager.Create(context.Background(), CreateOptions{
		Name: "crashed", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome, AutoEnter: true,
	})
	if err == nil || !strings.Contains(err.Error(), "pty failed") {
		t.Fatalf("Create() error = %v, want enter failure", err)
	}
	if _, err := store.Get("crashed"); !errors.Is(err, metadata.ErrNotFound) {
		t.Fatalf("metadata was not rolled back: %v", err)
	}
	if _, err := os.Stat(store.Home("crashed")); !os.IsNotExist(err) {
		t.Fatalf("home was not rolled back: %v", err)
	}
	if len(container.deleted) != 1 || container.deleted[0] != "crashed" {
		t.Fatalf("container deletes = %v, want exactly one for crashed", container.deleted)
	}
}

// racingInspector plants a winner mid-create to simulate a concurrent create
// winning the reservation between this manager's checks and its reserve.
type racingInspector struct {
	fakeInspector
	during func()
}

func (r *racingInspector) Status(ctx context.Context, name string) (Status, error) {
	status, err := r.fakeInspector.Status(ctx, name)
	if r.during != nil {
		r.during()
		r.during = nil
	}
	return status, err
}

func TestLosingCreateCannotDeleteWinnersHome(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	distro, _ := FindDistribution("arch")
	winnerHome := store.Home("winner")
	inspector := &racingInspector{during: func() {
		record := Record{Name: "winner", Distribution: distro.ID, Image: distro.Image, Persistence: Persistent, HomeMode: IsolatedHome, CreatedAt: time.Now().In(time.Local)}
		if err := store.SaveExclusive(record); err != nil {
			t.Errorf("winner reservation failed: %v", err)
		}
		if err := os.MkdirAll(winnerHome, 0700); err != nil {
			t.Errorf("winner home creation failed: %v", err)
		}
		marker := filepath.Join(winnerHome, "data.txt")
		if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
			t.Errorf("winner marker write failed: %v", err)
		}
	}}
	manager.Inspector = inspector
	_, err := manager.Create(context.Background(), CreateOptions{
		Name: "winner", Distribution: distro, Persistence: Persistent, HomeMode: IsolatedHome,
	})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("losing Create() error = %v, want collision", err)
	}
	marker := filepath.Join(winnerHome, "data.txt")
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatalf("winner's home content was removed by the loser: %v", statErr)
	}
	if len(container.created) != 0 || len(container.deleted) != 0 {
		t.Fatalf("loser touched the runtime: created=%v deleted=%v", container.created, container.deleted)
	}
	if _, err := store.Get("winner"); err != nil {
		t.Fatalf("winner's metadata was damaged: %v", err)
	}
}

func TestSubIDConfiguredParsesLinuxColonFormat(t *testing.T) {
	path := t.TempDir() + "/subuid"
	if err := os.WriteFile(path, []byte("pablo:100000:65536\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if !SubIDConfigured(path, "pablo", "1000") {
		t.Fatal("colon-delimited subuid entry was not detected")
	}
}

func TestDeleteRefusesUnexpectedHomePath(t *testing.T) {
	container := &fakeContainer{}
	manager, store := newTestManager(t, container)
	if err := store.Save(Record{
		Name: "unsafe", Distribution: "arch", Image: "arch", Persistence: Persistent,
		HomeMode: IsolatedHome, HomePath: "/tmp/not-managed",
	}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Delete(context.Background(), "unsafe", DeleteOptions{DeleteHome: true}); err == nil {
		t.Fatal("unexpected home path was accepted")
	}
	if len(container.deleted) != 0 {
		t.Fatal("container was deleted despite unsafe home metadata")
	}
}
