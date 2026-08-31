//go:build integration

package main_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestE2EExecRunsWithoutTTYAndPropagatesExitCode(t *testing.T) {
	bin, env := prepareE2E(t)
	name := uniqueName("exec")
	cleanupSandbox(t, bin, env, name)
	runCLI(t, bin, env, "create", name, "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes")

	output := runCLI(t, bin, env, "exec", name, "--", "echo", "hello-exec")
	if !strings.Contains(output, "hello-exec") {
		t.Fatalf("exec echo output:\n%s", output)
	}

	// Exit codes from inside the sandbox must become the CLI exit code.
	failing := exec.Command(bin, "exec", name, "--", "sh", "-c", "echo boom >&2; exit 7")
	failing.Env = env
	combined, err := failing.CombinedOutput()
	if err == nil {
		t.Fatalf("exec with exit 7 unexpectedly succeeded:\n%s", combined)
	}
	if code := failing.ProcessState.ExitCode(); code != 7 {
		t.Fatalf("exec exit code = %d, want 7\n%s", code, combined)
	}
	if !strings.Contains(string(combined), "boom") {
		t.Fatalf("exec stderr was not preserved:\n%s", combined)
	}

	runCLI(t, bin, env, "delete", name, "--yes")
}

func TestE2EExecStartsStoppedSandbox(t *testing.T) {
	bin, env := prepareE2E(t)
	name := uniqueName("exec-stop")
	cleanupSandbox(t, bin, env, name)
	runCLI(t, bin, env, "create", name, "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes")
	runCLI(t, bin, env, "stop", name)

	output := runCLI(t, bin, env, "exec", name, "--", "echo", "restarted")
	if !strings.Contains(output, "restarted") {
		t.Fatalf("exec after stop output:\n%s", output)
	}
	var entries []sandboxEntry
	if err := json.Unmarshal([]byte(runCLI(t, bin, env, "list", "--json")), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Status != "running" {
		t.Fatalf("list after exec = %#v, want running", entries)
	}
	runCLI(t, bin, env, "delete", name, "--yes")
}

func TestE2EExecMissingSandboxFails(t *testing.T) {
	bin, env := prepareE2E(t)
	output := runCLIFailure(t, bin, env, "exec", uniqueName("exec-ghost"), "--", "true")
	if !strings.Contains(output, "does not exist") {
		t.Fatalf("exec on missing sandbox output:\n%s", output)
	}
}

func TestE2EExecRequiresCommandAfterDash(t *testing.T) {
	bin, env := prepareE2E(t)
	output := runCLIFailure(t, bin, env, "exec", uniqueName("exec-nocmd"), "--")
	if !strings.Contains(output, "command after --") {
		t.Fatalf("exec without command output:\n%s", output)
	}
}
