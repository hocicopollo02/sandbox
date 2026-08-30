package podman

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// execRecordingRunner captures calls, reports a stopped sandbox on inspect and
// optionally fails streamed execution with the given error.
type execRecordingRunner struct {
	recordingRunner
	streamErr error
}

func (r *execRecordingRunner) RunStream(_ context.Context, name string, args ...string) error {
	r.calls = append(r.calls, append([]string{name}, args...))
	return r.streamErr
}

func TestExecStartsStoppedSandboxWithoutTTY(t *testing.T) {
	runner := &execRecordingRunner{}
	client := New(runner)
	if err := client.Exec(context.Background(), "gentle-ai", []string{"echo", "hello"}); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"podman", "inspect", "--format", "{{.State.Status}}", "gentle-ai"},
		{"podman", "start", "gentle-ai"},
		{"podman", "exec", "gentle-ai", "echo", "hello"},
	}
	if !sameCalls(runner.calls, want) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, want)
	}
	for _, call := range runner.calls {
		if len(call) > 0 && call[0] == "podman" && call[1] == "exec" {
			for _, arg := range call[2:] {
				if arg == "--tty" || arg == "--interactive" || arg == "-t" || arg == "-i" {
					t.Fatalf("exec call %#v must not attach a TTY or stdin", call)
				}
			}
		}
	}
}

func TestExecRunningSandboxSkipsStart(t *testing.T) {
	running := &statusRunner{status: "running"}
	client := New(running)
	if err := client.Exec(context.Background(), "gentle-ai", []string{"true"}); err != nil {
		t.Fatal(err)
	}
	for _, call := range running.calls {
		if len(call) >= 2 && call[0] == "podman" && call[1] == "start" {
			t.Fatalf("running sandbox must not be started: %#v", call)
		}
	}
}

type statusRunner struct {
	calls   [][]string
	status  string
	streams [][]string
}

func (r *statusRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if len(args) > 0 && args[0] == "inspect" {
		return []byte(r.status + "\n"), nil
	}
	return nil, nil
}

func (r *statusRunner) RunStream(_ context.Context, name string, args ...string) error {
	r.streams = append(r.streams, append([]string{name}, args...))
	return nil
}

func (r *statusRunner) Attach(context.Context, string, ...string) error { return nil }
func (r *statusRunner) LookPath(string) (string, error)                 { return "/bin/podman", nil }

func TestExecArgsPassUnmodified(t *testing.T) {
	runner := &execRecordingRunner{}
	client := New(runner)
	command := []string{"bash", "-lc", "echo $HOME && ls -la 'some dir'"}
	if err := client.Exec(context.Background(), "box", command); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("calls = %#v", runner.calls)
	}
	got := runner.calls[2]
	want := append([]string{"podman", "exec", "box"}, command...)
	if !sameCalls([][]string{got}, [][]string{want}) {
		t.Fatalf("exec call = %#v, want %#v", got, want)
	}
}

func TestExecMissingContainerError(t *testing.T) {
	runner := &missingRunner{}
	client := New(runner)
	err := client.Exec(context.Background(), "ghost", []string{"true"})
	if err == nil || !strings.Contains(err.Error(), "does not exist in the container runtime") {
		t.Fatalf("Exec() = %v, want missing-container error", err)
	}
}

type missingRunner struct{}

func (r *missingRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	return []byte("Error: no such container ghost"), errMissingContainer
}

var errMissingContainer = errors.New("inspect failed")

func (r *missingRunner) RunStream(context.Context, string, ...string) error { return nil }
func (r *missingRunner) Attach(context.Context, string, ...string) error    { return nil }
func (r *missingRunner) LookPath(string) (string, error)                    { return "/bin/podman", nil }
