package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestInfoJSONHasStableKeys(t *testing.T) {
	info := Info{
		Record: Record{
			Name:         "gentle-ai",
			Distribution: "arch",
			Image:        "docker.io/library/archlinux:latest",
			Persistence:  Persistent,
			HomeMode:     IsolatedHome,
			HomePath:     "/tmp/homes/gentle-ai",
			CreatedAt:    time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC),
		},
		Status: Running,
	}
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	if err := json.Unmarshal(data, &struct{}{}); err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal keys: %v", err)
	}
	want := []string{"name", "distribution", "image", "persistence", "home_mode", "home_path", "created_at", "status"}
	if len(raw) != len(want) {
		t.Fatalf("json keys = %d, want %d (%v); got %v", len(raw), len(want), want, raw)
	}
	for _, key := range want {
		if _, ok := raw[key]; !ok {
			t.Errorf("json key %q missing; got %v", key, raw)
		}
	}
	for key := range raw {
		found := false
		for _, expected := range want {
			if key == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected json key %q; want %v", key, want)
		}
	}
	if string(raw["status"]) != `"running"` {
		t.Errorf("status = %s, want \"running\"", raw["status"])
	}
	_ = keys
}
