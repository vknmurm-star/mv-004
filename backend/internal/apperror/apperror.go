package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// Code is a stable machine-readable error code.
type Code string

const (
	CodeInvalid       Code = "invalid"
	CodeNotFound      Code = "not_found"
	CodeConflict      Code = "conflict"
	CodeUnauthorized  Code = "unauthorized"
	CodeForbidden     Code = "forbidden"
	CodeRateLimited   Code = "rate_limited"
	CodeInternal      Code = "internal"
	CodeUnprocessable Code = "unprocessable"
)

// Error is a domain-level error carrying a stable Code and a human message.
type Error struct {
	Code    Code
	Message string
	Err     error
	Status  int
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

func New(c Code, msg string, status int) *Error {
	return &Error{Code: c, Message: msg, Status: status}
}

func Wrap(c Code, msg string, status int, err error) *Error {
	return &Error{Code: c, Message: msg, Status: status, Err: err}
}

// Convenience constructors.
func Invalid(msg string) *Error      { return New(CodeInvalid, msg, http.StatusBadRequest) }
func NotFound(msg string) *Error     { return New(CodeNotFound, msg, http.StatusNotFound) }
func Conflict(msg string) *Error     { return New(CodeConflict, msg, http.StatusConflict) }
func Unauthorized(msg string) *Error { return New(CodeUnauthorized, msg, http.StatusUnauthorized) }
func Forbidden(msg string) *Error    { return New(CodeForbidden, msg, http.StatusForbidden) }
func RateLimited(msg string) *Error  { return New(CodeRateLimited, msg, http.StatusTooManyRequests) }
func Internal(msg string, err error) *Error {
	return Wrap(CodeInternal, msg, http.StatusInternalServerError, err)
}
func Unprocessable(msg string) *Error {
	return New(CodeUnprocessable, msg, http.StatusUnprocessableEntity)
}

// AsCode returns the Code of an error, defaulting to CodeInternal.
func AsCode(err error) Code {
	var ae *Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return CodeInternal
}

// StatusOf returns the HTTP status for an error, defaulting to 500.
func StatusOf(err error) int {
	var ae *Error
	if errors.As(err, &ae) {
		if ae.Status == 0 {
			return http.StatusInternalServerError
		}
		return ae.Status
	}
	return http.StatusInternalServerError
}
