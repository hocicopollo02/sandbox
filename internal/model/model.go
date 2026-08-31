package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Persistence string

const (
	Disposable Persistence = "disposable"
	Persistent Persistence = "persistent"
)

type HomeMode string

const (
	IsolatedHome HomeMode = "isolated"
	SharedHome   HomeMode = "shared"
)

type Status string

const (
	Running Status = "running"
	Stopped Status = "stopped"
	Missing Status = "missing"
	Unknown Status = "unknown"
)

type Distribution struct {
	ID    string
	Name  string
	Image string
}

var distributions = []Distribution{
	{ID: "arch", Name: "Arch Linux", Image: "docker.io/library/archlinux:latest"},
	{ID: "ubuntu", Name: "Ubuntu", Image: "docker.io/library/ubuntu:24.04"},
	{ID: "fedora", Name: "Fedora", Image: "registry.fedoraproject.org/fedora:latest"},
	{ID: "debian", Name: "Debian", Image: "docker.io/library/debian:stable"},
}

func Distributions() []Distribution {
	return append([]Distribution(nil), distributions...)
}

func FindDistribution(id string) (Distribution, bool) {
	for _, distro := range distributions {
		if distro.ID == strings.ToLower(strings.TrimSpace(id)) {
			return distro, true
		}
	}
	return Distribution{}, false
}

var validName = regexp.MustCompile(`^[a-z0-9_-]+$`)

func ValidateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("sandbox name is required")
	}
	if !validName.MatchString(name) {
		return "", fmt.Errorf("invalid sandbox name %q: use lowercase letters, numbers, _ or -", name)
	}
	return name, nil
}

type Record struct {
	Name         string      `json:"name"`
	Distribution string      `json:"distribution"`
	Image        string      `json:"image"`
	Persistence  Persistence `json:"persistence"`
	HomeMode     HomeMode    `json:"home_mode"`
	HomePath     string      `json:"home_path,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
}

type ListEntry struct {
	Name         string      `json:"name"`
	Distribution string      `json:"distro"`
	Persistence  Persistence `json:"persistence"`
	HomeMode     HomeMode    `json:"home"`
	Status       Status      `json:"status"`
}

type Info struct {
	Record
	Status Status `json:"status"`
}

type CreateOptions struct {
	Name         string
	Distribution Distribution
	Persistence  Persistence
	HomeMode     HomeMode
	AutoEnter    bool
	IfNotExists  bool
}

func (o CreateOptions) Validate() error {
	if _, err := ValidateName(o.Name); err != nil {
		return err
	}
	if o.Distribution.ID == "" || o.Distribution.Image == "" {
		return fmt.Errorf("a valid distribution is required")
	}
	if o.Persistence != Disposable && o.Persistence != Persistent {
		return fmt.Errorf("invalid persistence %q", o.Persistence)
	}
	if o.HomeMode == SharedHome {
		return fmt.Errorf("shared home is disabled: sandbox never mounts the host home")
	}
	if o.HomeMode != IsolatedHome {
		return fmt.Errorf("invalid home mode %q", o.HomeMode)
	}
	if o.Persistence == Disposable && !o.AutoEnter {
		return fmt.Errorf("disposable sandboxes must be entered immediately; remove --no-enter")
	}
	return nil
}

type DeleteOptions struct {
	DeleteHome bool
	IfExists   bool
}
