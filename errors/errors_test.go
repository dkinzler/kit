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
	a.Equal("InvalidErrorCode", ErrorCode(42).String())
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
	//we define these here, since multiple instances created with New(nil, "x", y) will not be equal because of the stack trace contained
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
