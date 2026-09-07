package sandbox

import (
	"context"
	"testing"
	"time"
)

type timeoutAwareContainer struct {
	fakeContainer
}

func (c *timeoutAwareContainer) Enter(ctx context.Context, name string) error {
	c.entered = append(c.entered, name)
	select {
	case <-time.After(5 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestCreateAutoEnterOutlivesSetupTimeoutAndCleansDisposable(t *testing.T) {
	container := &timeoutAwareContainer{}
	manager, store := newTestManager(t, &container.fakeContainer)
	manager.Container = container
	distro, _ := FindDistribution("arch")

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	time.Sleep(3 * time.Millisecond)

	result, err := manager.CreateWithResult(ctx, CreateOptions{
		Name:         "temporary",
		Distribution: distro,
		Persistence:  Disposable,
		HomeMode:     IsolatedHome,
		AutoEnter:    true,
	})
	if err != nil {
		t.Fatalf("CreateWithResult() error = %v, want auto-enter to survive setup timeout", err)
	}
	if result != CreateResultRemoved {
		t.Fatalf("CreateWithResult() result = %q, want removed", result)
	}
	if len(container.deleted) != 1 || container.deleted[0] != "temporary" {
		t.Fatalf("delete calls = %v, want disposable cleanup", container.deleted)
	}
	if _, err := store.Get("temporary"); err == nil {
		t.Fatal("disposable metadata still exists after auto-enter")
	}
}
