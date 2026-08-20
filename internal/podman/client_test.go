package podman

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hocicopollo02/sandbox/internal/sandbox"
)

type fakeRunner struct {
	output []byte
	err    error
}

func (f fakeRunner) Run(context.Context, string, ...string) ([]byte, error) { return f.output, f.err }
func (f fakeRunner) Attach(context.Context, string, ...string) error        { return f.err }
func (f fakeRunner) LookPath(string) (string, error)                        { return "/bin/tool", nil }

type recordingRunner struct {
	calls [][]string
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if len(args) > 0 && args[0] == "inspect" {
		return []byte("exited\n"), nil
	}
	return nil, nil
}

func (r *recordingRunner) Attach(_ context.Context, name string, args ...string) error {
	r.calls = append(r.calls, append([]string{name}, args...))
	return nil
}

func (r *recordingRunner) LookPath(string) (string, error) { return "/bin/podman", nil }

func TestCreateUsesOnlyManagedHomeMount(t *testing.T) {
	runner := &recordingRunner{}
	client := New(runner)
	if err := client.Create(context.Background(), "gentle-ai", "archlinux:latest", "/tmp/managed-home"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"podman", "create", "--pull=missing", "--name", "gentle-ai", "--hostname", "gentle-ai", "--workdir", "/home/sandbox", "--env", "HOME=/home/sandbox", "--volume", "/tmp/managed-home:/home/sandbox", "archlinux:latest", "sleep", "infinity"},
		{"podman", "start", "gentle-ai"},
	}
	if !sameCalls(runner.calls, want) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, want)
	}
}

type failingStartRunner struct {
	calls [][]string
}

func (r *failingStartRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	switch args[0] {
	case "start":
		return []byte("start failed"), errors.New("start exit status 1")
	case "rm":
		return []byte("cleanup failed"), errors.New("cleanup exit status 1")
	default:
		return nil, nil
	}
}

func (r *failingStartRunner) Attach(context.Context, string, ...string) error { return nil }
func (r *failingStartRunner) LookPath(string) (string, error)                 { return "/bin/podman", nil }

func TestCreateReportsCleanupFailureAfterStartFailure(t *testing.T) {
	runner := &failingStartRunner{}
	client := New(runner)
	err := client.Create(context.Background(), "broken", "archlinux:latest", "")
	if err == nil || !strings.Contains(err.Error(), "cleanup failed") {
		t.Fatalf("Create() error = %v, want cleanup failure", err)
	}
	want := [][]string{
		{"podman", "create", "--pull=missing", "--name", "broken", "--hostname", "broken", "--workdir", "/home/sandbox", "--env", "HOME=/home/sandbox", "archlinux:latest", "sleep", "infinity"},
		{"podman", "start", "broken"},
		{"podman", "rm", "--force", "broken"},
	}
	if !sameCalls(runner.calls, want) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, want)
	}
}

func TestEnterStartsStoppedContainerAndUsesExec(t *testing.T) {
	runner := &recordingRunner{}
	client := New(runner)
	if err := client.Enter(context.Background(), "gentle-ai"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"podman", "inspect", "--format", "{{.State.Status}}", "gentle-ai"},
		{"podman", "start", "gentle-ai"},
		{"podman", "exec", "--interactive", "--tty", "gentle-ai", "/bin/bash"},
	}
	if !sameCalls(runner.calls, want) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, want)
	}
}

func sameCalls(left, right [][]string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if len(left[i]) != len(right[i]) {
			return false
		}
		for j := range left[i] {
			if left[i][j] != right[i][j] {
				return false
			}
		}
	}
	return true
}

func TestStatusMapsPodmanOutput(t *testing.T) {
	for _, test := range []struct {
		output string
		want   sandbox.Status
	}{
		{"running\n", sandbox.Running},
		{"exited\n", sandbox.Stopped},
		{"created\n", sandbox.Stopped},
	} {
		client := New(fakeRunner{output: []byte(test.output)})
		got, err := client.Status(context.Background(), "box")
		if err != nil || got != test.want {
			t.Errorf("Status(%q) = %q, %v; want %q", test.output, got, err, test.want)
		}
	}
}

func TestStatusTreatsMissingObjectAsMissing(t *testing.T) {
	client := New(fakeRunner{output: []byte(`Error: no such object: "box"`), err: errors.New("exit status 125")})
	got, err := client.Status(context.Background(), "box")
	if err != nil || got != sandbox.Missing {
		t.Fatalf("Status() = %q, %v; want missing", got, err)
	}
}
