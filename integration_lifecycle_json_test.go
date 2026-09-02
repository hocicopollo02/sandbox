//go:build integration

package main_test

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

type lifecycleJSONResult struct {
	Name         string  `json:"name"`
	Result       string  `json:"result"`
	RetainedHome *string `json:"retained_home"`
}

func TestE2ELifecycleSuccessJSON(t *testing.T) {
	bin, env := prepareE2E(t)
	name := uniqueName("lifecycle-json")
	home := envHome(env) + "/.local/share/sandbox/homes/" + name
	metadata := envHome(env) + "/.local/share/sandbox/sandboxes/" + name + ".json"
	cleanupSandbox(t, bin, env, name)

	created := decodeLifecycleJSON(t, runCLI(t, bin, env, "create", name, "--distro", "arch", "--persistent", "--isolated-home", "--no-enter", "--yes", "--json"))
	if created.Name != name || created.Result != "created" || created.RetainedHome != nil {
		t.Fatalf("create JSON = %#v", created)
	}

	stopped := decodeLifecycleJSON(t, runCLI(t, bin, env, "stop", name, "--json"))
	if stopped.Name != name || stopped.Result != "stopped" || stopped.RetainedHome != nil {
		t.Fatalf("stop JSON = %#v", stopped)
	}

	deleted := decodeLifecycleJSON(t, runCLI(t, bin, env, "delete", name, "--yes", "--json"))
	if deleted.Name != name || deleted.Result != "deleted" || deleted.RetainedHome != nil {
		t.Fatalf("delete JSON = %#v", deleted)
	}
	assertMissing(t, home)
	assertMissing(t, metadata)
	assertContainerMissing(t, env, name)
}

func decodeLifecycleJSON(t *testing.T, output string) lifecycleJSONResult {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(output))
	var result lifecycleJSONResult
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("lifecycle output is not JSON: %v\n%s", err, output)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("lifecycle output contains more than one JSON value: %q", output)
	}
	return result
}
