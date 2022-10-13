package endpoint

import (
	"context"
	"fmt"
	"time"

	"github.com/d39b/kit/errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/log"
)

// Logs errors from the underlying business/service/component logic of the endpoint, if the result value implements the Responder interface.
// Errors from endpoint code should be caught/logged with the transport level error handler (see e.g. the errorHandler option to github.com/go-kit/kit/transport/http.NewServer)
func ErrorLoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (result interface{}, err error) {
			defer func() {
				resp, ok := result.(Responder)
				if ok && resp.Error() != nil {
					e, ok := resp.Error().(errors.Error)
					if ok {
						logger.Log("error", e.ToMap())
					} else {
						logger.Log("error", resp.Error())
					}
				}
			}()
			return next(ctx, request)
		}
	}
}

// Records the time it took to process the request.
func InstrumentRequestTimeMiddleware(duration metrics.Histogram) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (result interface{}, err error) {
			defer func(begin time.Time) {
				resp, ok := result.(Responder)
				if ok {
					//if the result value implements Responder, label the obervation based on success, i.e. if the error returned was nil or not
					duration.With("success", fmt.Sprint(resp.Error() == nil)).Observe(float64(time.Since(begin).Milliseconds()))
				} else {
					duration.Observe(float64(time.Since(begin).Milliseconds()))
				}
			}(time.Now())
			return next(ctx, request)
		}
	}
}

// Applies zero or more middlewares to an Endpoint.
// Middlewares are applied in order, first middleware passed is applied first and therefore innermost, last middleware passed is outermost.
func ApplyMiddlewares(e endpoint.Endpoint, mws ...endpoint.Middleware) endpoint.Endpoint {
	var result endpoint.Endpoint = e
	for _, mw := range mws {
		result = mw(result)
	}
	return result
}
