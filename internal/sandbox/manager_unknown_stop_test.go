package sandbox

import (
	"context"
	"testing"
)

type stopRecordingContainer struct {
	fakeContainer
	stopCalls []string
}

func (f *stopRecordingContainer) Stop(_ context.Context, name string) error {
	f.stopCalls = append(f.stopCalls, name)
	return nil
}

func TestStopUnknownSandboxFailsWithoutTouchingRuntime(t *testing.T) {
	container := &stopRecordingContainer{}
	manager, store := newTestManager(t, &container.fakeContainer)
	manager.Container = container
	distro, _ := FindDistribution("arch")
	if err := store.Save(Record{
		Name: "box", Distribution: distro.ID, Image: distro.Image,
		Persistence: Persistent, HomeMode: IsolatedHome,
	}); err != nil {
		t.Fatal(err)
	}
	manager.Inspector = &fakeInspector{status: Unknown}

	if err := manager.Stop(context.Background(), "box"); err == nil {
		t.Fatal("Stop() succeeded for an unknown sandbox")
	}
	if len(container.stopCalls) != 0 {
		t.Fatalf("runtime was touched for an unknown sandbox: %#v", container.stopCalls)
	}
}
