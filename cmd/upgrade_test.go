package cmd

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type upgradeFakeRunner struct {
	latest string
	calls  [][]string
	envs   []map[string]string
}

func (r *upgradeFakeRunner) RunWithEnv(ctx context.Context, env map[string]string, name string, args ...string) ([]byte, error) {
	r.envs = append(r.envs, env)
	return r.Run(ctx, name, args...)
}

func (r *upgradeFakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if name != "go" || len(args) == 0 {
		return nil, errors.New("unexpected command")
	}
	switch args[0] {
	case "list":
		return []byte(r.latest + "\n"), nil
	case "env":
		return nil, nil
	case "install":
		return nil, nil
	default:
		return nil, errors.New("unexpected go subcommand")
	}
}

func (r *upgradeFakeRunner) RunStream(context.Context, string, ...string) error { return nil }
func (r *upgradeFakeRunner) Attach(context.Context, string, ...string) error    { return nil }
func (r *upgradeFakeRunner) LookPath(string) (string, error)                    { return "/usr/bin/go", nil }

func newUpgradeTestApp(runner *upgradeFakeRunner) (*app, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return &app{
		runner: runner,
		out:    out,
		errOut: &bytes.Buffer{},
		executablePath: func() (string, error) {
			return "/home/user/go/bin/sandbox", nil
		},
	}, out
}

func TestUpgradeJSONReportsUnchangedWithoutInstalling(t *testing.T) {
	runner := &upgradeFakeRunner{latest: "v1.2.0"}
	appState, out := newUpgradeTestApp(runner)
	cmd := newUpgradeCommand(appState)
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := `{"name":"sandbox","current_version":"1.2.0","latest_version":"1.2.0","result":"unchanged"}
`
	if out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
	for _, call := range runner.calls {
		if len(call) > 1 && call[1] == "install" {
			t.Fatal("upgrade installed an already-current version")
		}
	}
}

func TestUpgradeJSONInstallsResolvedLatestVersion(t *testing.T) {
	runner := &upgradeFakeRunner{latest: "v1.3.0"}
	appState, out := newUpgradeTestApp(runner)
	cmd := newUpgradeCommand(appState)
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := `{"name":"sandbox","current_version":"1.2.0","latest_version":"1.3.0","result":"upgraded"}
`
	if out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
	wantCalls := [][]string{
		{"go", "list", "-m", "-f", "{{.Version}}", "github.com/hocicopollo02/sandbox@latest"},
		{"go", "install", "github.com/hocicopollo02/sandbox@v1.3.0"},
	}
	if !sameUpgradeCalls(runner.calls, wantCalls) {
		t.Fatalf("go calls = %#v, want %#v", runner.calls, wantCalls)
	}
	if len(runner.envs) != 1 || runner.envs[0]["GOBIN"] != "/home/user/go/bin" {
		t.Fatalf("install environment = %#v, want GOBIN=/home/user/go/bin", runner.envs)
	}
}

func sameUpgradeCalls(got, want [][]string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if len(got[i]) != len(want[i]) {
			return false
		}
		for j := range got[i] {
			if got[i][j] != want[i][j] {
				return false
			}
		}
	}
	return true
}

func TestUpgradeInstallsIntoExecutableDirectory(t *testing.T) {
	runner := &upgradeFakeRunner{latest: "v1.3.0"}
	appState, out := newUpgradeTestApp(runner)
	appState.executablePath = func() (string, error) {
		return "/usr/local/bin/sandbox", nil
	}
	cmd := newUpgradeCommand(appState)
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 {
		t.Fatal("stdout is empty, want upgrade result")
	}
	if len(runner.envs) != 1 || runner.envs[0]["GOBIN"] != "/usr/local/bin" {
		t.Fatalf("install environment = %#v, want GOBIN=/usr/local/bin", runner.envs)
	}
}

func TestUpgradeHumanOutputReportsCurrentVersion(t *testing.T) {
	runner := &upgradeFakeRunner{latest: "v1.2.0"}
	appState, out := newUpgradeTestApp(runner)
	cmd := newUpgradeCommand(appState)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "sandbox is already up to date (1.2.0)\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
