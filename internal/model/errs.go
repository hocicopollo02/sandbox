package model

import "errors"

// Machine-readable error sentinels for agent integrations. Errors from the
// manager, metadata store and Podman client wrap one of these so that
// ErrorCode can classify failures without text matching.
var (
	ErrExists             = errors.New("resource already exists")
	ErrNotFound           = errors.New("resource not found")
	ErrInvalidName        = errors.New("invalid name")
	ErrRuntimeUnavailable = errors.New("runtime unavailable")
)

// ErrorCode maps err to a stable machine code for --error-format json.
func ErrorCode(err error) string {
	switch {
	case err == nil:
		return "E_ERROR"
	case errors.Is(err, ErrExists):
		return "E_EXISTS"
	case errors.Is(err, ErrNotFound):
		return "E_NOT_FOUND"
	case errors.Is(err, ErrInvalidName):
		return "E_INVALID_NAME"
	case errors.Is(err, ErrRuntimeUnavailable):
		return "E_RUNTIME"
	default:
		return "E_ERROR"
	}
}
