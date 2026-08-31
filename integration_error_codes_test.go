//go:build integration

package main_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestE2EErrorCodesMachineJSON(t *testing.T) {
	bin, env := prepareE2E(t)
	name := uniqueName("error-codes")
	cleanupSandbox(t, bin, env, name)

	// A missing sandbox must yield a machine JSON error on stderr with
	// E_NOT_FOUND and exit code 1.
	cmd := exec.Command(bin, "--error-format", "json", "exec", name, "--", "echo", "hi")
	cmd.Env = setEnv(env, "NO_COLOR", "1")
	cmd.Env = setEnv(cmd.Env, "TERM", "dumb")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("exec on missing sandbox unexpectedly succeeded:\n%s", output)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("exit error type = %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}
	if !strings.Contains(string(output), `"code":"E_NOT_FOUND"`) {
		t.Errorf("output does not contain E_NOT_FOUND code:\n%s", output)
	}
	if !strings.Contains(string(output), `"error"`) {
		t.Errorf("output does not contain machine error object:\n%s", output)
	}
}
