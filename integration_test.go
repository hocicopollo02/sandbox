//go:build integration

package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/creack/pty"
)

var e2eSequence uint64

type sandboxEntry struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type interactiveProcess struct {
	cmd    *exec.Cmd
	pty    *os.File
	mu     sync.Mutex
	output bytes.Buffer
	done   chan error
}

func TestE2EDoctor(t *testing.T) {
	bin, env := prepareE2E(t)
	output := runCLI(t, bin, env, "doctor")
	if !strings.Contains(output, "Everything looks good.") {
		t.Fatalf("doctor output did not report success:\n%s", output)
	}
}

func TestE2EPersistentLifecycle(t *testing.T) {
	bin, env := prepareE2E(t)
	name := uniqueName("persistent")
	home := filepath.Join(envHome(env), ".local", "share", "sandbox", "homes", name)
	metadata := filepath.Join(envHome(env), ".local", "share", "sandbox", "sandboxes", name+".json")
	cleanupSandbox(t, bin, env, name)

	output := runCLI(t, bin, env, "create", name, "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes")
	if !strings.Contains(output, "Sandbox ready") {
		t.Fatalf("create output:\n%s", output)
	}
	assertExists(t, home)
	assertExists(t, metadata)

	var entries []sandboxEntry
	if err := json.Unmarshal([]byte(runCLI(t, bin, env, "list", "--json")), &entries); err != nil {
		t.Fatalf("list --json: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != name || entries[0].Status != "stopped" {
		t.Fatalf("list entries = %#v", entries)
	}
	info := runCLI(t, bin, env, "info", name)
	for _, expected := range []string{"Arch Linux", "persistent", "isolated", name} {
		if !strings.Contains(info, expected) {
			t.Errorf("info output does not contain %q:\n%s", expected, info)
		}
	}

	entered := startInteractive(t, bin, env, "enter", name)
	waitForShell(t, entered)
	entered.write("echo READY\n")
	entered.waitFor(t, "READY", 30*time.Second, nil)
	if output := runCLI(t, bin, env, "stop", name); !strings.Contains(output, "Sandbox stopped") {
		t.Fatalf("stop output:\n%s", output)
	}
	entered.wait(t, 30*time.Second)

	var afterStop []sandboxEntry
	if err := json.Unmarshal([]byte(runCLI(t, bin, env, "list", "--json")), &afterStop); err != nil {
		t.Fatalf("list after stop: %v", err)
	}
	if len(afterStop) != 1 || afterStop[0].Status != "stopped" {
		t.Fatalf("list after stop = %#v", afterStop)
	}

	runCLI(t, bin, env, "delete", name, "--yes")
	assertMissing(t, home)
	assertMissing(t, metadata)
	assertContainerMissing(t, env, name)
}

func TestE2EDisposableLifecycle(t *testing.T) {
	bin, env := prepareE2E(t)
	name := uniqueName("disposable")
	home := filepath.Join(envHome(env), ".local", "share", "sandbox", "homes", name)
	metadata := filepath.Join(envHome(env), ".local", "share", "sandbox", "sandboxes", name+".json")
	cleanupSandbox(t, bin, env, name)

	process := startInteractive(t, bin, env, "create", name, "--distro", "arch", "--disposable", "--isolated-home", "--yes")
	waitForShell(t, process)
	process.write("echo READY\n")
	process.waitFor(t, "READY", 30*time.Second, nil)
	process.write("exit\n")
	process.wait(t, 30*time.Second)
	if !strings.Contains(process.outputString(), "Sandbox removed") {
		t.Fatalf("disposable output:\n%s", process.outputString())
	}
	assertMissing(t, home)
	assertMissing(t, metadata)
	assertContainerMissing(t, env, name)
}

func TestE2EDisposableInterruptCleansUp(t *testing.T) {
	bin, env := prepareE2E(t)
	name := uniqueName("interrupt")
	home := filepath.Join(envHome(env), ".local", "share", "sandbox", "homes", name)
	cleanupSandbox(t, bin, env, name)

	process := startInteractive(t, bin, env, "create", name, "--distro", "arch", "--disposable", "--isolated-home", "--yes")
	waitForShell(t, process)
	if err := process.cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("send interrupt: %v", err)
	}
	process.wait(t, 30*time.Second)
	assertMissing(t, home)
	assertContainerMissing(t, env, name)
}

func prepareE2E(t *testing.T) (string, []string) {
	t.Helper()
	if os.Getenv("SANDBOX_E2E") != "1" {
		t.Skip("set SANDBOX_E2E=1 to run Podman/Distrobox integration tests")
	}
	if os.Geteuid() == 0 {
		t.Skip("integration tests require a non-root user")
	}
	for _, command := range []string{"podman", "distrobox"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("%s is not installed", command)
		}
	}
	check := exec.Command("podman", "info", "--format", "{{.Host.Security.Rootless}}")
	if output, err := check.Output(); err != nil || strings.TrimSpace(string(output)) != "true" {
		t.Skip("Podman is not available in rootless mode")
	}

	_, source, _, _ := runtime.Caller(0)
	root := filepath.Dir(source)
	bin := filepath.Join(t.TempDir(), "sandbox")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build sandbox binary: %v\n%s", err, output)
	}
	home := t.TempDir()
	env := replaceEnv(home, "1")
	return bin, env
}

func startInteractive(t *testing.T, bin string, env []string, args ...string) *interactiveProcess {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	terminal, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("start interactive command: %v", err)
	}
	process := &interactiveProcess{cmd: cmd, pty: terminal, done: make(chan error, 1)}
	go func() {
		_, _ = io.Copy(lockedBuffer{process}, terminal)
		process.done <- cmd.Wait()
	}()
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = terminal.Close()
	})
	return process
}

type lockedBuffer struct{ process *interactiveProcess }

func (b lockedBuffer) Write(data []byte) (int, error) {
	b.process.mu.Lock()
	defer b.process.mu.Unlock()
	return b.process.output.Write(data)
}

func (p *interactiveProcess) write(input string) {
	_, _ = p.pty.Write([]byte(input))
}

func waitForShell(t *testing.T, process *interactiveProcess) {
	t.Helper()
	process.waitFor(t, "]$ ", 60*time.Second, nil)
}

func (p *interactiveProcess) waitFor(t *testing.T, needle string, timeout time.Duration, trigger func()) {
	t.Helper()
	if trigger != nil {
		trigger()
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(p.outputString(), needle) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("interactive process did not output %q:\n%s", needle, p.outputString())
}

func (p *interactiveProcess) wait(t *testing.T, timeout time.Duration) {
	t.Helper()
	select {
	case <-p.done:
	case <-time.After(timeout):
		t.Fatalf("interactive process did not exit:\n%s", p.outputString())
	}
}

func (p *interactiveProcess) outputString() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.output.String()
}

func runCLI(t *testing.T, bin string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sandbox %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func cleanupSandbox(t *testing.T, bin string, env []string, name string) {
	t.Helper()
	remove := func() {
		cmd := exec.Command(bin, "delete", name, "--yes")
		cmd.Env = env
		_ = cmd.Run()
		cmd = exec.Command("podman", "rm", "--force", name)
		cmd.Env = env
		_ = cmd.Run()
		_ = os.RemoveAll(filepath.Join(envHome(env), ".local", "share", "sandbox", "homes", name))
	}
	remove()
	t.Cleanup(remove)
}

func assertContainerMissing(t *testing.T, env []string, name string) {
	t.Helper()
	cmd := exec.Command("podman", "inspect", name)
	cmd.Env = env
	if err := cmd.Run(); err == nil {
		t.Fatalf("container %q still exists", name)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, got %v", path, err)
	}
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("e2e-%s-%d-%d", prefix, os.Getpid(), atomic.AddUint64(&e2eSequence, 1))
}

func replaceEnv(home, noColor string) []string {
	overrides := map[string]string{"HOME": home, "NO_COLOR": noColor}
	env := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, overridden := overrides[key]; !overridden {
			env = append(env, entry)
		}
	}
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}

func envHome(env []string) string {
	for _, entry := range env {
		if strings.HasPrefix(entry, "HOME=") {
			return strings.TrimPrefix(entry, "HOME=")
		}
	}
	panic("HOME not found in test environment")
}
