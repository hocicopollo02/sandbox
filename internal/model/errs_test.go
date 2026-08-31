package model

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorCodeMapsSentinels(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{ErrExists, "E_EXISTS"},
		{ErrNotFound, "E_NOT_FOUND"},
		{ErrInvalidName, "E_INVALID_NAME"},
		{ErrRuntimeUnavailable, "E_RUNTIME"},
		{errors.New("something else"), "E_ERROR"},
		{nil, "E_ERROR"},
	} {
		if got := ErrorCode(test.err); got != test.want {
			t.Errorf("ErrorCode(%v) = %q, want %q", test.err, got, test.want)
		}
	}
}

func TestErrorCodeMatchesWrappedSentinels(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{fmt.Errorf("sandbox %q already exists: %w", "x", ErrExists), "E_EXISTS"},
		{fmt.Errorf("sandbox %q does not exist: %w", "x", ErrNotFound), "E_NOT_FOUND"},
		{fmt.Errorf("invalid sandbox name: %w", ErrInvalidName), "E_INVALID_NAME"},
		{fmt.Errorf("podman is required: %w", ErrRuntimeUnavailable), "E_RUNTIME"},
	} {
		if got := ErrorCode(test.err); got != test.want {
			t.Errorf("ErrorCode(%v) = %q, want %q", test.err, got, test.want)
		}
	}
}

// TestErrorCodeUnwrapsChain verifies errors.Is semantics through ErrorCode.
func TestErrorCodeUnwrapsChain(t *testing.T) {
	err := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", ErrNotFound))
	if got := ErrorCode(err); got != "E_NOT_FOUND" {
		t.Errorf("ErrorCode(chain) = %q, want E_NOT_FOUND", got)
	}
}
