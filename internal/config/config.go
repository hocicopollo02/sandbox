package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pablo/sandbox/internal/sandbox"
)

type Config struct {
	DefaultDistro      string
	DefaultPersistence sandbox.Persistence
	DefaultHome        sandbox.HomeMode
	AutoEnter          bool
}

func Defaults() Config {
	return Config{
		DefaultDistro:      "arch",
		DefaultPersistence: sandbox.Disposable,
		DefaultHome:        sandbox.IsolatedHome,
		AutoEnter:          true,
	}
}

func PathFor(home string) string {
	return filepath.Join(home, ".config", "sandbox", "config.toml")
}

func Load(home string) (Config, error) {
	cfg := Defaults()
	file, err := os.Open(PathFor(home))
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if text == "" {
			continue
		}
		parts := strings.SplitN(text, "=", 2)
		if len(parts) != 2 {
			return cfg, fmt.Errorf("invalid config at line %d: expected key = value", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		switch key {
		case "default_distro":
			if _, ok := sandbox.FindDistribution(value); !ok {
				return cfg, fmt.Errorf("invalid default_distro %q", value)
			}
			cfg.DefaultDistro = strings.ToLower(value)
		case "default_persistence":
			persistence := sandbox.Persistence(strings.ToLower(value))
			if persistence != sandbox.Disposable && persistence != sandbox.Persistent {
				return cfg, fmt.Errorf("invalid default_persistence %q", value)
			}
			cfg.DefaultPersistence = persistence
		case "default_home":
			homeMode := sandbox.HomeMode(strings.ToLower(value))
			if homeMode == sandbox.SharedHome {
				return cfg, fmt.Errorf("shared home is disabled: sandbox never mounts the host home")
			}
			if homeMode != sandbox.IsolatedHome {
				return cfg, fmt.Errorf("invalid default_home %q", value)
			}
			cfg.DefaultHome = homeMode
		case "auto_enter":
			autoEnter, err := strconv.ParseBool(value)
			if err != nil {
				return cfg, fmt.Errorf("invalid auto_enter %q", value)
			}
			cfg.AutoEnter = autoEnter
		}
	}
	if err := scanner.Err(); err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	return cfg, nil
}
