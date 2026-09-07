package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUninstallJSONRemovesCurrentExecutable(t *testing.T) {
	executable := writeFakeExecutable(t)
	data := filepath.Join(filepath.Dir(executable), "sandbox-data")
	if err := os.WriteFile(data, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	command := newUninstallCommand(&app{
		in:             strings.NewReader(""),
		out:            &out,
		executablePath: func() (string, error) { return executable, nil },
	})
	command.SetArgs([]string{"--yes", "--json"})

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(executable); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("executable still exists or could not be checked: %v", err)
	}
	if content, err := os.ReadFile(data); err != nil || string(content) != "keep" {
		t.Fatalf("uninstall changed adjacent data: content=%q error=%v", content, err)
	}

	var got struct {
		Name   string `json:"name"`
		Path   string `json:"path"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("uninstall output is not JSON: %v", err)
	}
	if got.Name != sandboxBinary || got.Path != executable || got.Result != "uninstalled" {
		t.Fatalf("uninstall JSON = %#v", got)
	}
}

func TestUninstallJSONRequiresConfirmation(t *testing.T) {
	executable := writeFakeExecutable(t)
	var out bytes.Buffer
	command := newUninstallCommand(&app{
		out:            &out,
		executablePath: func() (string, error) { return executable, nil },
	})
	command.SetArgs([]string{"--json"})

	if err := command.Execute(); err == nil {
		t.Fatal("uninstall --json succeeded without --yes")
	}
	if _, err := os.Stat(executable); err != nil {
		t.Fatalf("executable was removed after rejected confirmation: %v", err)
	}
}

func writeFakeExecutable(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), sandboxBinary)
	if err := os.WriteFile(path, []byte("sandbox"), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}
