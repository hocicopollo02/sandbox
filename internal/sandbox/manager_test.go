package sandbox

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/pablo/sandbox/internal/metadata"
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
	status Status
}

func (f *fakeInspector) Available() error { return nil }
func (f *fakeInspector) Status(_ context.Context, name string) (Status, error) {
	if f.status == "" {
		return Missing, nil
	}
	return f.status, nil
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
