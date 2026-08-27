//go:build integration

package main_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2EDistroMatrix runs the persistent lifecycle for every supported
// distribution. It is opt-in and serial because it downloads several images.
func TestE2EDistroMatrix(t *testing.T) {
	if os.Getenv("SANDBOX_E2E_MATRIX") != "1" {
		t.Skip("set SANDBOX_E2E_MATRIX=1 to run the distribution matrix")
	}
	bin, env := prepareE2E(t)

	for _, distro := range []string{"ubuntu", "fedora", "debian"} {
		t.Run(distro, func(t *testing.T) {
			name := uniqueName("matrix-" + distro)
			home := filepath.Join(envHome(env), ".local", "share", "sandbox", "homes", name)
			metadata := filepath.Join(envHome(env), ".local", "share", "sandbox", "sandboxes", name+".json")
			cleanupSandbox(t, bin, env, name)

			output := runCLI(t, bin, env, "create", name, "--distro", distro, "--persistent", "--isolated-home", "--no-enter", "--yes")
			if !strings.Contains(output, "Sandbox ready") {
				t.Fatalf("create output:\n%s", output)
			}
			assertExists(t, home)
			assertExists(t, metadata)

			info := runCLI(t, bin, env, "info", name)
			expectedNames := map[string]string{
				"ubuntu": "Ubuntu",
				"fedora": "Fedora",
				"debian": "Debian",
			}
			if !strings.Contains(info, expectedNames[distro]) {
				t.Errorf("info output does not report %q:\n%s", expectedNames[distro], info)
			}

			runCLI(t, bin, env, "delete", name, "--yes")
			assertMissing(t, home)
			assertMissing(t, metadata)
			assertContainerMissing(t, env, name)
		})
	}
}
