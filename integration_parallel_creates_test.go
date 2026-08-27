//go:build integration

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// TestE2EIndependentParallelCreates proves that two concurrent creates with
// different names do not contaminate each other's homes or metadata.
func TestE2EIndependentParallelCreates(t *testing.T) {
	bin, env := prepareE2E(t)
	names := []string{uniqueName("parallel-a"), uniqueName("parallel-b")}
	for _, name := range names {
		cleanupSandbox(t, bin, env, name)
	}

	type result struct {
		output string
		err    error
	}
	results := make([]result, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			cmd := exec.Command(bin, "create", name, "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes")
			cmd.Env = env
			output, err := cmd.CombinedOutput()
			results[i] = result{string(output), err}
		}(i, name)
	}
	wg.Wait()

	for i, r := range results {
		if r.err != nil {
			t.Errorf("create %q failed: %v\n%s", names[i], r.err, r.output)
		}
	}

	homes := make([]string, len(names))
	for i, name := range names {
		homes[i] = filepath.Join(envHome(env), ".local", "share", "sandbox", "homes", name)
		assertExists(t, homes[i])
		marker := filepath.Join(homes[i], "owner.txt")
		if err := os.WriteFile(marker, []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}

	// Deleting the first sandbox must not touch the second one.
	runCLI(t, bin, env, "delete", names[0], "--yes")
	assertMissing(t, homes[0])
	if _, err := os.Stat(filepath.Join(homes[1], "owner.txt")); err != nil {
		t.Fatalf("second sandbox home damaged by deleting the first: %v", err)
	}

	runCLI(t, bin, env, "delete", names[1], "--yes")
	assertMissing(t, homes[1])
}
