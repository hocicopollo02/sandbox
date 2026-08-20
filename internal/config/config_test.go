package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hocicopollo02/sandbox/internal/sandbox"
)

func TestLoadDefaultsWhenConfigIsMissing(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultDistro != "arch" || cfg.DefaultPersistence != sandbox.Disposable || cfg.DefaultHome != sandbox.IsolatedHome || !cfg.AutoEnter {
		t.Fatalf("defaults = %#v", cfg)
	}
}

func TestLoadConfigValues(t *testing.T) {
	home := t.TempDir()
	path := PathFor(home)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`default_distro = "ubuntu"
default_persistence = "persistent"
default_home = "isolated"
auto_enter = false
`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(home)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultDistro != "ubuntu" || cfg.DefaultPersistence != sandbox.Persistent || cfg.DefaultHome != sandbox.IsolatedHome || cfg.AutoEnter {
		t.Fatalf("loaded config = %#v", cfg)
	}
}

func TestLoadRejectsSharedHome(t *testing.T) {
	home := t.TempDir()
	path := PathFor(home)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("default_home = \"shared\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(home)
	if err == nil || !strings.Contains(err.Error(), "shared home is disabled") {
		t.Fatalf("Load() error = %v", err)
	}
}
