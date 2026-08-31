//go:build integration

package main_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os/exec"
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
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatalf("exec on missing sandbox unexpectedly succeeded:\n%s", stderr.String())
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("exit error type = %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decoder := json.NewDecoder(&stderr)
	if err := decoder.Decode(&payload); err != nil {
		t.Fatalf("stderr is not one JSON error object: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Errorf("stderr contains more than one JSON document: %v", err)
	}
	if payload.Error.Code != "E_NOT_FOUND" {
		t.Errorf("error code = %q, want E_NOT_FOUND", payload.Error.Code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout is not empty: %s", stdout.String())
	}
}
