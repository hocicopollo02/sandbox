package distrobox

import (
	"context"
	"errors"
	"testing"
)

type recordingRunner struct {
	calls [][]string
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return nil, nil
}
func (r *recordingRunner) Attach(_ context.Context, name string, args ...string) error {
	r.calls = append(r.calls, append([]string{name}, args...))
	return nil
}
func (r *recordingRunner) LookPath(file string) (string, error) { return "/usr/bin/" + file, nil }

func TestCreateBuildsSafeDistroboxArguments(t *testing.T) {
	runner := &recordingRunner{}
	client := New(runner)
	if err := client.Create(context.Background(), "gentle-ai", "archlinux:latest", "/tmp/home"); err != nil {
		t.Fatal(err)
	}
	want := []string{"distrobox", "create", "--yes", "--name", "gentle-ai", "--image", "archlinux:latest", "--home", "/tmp/home"}
	if len(runner.calls) != 1 || !same(runner.calls[0], want) {
		t.Fatalf("call = %#v, want %#v", runner.calls, want)
	}
}

type fallbackRunner struct {
	calls [][]string
}

func (r *fallbackRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if name == "distrobox" {
		return []byte("container has active exec sessions"), errors.New("exit status 1")
	}
	return nil, nil
}
func (r *fallbackRunner) Attach(context.Context, string, ...string) error { return nil }
func (r *fallbackRunner) LookPath(file string) (string, error)            { return "/usr/bin/" + file, nil }

func TestDeleteFallsBackToPodmanForActiveExecSession(t *testing.T) {
	runner := &fallbackRunner{}
	client := New(runner)
	if err := client.Delete(context.Background(), "gentle-ai"); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 2 || !same(runner.calls[1], []string{"podman", "rm", "--force", "gentle-ai"}) {
		t.Fatalf("calls = %#v", runner.calls)
	}
}

func TestStopFallsBackToPodmanForActiveExecSession(t *testing.T) {
	runner := &fallbackRunner{}
	client := New(runner)
	if err := client.Stop(context.Background(), "gentle-ai"); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 2 || !same(runner.calls[1], []string{"podman", "stop", "gentle-ai"}) {
		t.Fatalf("calls = %#v", runner.calls)
	}
}

func TestStopSkipsInteractiveConfirmation(t *testing.T) {
	runner := &recordingRunner{}
	client := New(runner)
	if err := client.Stop(context.Background(), "gentle-ai"); err != nil {
		t.Fatal(err)
	}
	want := []string{"distrobox", "stop", "--yes", "gentle-ai"}
	if len(runner.calls) != 1 || !same(runner.calls[0], want) {
		t.Fatalf("call = %#v, want %#v", runner.calls, want)
	}
}

func same(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
