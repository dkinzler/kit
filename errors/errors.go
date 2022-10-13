// Package errors provides a custom error implementation that can be used throughout the project.
// A custom error can include e.g. an error code, message, origin, stack trace and wrap another error.
// The package also implements convenience functions, e.g. to check for certain error types or unwrap nested errors.
package errors

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// Instead of every package defining their own error codes, we can use a list of common error codes that apply to most situations.
// E.g. an error with code InvalidArgument might be returned by a data store if a query parameter is invalid or by a http handler if the request body cannot be decoded because it is not in json format.
// Having a small set of common error codes is also useful to e.g. automatically determine an appropriate HTTP response code by inspecting the error returned from some service method.
// This approach is copied from protobuf/grpc (see e.g. https://grpc.github.io/grpc/core/md_doc_statuscodes.html).
type ErrorCode int

const (
	Unknown ErrorCode = iota
	Cancelled
	//invalid argument was provided, e.g. an invalid date or malformed string, in contrast to FAILED_PRECONDITION the argument is invalid regardless of the state of the system
	InvalidArgument
	DeadlineExceeded
	NotFound
	AlreadyExists
	//caller does not have permission to execute the operation
	PermissionDenied
	//no or invalid authentication credentials were provided
	Unauthenticated
	//system is not in the correct state to execute the operation, e.g. in a banking applicaton we cannot perform a transfer if an account doesn't contain any money
	FailedPrecondition
	Aborted
	OutOfRange
	Unimplemented
	Internal
	Unavailable
)

func (e ErrorCode) String() string {
	s := [...]string{"Unknown", "Cancelled", "InvalidArgument", "DeadlineExceeded", "NotFound", "AlreadyExists", "PermissionDenied", "Unauthenticated", "FailedPrecondition", "Aborted", "OutOfRange", "Unimplemented", "Internal", "Unavailable"}
	if int(e) < len(s) {
		return s[e]
	}
	return "InvalidErrorCode"
}

// Structured error that can encode additional context about an error, e.g. the component the error originated in, a strack trace or an internal error message.
// All the fields are optional, although it makes sense to always provide at least the origin and a general error code.
type Error struct {
	//component this error was created in
	Origin string
	//wrap another error
	Inner      error
	StackTrace []byte

	//general error code, see comments at the definition of type ErrorCode
	Code ErrorCode

	//code or message that might be provided to a user/client
	//e.g. a http handler might return these in the response body (e.g. in json format) if the request fails
	//packages can use more specific error codes here, e.g. code 5 = "invalid username", code 6 = "password not strong enough", ...
	//Note: error codes should be > 0, a zero value will be interpreted as "no error code set"
	PublicCode    int
	PublicMessage string

	//code or message for internal use only, these could e.g. be written to application logs
	//Note: error codes should be > 0, a zero value will be interpreted as "no error code set"
	InternalCode    int
	InternalMessage string
}

// Errors can be build incrementally by chaining:
//
//	New(nil, "abc", InvalidArgument).WithPublicCode(1).WithPublicMessage("message")
//
// Packages/components can define their own functions to create errors more conveniently.
// E.g. a package "xyz" that usually sets Code and PublicMessage could use the following helper function:
//
//	func NewError(code ErrorCode, message string) error {
//		return New(nil, "xyz", code).WithPublicMessage(message)
//	}
func New(inner error, origin string, code ErrorCode) Error {
	var stack []byte
	//only add a stack trace if this is the innermost error
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

// implements the error interface
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

// error codes should be > 0, a zero value will be interpreted as "no error code set"
func (e Error) WithPublicCode(code int) Error {
	e.PublicCode = code
	return e
}

func (e Error) WithPublicMessage(message string) Error {
	e.PublicMessage = message
	return e
}

// error codes should be > 0, a zero value will be interpreted as "no error code set"
func (e Error) WithInternalCode(code int) Error {
	e.InternalCode = code
	return e
}

func (e Error) WithInternalMessage(message string) Error {
	e.InternalMessage = message
	return e
}

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
		// since this usually ends up as a json log message, we split the stack trace apart into individual lines
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
	return m
}

// Traverse inner errors and concatenate them in a slice
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

func HasInternalCode(err error, code int) bool {
	e, ok := err.(Error)
	if !ok {
		return false
	}
	return e.InternalCode == code
}

func HasPublicCode(err error, code int) bool {
	e, ok := err.(Error)
	if !ok {
		return false
	}
	return e.PublicCode == code
}
