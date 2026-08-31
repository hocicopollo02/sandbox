//go:build integration

package main_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestE2EIdempotentCreateDeleteFlags(t *testing.T) {
	bin, env := prepareE2E(t)
	name := uniqueName("idempotent")
	cleanupSandbox(t, bin, env, name)

	runCLI(t, bin, env, "create", name, "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes")

	// Second create with --if-not-exists must succeed without creating a second container.
	output := runCLI(t, bin, env, "create", name, "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes", "--if-not-exists")
	if !strings.Contains(output, "Sandbox ready") {
		t.Fatalf("idempotent create output:\n%s", output)
	}
	cmd := exec.Command("podman", "ps", "-a", "--format", "{{.Names}}")
	cmd.Env = env
	names, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(names), name); count != 1 {
		t.Fatalf("container count for %q = %d, want exactly 1", name, count)
	}

	runCLI(t, bin, env, "delete", name, "--yes")

	// Deleting a missing sandbox with --if-exists must succeed with a no-op message.
	output = runCLI(t, bin, env, "delete", name, "--yes", "--if-exists")
	if !strings.Contains(output, "nothing to do") {
		t.Fatalf("idempotent delete output:\n%s", output)
	}

	// Deleting a never-existing sandbox with --if-exists must also succeed.
	ghost := uniqueName("never")
	runCLI(t, bin, env, "delete", ghost, "--yes", "--if-exists")

	// list --json must not contain the deleted sandbox.
	entries := runCLI(t, bin, env, "list", "--json")
	var parsed []sandboxEntry
	if err := json.Unmarshal([]byte(entries), &parsed); err != nil {
		t.Fatalf("list --json: %v", err)
	}
	for _, entry := range parsed {
		if entry.Name == name || entry.Name == ghost {
			t.Fatalf("deleted sandbox still listed: %#v", entry)
		}
	}
}
