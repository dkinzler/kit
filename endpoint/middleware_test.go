package endpoint

import (
	"context"
	stderrors "errors"
	"strconv"
	"testing"

	"github.com/dkinzler/kit/errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/stretchr/testify/assert"
)

type testLogger struct {
	keyVals map[string]interface{}
}

func (t *testLogger) Log(keyvals ...interface{}) error {
	r := make(map[string]interface{})
	var key string
	var ok bool
	for i, x := range keyvals {
		if i%2 == 0 {
			key, ok = x.(string)
			if !ok {
				key = strconv.Itoa(i)
			}
		} else {
			r[key] = x
		}
	}
	t.keyVals = r
	return nil
}

func TestErrorLoggingMiddleware(t *testing.T) {
	a := assert.New(t)
	//logger not called if no error returned
	logger := &testLogger{}
	mw := ErrorLoggingMiddleware(logger)
	endpoint := func(ctx context.Context, request interface{}) (interface{}, error) {
		return Response{
			R:   nil,
			Err: nil,
		}, nil
	}
	mw(endpoint)(context.Background(), nil)
	a.Empty(logger.keyVals)

	//logger not called on error from endpoint func
	var err error
	err = errors.New(nil, "test", errors.Internal)
	logger = &testLogger{}
	mw = ErrorLoggingMiddleware(logger)
	endpoint = func(ctx context.Context, request interface{}) (interface{}, error) {
		return Response{
			R:   nil,
			Err: nil,
		}, err
	}
	mw(endpoint)(context.Background(), nil)
	a.Empty(logger.keyVals)

	//logger called for errors.Error in endpoint result
	logger = &testLogger{}
	mw = ErrorLoggingMiddleware(logger)
	endpoint = func(ctx context.Context, request interface{}) (interface{}, error) {
		return Response{
			R:   nil,
			Err: err,
		}, nil
	}
	mw(endpoint)(context.Background(), nil)
	a.NotEmpty(logger.keyVals)
	a.Contains(logger.keyVals, "error")
	a.Equal(err.(errors.Error).ToMap(), logger.keyVals["error"])

	//logger called for other error in endpoint result
	err = stderrors.New("test")
	logger = &testLogger{}
	mw = ErrorLoggingMiddleware(logger)
	endpoint = func(ctx context.Context, request interface{}) (interface{}, error) {
		return Response{
			R:   nil,
			Err: err,
		}, nil
	}
	mw(endpoint)(context.Background(), nil)
	a.NotEmpty(logger.keyVals)
	a.Contains(logger.keyVals, "error")
	a.Equal(err, logger.keyVals["error"])
}

func TestApplyMiddlewares(t *testing.T) {
	a := assert.New(t)

	//just endpoint called if no middlewares provided
	ep := func(ctx context.Context, request interface{}) (interface{}, error) {
		return Response{
			R:   "test",
			Err: nil,
		}, nil
	}
	mwe := ApplyMiddlewares(ep)
	r, err := mwe(context.Background(), nil)
	a.Nil(err)
	resp, ok := r.(Response)
	a.True(ok)
	a.Equal("test", resp.R)

	//middlewares applied in correct order
	//i.e. first middleware passed is innermost and last middleware passed is outermost
	ep = func(ctx context.Context, request interface{}) (interface{}, error) {
		return []int{}, nil
	}
	makeMw := func(i int) endpoint.Middleware {
		return func(next endpoint.Endpoint) endpoint.Endpoint {
			return func(ctx context.Context, request interface{}) (interface{}, error) {
				r, _ := next(ctx, request)
				if s, ok := r.([]int); ok {
					r = append(s, i)
				}
				return r, nil
			}
		}
	}
	r, err = ApplyMiddlewares(ep, []endpoint.Middleware{
		makeMw(1),
		makeMw(2),
		makeMw(3),
	}...)(context.Background(), nil)
	a.Nil(err)
	is, ok := r.([]int)
	a.True(ok)
	a.Equal([]int{1, 2, 3}, is)
}
