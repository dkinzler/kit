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

// Logs errors contained in the endpoint response, if the response type implements the Responder interface.
// Errors from endpoint code should be caught/logged with transport level error handlers (see e.g. the errorHandler option to github.com/go-kit/kit/transport/http.NewServer).
func ErrorLoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			defer func() {
				resp, ok := response.(Responder)
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

// Endpoint middleware that records to the given histrogram the time it takes the endpoint to process requests.
func InstrumentRequestTimeMiddleware(duration metrics.Histogram) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (result interface{}, err error) {
			defer func(begin time.Time) {
				t := float64(time.Since(begin).Milliseconds())

				resp, ok := result.(Responder)
				if ok {
					// If the result value implements Responder, label the observation based on success,
					// i.e. if the error returned was nil or not.
					duration.With("success", fmt.Sprint(resp.Error() == nil)).Observe(t)
				} else {
					duration.Observe(t)
				}
			}(time.Now())
			return next(ctx, request)
		}
	}
}

// Applies zero or more middlewares to an endpoint.
// Middlewares are applied in order, i.e. first middleware passed is applied first and therefore innermost,
// last middleware passed is outermost.
func ApplyMiddlewares(e endpoint.Endpoint, mws ...endpoint.Middleware) endpoint.Endpoint {
	var result endpoint.Endpoint = e
	for _, mw := range mws {
		result = mw(result)
	}
	return result
}
