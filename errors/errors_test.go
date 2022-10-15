package errors

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorCode(t *testing.T) {
	a := assert.New(t)

	a.Equal(ErrorCode(0), Unknown)
	a.Equal(ErrorCode(1), Cancelled)
	a.Equal(ErrorCode(2), InvalidArgument)

	a.Equal("Unknown", Unknown.String())
	a.Equal("InvalidArgument", InvalidArgument.String())
	a.Equal("UndefinedErrorCode", ErrorCode(42).String())
}

func TestNew(t *testing.T) {
	a := assert.New(t)

	inner := stderrors.New("xyz")
	err := New(inner, "test", InvalidArgument)
	a.Equal("test", err.Origin)
	a.Equal(inner, err.Inner)
	a.Equal(InvalidArgument, err.Code)
	a.NotNil(err.StackTrace)

	err = Error{}
	a.Equal(err.WithOrigin("origin").Origin, "origin")
	a.Equal(err.WithInner(inner).Inner, inner)
	a.Equal(err.WithCode(Unimplemented).Code, Unimplemented)
	a.Equal(err.WithPublicCode(42).PublicCode, 42)
	a.Equal(err.WithPublicMessage("pmessage").PublicMessage, "pmessage")
	a.Equal(err.WithInternalCode(43).InternalCode, 43)
	a.Equal(err.WithInternalMessage("imessage").InternalMessage, "imessage")

	err = err.With("testkey", 1001)
	err = err.With("anotherkey", "value")
	v, ok := err.KeyVals["testkey"]
	a.True(ok)
	a.Equal(1001, v)
	v, ok = err.KeyVals["anotherkey"]
	a.True(ok)
	a.Equal("value", v)
}

func TestErrorString(t *testing.T) {
	a := assert.New(t)

	inner := stderrors.New("xyz")
	err := New(inner, "testorigin", InvalidArgument).
		WithInternalCode(42).
		WithInternalMessage("internal message").
		WithPublicCode(43).
		WithPublicMessage("public message").
		With("key", "value")

	s := err.Error()
	a.Contains(s, "origin: testorigin")
	a.Contains(s, "code: InvalidArgument")
	a.Contains(s, "internalCode: 42")
	a.Contains(s, "internalMessage: internal message")
	a.Contains(s, "publicCode: 43")
	a.Contains(s, "publicMessage: public message")
	a.Contains(s, "key: value")
	a.Contains(s, "inner: ["+inner.Error()+"]")
	a.Contains(s, "stackTrace:")
}

func TestErrorToMap(t *testing.T) {
	a := assert.New(t)

	inner := stderrors.New("xyz")
	err := New(inner, "testorigin", InvalidArgument).
		WithInternalCode(42).
		WithInternalMessage("internal message").
		WithPublicCode(43).
		WithPublicMessage("public message").
		With("key", "value")

	m := err.ToMap()
	a.Equal("testorigin", m["origin"])
	a.Equal("InvalidArgument", m["code"])
	a.Equal(42, m["internalCode"])
	a.Equal("internal message", m["internalMessage"])
	a.Equal(43, m["publicCode"])
	a.Equal("public message", m["publicMessage"])
	a.Equal("value", m["key"])
	a.Equal(inner.Error(), m["inner"])
	a.Contains(m, "stackTrace")

	// inner error of type Error should also be encoded as map

	inner = New(nil, "innerorigin", FailedPrecondition)
	err = New(inner, "test", InvalidArgument)
	m = err.ToMap()
	a.Equal("test", m["origin"])
	a.Equal("InvalidArgument", m["code"])
	a.Equal(inner.(Error).ToMap(), m["inner"])
}

func TestIs(t *testing.T) {
	a := assert.New(t)

	cases := []struct {
		Err      error
		Code     ErrorCode
		Expected bool
	}{
		{
			Err:      nil,
			Code:     InvalidArgument,
			Expected: false,
		},
		{
			Err:      stderrors.New("just some error"),
			Code:     InvalidArgument,
			Expected: false,
		},
		{
			Err:      New(nil, "test", InvalidArgument),
			Code:     InvalidArgument,
			Expected: true,
		},
		{
			Err:      New(nil, "test", InvalidArgument),
			Code:     PermissionDenied,
			Expected: false,
		},
		{
			Err:      New(nil, "test", Aborted),
			Code:     Aborted,
			Expected: true,
		},
	}
	for i, c := range cases {
		actual := Is(c.Err, c.Code)
		a.Equal(c.Expected, actual, "case %v", i)
	}

	a.True(IsInvalidArgumentError(New(nil, "test", InvalidArgument)))
	a.False(IsInvalidArgumentError(New(nil, "test", AlreadyExists)))
	a.True(IsAlreadyExistsError(New(nil, "test", AlreadyExists)))
	a.True(IsAbortedError(New(nil, "test", Aborted)))
	a.True(IsCancelledError(New(nil, "test", Cancelled)))
	a.True(IsDeadlineExceededError(New(nil, "test", DeadlineExceeded)))
	a.True(IsFailedPreconditionError(New(nil, "test", FailedPrecondition)))
	a.True(IsInternalError(New(nil, "test", Internal)))
	a.True(IsNotFoundError(New(nil, "test", NotFound)))
	a.True(IsOutOfRangeError(New(nil, "test", OutOfRange)))
	a.True(IsPermissionDeniedError(New(nil, "test", PermissionDenied)))
	a.True(IsUnauthenticatedError(New(nil, "test", Unauthenticated)))
	a.True(IsUnavailableError(New(nil, "test", Unavailable)))
	a.True(IsUnimplementedError(New(nil, "test", Unimplemented)))
	a.True(IsUnknownError(New(nil, "test", Unknown)))
}

func TestHasCode(t *testing.T) {
	a := assert.New(t)

	// test HasPublicCode
	cases := []struct {
		Err      error
		Code     int
		Expected bool
	}{
		{
			Err:      New(nil, "test", InvalidArgument).WithPublicCode(42),
			Code:     42,
			Expected: true,
		},
		{
			Err:      New(nil, "test", InvalidArgument).WithPublicCode(42),
			Code:     43,
			Expected: false,
		},
		{
			Err:      stderrors.New("testerror"),
			Code:     0,
			Expected: false,
		},
	}

	for i, c := range cases {
		a.Equal(c.Expected, HasPublicCode(c.Err, c.Code), "test case %v", i)
	}

	cases = []struct {
		Err      error
		Code     int
		Expected bool
	}{
		{
			Err:      New(nil, "test", InvalidArgument).WithInternalCode(43),
			Code:     43,
			Expected: true,
		},
		{
			Err:      New(nil, "test", InvalidArgument).WithPublicCode(43),
			Code:     44,
			Expected: false,
		},
		{
			Err:      stderrors.New("testerror"),
			Code:     0,
			Expected: false,
		},
	}

	for i, c := range cases {
		a.Equal(c.Expected, HasInternalCode(c.Err, c.Code), "test case %v", i)
	}
}

func TestUnstackErrors(t *testing.T) {
	// We define and reuse this error since multiple instances created with New(nil, "x", InvalidArgument) will not be equal because of the stack trace.
	case1 := New(nil, "test", InvalidArgument)

	e := New(New(New(stderrors.New("test"), "anotherone", 9001), "othertest", InvalidArgument), "test", InvalidArgument)
	ee := e.Inner.(Error)
	eee := ee.Inner.(Error)
	eeee := eee.Inner

	cases := []struct {
		Err      error
		Expected []error
	}{
		{
			Err:      nil,
			Expected: nil,
		},
		{
			Err:      stderrors.New("test"),
			Expected: []error{stderrors.New("test")},
		},
		{
			Err: case1,
			Expected: []error{
				case1,
			},
		},
		{
			Err: e,
			Expected: []error{
				e,
				ee,
				eee,
				eeee,
			},
		},
	}
	for i, c := range cases {
		actual := UnstackErrors(c.Err)
		assert.Equal(t, c.Expected, actual, "case %v", i)
	}
}
