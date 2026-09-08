package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestVersionJSONIncludesBuildIdentity(t *testing.T) {
	var out bytes.Buffer
	command := newVersionCommand(&app{out: &out})
	command.SetArgs([]string{"--json"})

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}

	var got map[string]string
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("version output is not JSON: %v", err)
	}
	want := map[string]string{
		"name":       "sandbox",
		"version":    Version,
		"commit":     Commit,
		"build_date": BuildDate,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("version JSON = %#v, want %#v", got, want)
	}
}

func TestRootVersionFlagsPrintVersion(t *testing.T) {
	for _, flag := range []string{"--version", "-v"} {
		t.Run(flag, func(t *testing.T) {
			var out bytes.Buffer
			root, err := NewRootCommand(t.TempDir(), bytes.NewBuffer(nil), &out, &bytes.Buffer{})
			if err != nil {
				t.Fatal(err)
			}
			root.SetArgs([]string{flag})
			root.SetOut(&out)

			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if got, want := out.String(), fmt.Sprintf("sandbox %s\n", Version); got != want {
				t.Fatalf("version output = %q, want %q", got, want)
			}
		})
	}
}
