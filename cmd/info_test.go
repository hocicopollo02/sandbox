package cmd

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hocicopollo02/sandbox/internal/model"
)

func TestMarshalInfoJSONHasStableKeysWithEmptyHomePath(t *testing.T) {
	data, err := marshalInfoJSON(model.Info{
		Record: model.Record{
			Name:         "gentle-ai",
			Distribution: "arch",
			Image:        "docker.io/library/archlinux:latest",
			Persistence:  model.Persistent,
			HomeMode:     model.IsolatedHome,
			CreatedAt:    time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC),
		},
		Status: model.Running,
	})
	if err != nil {
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
	if string(raw["home_path"]) != `""` {
		t.Errorf("home_path = %s, want empty string", raw["home_path"])
	}
	if string(raw["status"]) != `"running"` {
		t.Errorf("status = %s, want running", raw["status"])
	}
}
