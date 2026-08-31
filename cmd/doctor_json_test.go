package cmd

import (
	"encoding/json"
	"testing"
)

func TestChecksOK(t *testing.T) {
	all := []doctorCheck{{Name: "a", Ok: true}, {Name: "b", Ok: true}}
	if !checksOK(all) {
		t.Fatal("checksOK(all ok) = false, want true")
	}
	one := []doctorCheck{{Name: "a", Ok: true}, {Name: "b", Ok: false}}
	if checksOK(one) {
		t.Fatal("checksOK(one failing) = true, want false")
	}
}

func TestRenderDoctorJSONShapeAndOK(t *testing.T) {
	checks := []doctorCheck{
		{Name: "Podman installed", Ok: true},
		{Name: "Podman runtime working", Ok: true},
		{Name: "Podman rootless", Ok: true},
		{Name: "User namespaces configured", Ok: true},
	}
	data, err := renderDoctorJSON(checks)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		OK     bool `json:"ok"`
		Checks []struct {
			Name   string `json:"name"`
			OK     bool   `json:"ok"`
			Detail string `json:"detail"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal doctor JSON: %v\n%s", err, data)
	}
	if !out.OK {
		t.Fatalf("ok = false, want true:\n%s", data)
	}
	if len(out.Checks) != 4 {
		t.Fatalf("checks = %d, want 4:\n%s", len(out.Checks), data)
	}
	for i, check := range out.Checks {
		if check.Name != checks[i].Name || !check.OK {
			t.Fatalf("check %d = %+v, want %q with ok=true", i, check, checks[i].Name)
		}
		if check.Detail != "" {
			t.Fatalf("check %d has detail on success: %q", i, check.Detail)
		}
	}
}

func TestRenderDoctorJSONFailingCheckAndDetail(t *testing.T) {
	checks := []doctorCheck{
		{Name: "Podman installed", Ok: true},
		{Name: "Podman runtime working", Ok: false, Detail: "podman is not working: boom"},
		{Name: "Podman rootless", Ok: false},
		{Name: "User namespaces configured", Ok: true},
	}
	data, err := renderDoctorJSON(checks)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		OK     bool `json:"ok"`
		Checks []struct {
			Name   string `json:"name"`
			OK     bool   `json:"ok"`
			Detail string `json:"detail"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal doctor JSON: %v\n%s", err, data)
	}
	if out.OK {
		t.Fatalf("ok = true, want false:\n%s", data)
	}
	if out.Checks[1].Detail != "podman is not working: boom" {
		t.Fatalf("detail = %q, want the failing check detail", out.Checks[1].Detail)
	}
}
