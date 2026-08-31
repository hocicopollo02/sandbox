package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hocicopollo02/sandbox/internal/model"
)

func TestRenderErrorTextKeepsHumanMessage(t *testing.T) {
	line, isJSON := renderError(errors.New("boom"), "text")
	if isJSON {
		t.Fatalf("text format reported machine line")
	}
	if line != "" {
		t.Fatalf("text format returned %q, want empty", line)
	}
}

func TestRenderErrorJSONCarriesCodeAndMessage(t *testing.T) {
	err := fmt.Errorf("sandbox %q does not exist: %w", "ghost", model.ErrNotFound)
	line, isJSON := renderError(err, "json")
	if !isJSON {
		t.Fatal("json format did not report a machine line")
	}
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal([]byte(line), &payload); decodeErr != nil {
		t.Fatalf("invalid JSON line %q: %v", line, decodeErr)
	}
	if payload.Error.Code != "E_NOT_FOUND" {
		t.Errorf("code = %q, want E_NOT_FOUND", payload.Error.Code)
	}
	if !strings.Contains(payload.Error.Message, "ghost") {
		t.Errorf("message = %q, want sandbox name included", payload.Error.Message)
	}
}

func TestRenderErrorJSONDefaultsToGenericCode(t *testing.T) {
	line, _ := renderError(errors.New("random failure"), "json")
	if !strings.Contains(line, `"code":"E_ERROR"`) {
		t.Errorf("generic error line = %q, want E_ERROR code", line)
	}
}
