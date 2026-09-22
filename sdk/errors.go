package sdk

import "fmt"

type ErrorCode string

const (
	ErrorInvalidArgument ErrorCode = "invalid_argument"
	ErrorNotFound        ErrorCode = "not_found"
	ErrorConflict        ErrorCode = "conflict"
	ErrorUnauthorized    ErrorCode = "unauthorized"
	ErrorForbidden       ErrorCode = "forbidden"
	ErrorUnavailable     ErrorCode = "unavailable"
	ErrorInternal        ErrorCode = "internal"
)

type Error struct {
	Code                   ErrorCode   `json:"code"`
	Message                string      `json:"message"`
	Field                  string      `json:"field,omitempty"`
	Violations             []Violation `json:"violations,omitempty"`
	ExpectedSourceRevision int64       `json:"expectedSourceRevision,omitempty"`
	CurrentSourceRevision  int64       `json:"currentSourceRevision,omitempty"`
	ExpectedETag           int64       `json:"expectedEtag,omitempty"`
	CurrentETag            int64       `json:"currentEtag,omitempty"`
	Cause                  error       `json:"-"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type Violation struct {
	Location string `json:"location,omitempty"`
	Field    string `json:"field,omitempty"`
	Check    string `json:"check,omitempty"`
	Message  string `json:"message"`
}
