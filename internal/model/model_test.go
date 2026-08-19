package model

import "testing"

func TestValidateName(t *testing.T) {
	got, err := ValidateName("  gentle-ai ")
	if err != nil || got != "gentle-ai" {
		t.Fatalf("ValidateName() = %q, %v", got, err)
	}
	for _, name := range []string{"", "Gentle", "has space", "../escape"} {
		if _, err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) accepted invalid name", name)
		}
	}
}

func TestFindDistribution(t *testing.T) {
	got, ok := FindDistribution(" ARCH ")
	if !ok || got.Image != "docker.io/library/archlinux:latest" {
		t.Fatalf("FindDistribution() = %#v, %v", got, ok)
	}
}

func TestDisposableRequiresEntry(t *testing.T) {
	distro, _ := FindDistribution("arch")
	if err := (CreateOptions{
		Name: "temp", Distribution: distro, Persistence: Disposable, HomeMode: IsolatedHome,
	}).Validate(); err == nil {
		t.Fatal("disposable sandbox without auto-enter was accepted")
	}
}
