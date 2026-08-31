//go:build integration

package main_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestE2EInfoJSON(t *testing.T) {
	bin, env := prepareE2E(t)
	name := uniqueName("info-json")
	cleanupSandbox(t, bin, env, name)

	runCLI(t, bin, env, "create", name, "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes")

	output := runCLI(t, bin, env, "info", name, "--json")
	var info struct {
		Name     string `json:"name"`
		Status   string `json:"status"`
		Image    string `json:"image"`
		HomeMode string `json:"home_mode"`
	}
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		t.Fatalf("info --json output is not valid JSON:\n%s\n%v", output, err)
	}
	if info.Name != name {
		t.Fatalf("info.Name = %q, want %q", info.Name, name)
	}
	if info.Status != "running" {
		t.Fatalf("info.Status = %q, want running", info.Status)
	}
	if info.Image == "" || info.HomeMode != "isolated" {
		t.Fatalf("info image/home = %q/%q, want image set and isolated home", info.Image, info.HomeMode)
	}
	if strings.Contains(output, "\"Status\"") {
		t.Fatalf("info --json leaked untagged Status key:\n%s", output)
	}
	runCLI(t, bin, env, "delete", name, "--yes")
}
