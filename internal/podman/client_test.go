package podman

import (
	"context"
	"errors"
	"testing"

	"github.com/pablo/sandbox/internal/sandbox"
)

type fakeRunner struct {
	output []byte
	err    error
}

func (f fakeRunner) Run(context.Context, string, ...string) ([]byte, error) { return f.output, f.err }
func (f fakeRunner) Attach(context.Context, string, ...string) error        { return f.err }
func (f fakeRunner) LookPath(string) (string, error)                        { return "/bin/tool", nil }

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
