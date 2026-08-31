//go:build integration

package main_test

import (
	"encoding/json"
	"strings"
	"testing"
)

type doctorJSONResult struct {
	OK     bool `json:"ok"`
	Checks []struct {
		Name string `json:"name"`
		OK   bool   `json:"ok"`
	} `json:"checks"`
}

func TestE2EDoctorJSON(t *testing.T) {
	bin, env := prepareE2E(t)
	output := runCLI(t, bin, env, "doctor", "--json")
	var result doctorJSONResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("doctor --json is not valid JSON: %v\n%s", err, output)
	}
	if !result.OK {
		t.Fatalf("doctor --json ok=false on healthy host:\n%s", output)
	}
	if len(result.Checks) != 4 {
		t.Fatalf("checks = %d, want 4:\n%s", len(result.Checks), output)
	}
	for _, check := range result.Checks {
		if !check.OK {
			t.Fatalf("check %q not ok on healthy host:\n%s", check.Name, output)
		}
	}

	human := runCLI(t, bin, env, "doctor")
	if !strings.Contains(human, "Everything looks good.") {
		t.Fatalf("human doctor output changed:\n%s", human)
	}
}

func TestE2EDoctorJSONFailsWithoutRuntime(t *testing.T) {
	bin, env := prepareE2E(t)
	missingPath := t.TempDir()
	env = setEnv(env, "PATH", missingPath)
	output := runCLIFailure(t, bin, env, "doctor", "--json")
	var result doctorJSONResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("doctor --json on failure is not valid JSON: %v\n%s", err, output)
	}
	if result.OK {
		t.Fatalf("ok = true without podman:\n%s", output)
	}
	found := false
	for _, check := range result.Checks {
		if check.Name == "Podman installed" {
			found = true
			if check.OK {
				t.Fatalf("Podman installed ok=true without podman:\n%s", output)
			}
		}
	}
	if !found {
		t.Fatalf("Podman installed check missing from output:\n%s", output)
	}
}
