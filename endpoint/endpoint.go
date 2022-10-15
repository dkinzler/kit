// Package endpoint provides helpers and additional functionality for github.com/go-kit/kit/endpoint .
package endpoint

/*
A type implementing Responder can be used to wrap a result and error value returned by the
business logic/service/component part of an endpoint.
This makes it easier to distinguish between errors originating in business logic/service/component code of an endpoint
and errors originating in endpoint code itself, e.g. an endpoint middleware.
An error from the business logic/service/component code will be wrapped in the Responder type
and therefore returned as part of the "response" (first) return value of an endpoint function.
Whereas an error from endpoint code will be returned as the error return value of an endpoint function.

The following examples demonstrate the intended usage of Responder:

	func endpointFunc(ctx context.Context, request interface{}) (interface{}, error) {
		result, err := service.SomeServiceMethod(ctx, request)
		// Wrap error from service in Response, the default type implementing Responder.
		// The error return value of this endpoint function will be nil.
		return Response{
			R: result,
			Err: err,
		}, nil
	}

	// Errors from an endpoint middleware will be returned in the error return value of the endpoint function.
	func exampleEndpointMiddleware() endpoint.Middleware {
		return func(next endpoint.Endpoint) endpoint.Endpoint {
			return func(ctx context.Context, request interface{}) (interface{}, error) {
				err := PerformSomeOperationThatMightFail(request)
				if err != nil {
					return nil, err
				}
				return next(ctx, request)
			}
		}
	}
*/
type Responder interface {
	Response() interface{}
	Error() error
}

// Default implementation of Responder.
type Response struct {
	R   interface{}
	Err error
}

func (r Response) Response() interface{} {
	return r.R
}

func (r Response) Error() error {
	return r.Err
}

// Compile-time assertion that makes sure Response implements Responder.
var _ Responder = Response{}
