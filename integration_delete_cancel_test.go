//go:build integration

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestE2EDeleteCancelKeepsEverything verifies that answering "No" to the
// destructive delete confirmation leaves the container, isolated home and
// metadata untouched, and that no podman removal is issued.
func TestE2EDeleteCancelKeepsEverything(t *testing.T) {
	bin, env := prepareE2E(t)
	name := uniqueName("delete-cancel")
	home := filepath.Join(envHome(env), ".local", "share", "sandbox", "homes", name)
	metadata := filepath.Join(envHome(env), ".local", "share", "sandbox", "sandboxes", name+".json")
	cleanupSandbox(t, bin, env, name)

	runCLI(t, bin, env, "create", name, "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes")
	marker := filepath.Join(home, "precious.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}

	env = setEnv(env, "TERM", "dumb")
	process := startInteractive(t, bin, env, "delete", name)
	process.waitFor(t, "Delete sandbox", 20*time.Second, nil)
	process.write("n\r")
	process.wait(t, 20*time.Second)

	if !strings.Contains(process.outputString(), "deletion cancelled") {
		t.Fatalf("cancel output does not report cancellation:\n%s", process.outputString())
	}

	// The container must still exist: no podman rm was issued.
	cmd := exec.Command("podman", "inspect", name)
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		t.Fatalf("container %q was removed despite cancel: %v", name, err)
	}
	assertExists(t, marker)
	assertExists(t, metadata)
}
