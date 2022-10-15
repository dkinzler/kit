// Package errors provides a structured error implementation that makes it easy
// to set and inspect error attributes like an error code, message or stack trace.
//
// Errors can be build incrementally by chaining:
//
//	New(nil, "abc", InvalidArgument).WithPublicCode(1).WithPublicMessage("message")
//
// Packages/components can define their own functions to create errors more conveniently.
// A package "xyz" that e.g. usually sets an error code and public message could use the following helper function:
//
//	func NewError(code ErrorCode, message string) error {
//	  return New(nil, "xyz", code).WithPublicMessage(message)
//	}
package errors

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// Structured error that can contain additional context about an error, e.g. the component
// the error originated in, a strack trace or an internal error message.
// All fields are optional, although it makes sense to at least provide the origin and a general error code.
type Error struct {
	Origin string
	// Wrap another error with additional context.
	Inner      error
	StackTrace []byte

	// General error code, see comments on type ErrorCode.
	Code ErrorCode

	// Code or message that is safe to be provided to outside systems/clients/users,
	// e.g. an http handler might return these in the response body if a request fails.
	// Packages can use this to provide more specific error codes, e.g. 5 = "invalid username" or 6 = "password not strong enough".
	// Note that a code should not be 0 since a zero value will be interpreted as no error code set.
	PublicCode    int
	PublicMessage string

	// Code or message for internal use only. These could e.g. be written to application logs.
	// Note that a code should not be 0 since a zero value will be interpreted as no error code set.
	InternalCode    int
	InternalMessage string

	// Any additional key-value pairs can be added here.
	KeyVals map[string]interface{}
}

// A stack trace is added automatically if the inner error is nil or not of type Error.
func New(inner error, origin string, code ErrorCode) Error {
	var stack []byte
	if _, ok := inner.(Error); !ok {
		stack = debug.Stack()
	}
	return Error{
		Origin:     origin,
		Inner:      inner,
		StackTrace: stack,
		Code:       code,
	}
}

// Implement the error interface.
func (e Error) Error() string {
	r := fmt.Sprintf("origin: %v, code: %v", e.Origin, e.Code.String())
	if e.PublicCode != 0 {
		r += fmt.Sprintf(", publicCode: %v", e.PublicCode)
	}
	if e.PublicMessage != "" {
		r += fmt.Sprintf(", publicMessage: %v", e.PublicMessage)
	}
	if e.InternalCode != 0 {
		r += fmt.Sprintf(", internalCode: %v", e.InternalCode)
	}
	if e.InternalMessage != "" {
		r += fmt.Sprintf(", internalMessage: %v", e.InternalMessage)
	}
	if e.StackTrace != nil {
		r += fmt.Sprintf(", stackTrace: %v", string(e.StackTrace))
	}
	for key, value := range e.KeyVals {
		r += fmt.Sprintf(", %v: %v", key, value)
	}

	if e.Inner != nil {
		r += fmt.Sprintf(", inner: [%v]", e.Inner.Error())
	}
	return r
}

func (e Error) WithOrigin(origin string) Error {
	e.Origin = origin
	return e
}

func (e Error) WithInner(inner error) Error {
	e.Inner = inner
	return e
}

func (e Error) WithCode(code ErrorCode) Error {
	e.Code = code
	return e
}

func (e Error) WithPublicCode(code int) Error {
	e.PublicCode = code
	return e
}

func (e Error) WithPublicMessage(message string) Error {
	e.PublicMessage = message
	return e
}

func (e Error) WithInternalCode(code int) Error {
	e.InternalCode = code
	return e
}

func (e Error) WithInternalMessage(message string) Error {
	e.InternalMessage = message
	return e
}

func (e Error) With(key string, value interface{}) Error {
	if e.KeyVals == nil {
		e.KeyVals = make(map[string]interface{})
	}
	e.KeyVals[key] = value
	return e
}

// Returns a map containing the non-zero field values of the error.
// Useful e.g. to log an error in JSON format.
func (e Error) ToMap() map[string]interface{} {
	m := map[string]interface{}{}
	if e.Origin != "" {
		m["origin"] = e.Origin
	}
	if e.Inner != nil {
		if inner, ok := e.Inner.(Error); ok {
			m["inner"] = inner.ToMap()
		} else {
			m["inner"] = e.Inner.Error()
		}
	}
	if e.StackTrace != nil {
		// Since this usually ends up as a json log message, we split the stack trace apart into individual lines.
		parts := strings.Split(strings.TrimSpace(string(e.StackTrace)), "\n")
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}
		m["stackTrace"] = parts
	}
	m["code"] = e.Code.String()
	if e.PublicCode != 0 {
		m["publicCode"] = e.PublicCode
	}
	if e.PublicMessage != "" {
		m["publicMessage"] = e.PublicMessage
	}
	if e.InternalCode != 0 {
		m["internalCode"] = e.InternalCode
	}
	if e.InternalMessage != "" {
		m["internalMessage"] = e.InternalMessage
	}
	for key, value := range e.KeyVals {
		m[key] = value
	}
	return m
}

// Return a slice of all the errors found by traversing inner errors.
func UnstackErrors(e error) []error {
	var result []error
	var err error = e
	v, ok := err.(Error)
	for ok {
		result = append(result, v)
		err = v.Inner
		v, ok = err.(Error)
	}
	if err != nil {
		result = append(result, err)
	}
	return result
}

// Retunrs true if the given error is of type Error and has the given ErrorCode set.
func Is(err error, code ErrorCode) bool {
	e, ok := err.(Error)
	if !ok {
		return false
	}
	return e.Code == code
}

func IsUnknownError(err error) bool {
	return Is(err, Unknown)
}

func IsCancelledError(err error) bool {
	return Is(err, Cancelled)
}

func IsInvalidArgumentError(err error) bool {
	return Is(err, InvalidArgument)
}

func IsDeadlineExceededError(err error) bool {
	return Is(err, DeadlineExceeded)
}

func IsNotFoundError(err error) bool {
	return Is(err, NotFound)
}

func IsAlreadyExistsError(err error) bool {
	return Is(err, AlreadyExists)
}

func IsPermissionDeniedError(err error) bool {
	return Is(err, PermissionDenied)
}

func IsUnauthenticatedError(err error) bool {
	return Is(err, Unauthenticated)
}

func IsFailedPreconditionError(err error) bool {
	return Is(err, FailedPrecondition)
}

func IsAbortedError(err error) bool {
	return Is(err, Aborted)
}

func IsOutOfRangeError(err error) bool {
	return Is(err, OutOfRange)
}

func IsUnimplementedError(err error) bool {
	return Is(err, Unimplemented)
}

func IsInternalError(err error) bool {
	return Is(err, Internal)
}

func IsUnavailableError(err error) bool {
	return Is(err, Unavailable)
}

// Returns true if the given error is of type Error and has the given code set as the internal error code.
func HasInternalCode(err error, code int) bool {
	e, ok := err.(Error)
	if !ok {
		return false
	}
	return e.InternalCode == code
}

// Returns true if the given error is of type Error and has the given code set as the public error code.
func HasPublicCode(err error, code int) bool {
	e, ok := err.(Error)
	if !ok {
		return false
	}
	return e.PublicCode == code
}

// Instead of every package defining their own error codes, we can use a list of common error codes that apply to most situations.
// E.g. an error with code InvalidArgument might be returned by a function if one of its parameter values is invalid or by a http handler if the request body cannot be decoded
// because it is not in the correct format.
// Having a small set of common error codes is also useful to e.g. automatically determine an appropriate HTTP response code
// by inspecting the error returned from some service method.
// This approach is copied from protobuf/grpc (see e.g. https://grpc.github.io/grpc/core/md_doc_statuscodes.html).
type ErrorCode int

const (
	Unknown ErrorCode = iota
	Cancelled
	// Invalid argument was provided, e.g. an invalid date or malformed string.
	// In contrast to FailedPrecondition the argument is invalid regardless of the state of the system.
	InvalidArgument
	DeadlineExceeded
	NotFound
	AlreadyExists
	// Caller does not have permission to execute the operation.
	PermissionDenied
	// No or invalid authentication credentials were provided.
	Unauthenticated
	// System is not in the correct state to execute the operation.
	// E.g. in a banking application we cannot perform a transfer if an account doesn't contain any money.
	FailedPrecondition
	Aborted
	OutOfRange
	Unimplemented
	Internal
	Unavailable
)

func (e ErrorCode) String() string {
	s := [...]string{
		"Unknown",
		"Cancelled",
		"InvalidArgument",
		"DeadlineExceeded",
		"NotFound",
		"AlreadyExists",
		"PermissionDenied",
		"Unauthenticated",
		"FailedPrecondition",
		"Aborted",
		"OutOfRange",
		"Unimplemented",
		"Internal",
		"Unavailable",
	}
	if int(e) >= 0 && int(e) < len(s) {
		return s[e]
	}
	return "UndefinedErrorCode"
}
