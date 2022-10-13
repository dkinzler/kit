// Package endpoint provides additional functionality and helpers for github.com/go-kit/kit/endpoint .
package endpoint

/*
A type implementing Responder is used to wrap a result and error value from the business/service/component logic in an endpoint.
This makes it easier to distinguish between errors originating in the business/service/component logic of an endpoint and errors originating in endpoint code itself, e.g. a endpoint middleware.
An error from the business/service/component logic will be wrapped in the Responder type and therefore returned as part of the result (first) return value of the endpoint function.
Whereas an error from endpoint code will be returned as the error return value of the endpoint function itself.

The following examples demonstrate the intended usage of Responder:

	// uses the Response type, the default implementation of Responder
	func endpointFunc(ctx context.Context, request interface{}) (interface{}, error) {
		result, err := svc.SomeServiceMethod(ctx, request)
		//wrap error from service in Response, the error return value of this endpoint function will be nil
		return Response{
			R: result,
			Err: err,
		}, nil
	}

	// errors from an endpoint middleware will be returned in the error return value of the endpoint function
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

// default implementation of Responder
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

// compile-time assertion to make sure Response implements Responder
var _ Responder = Response{}
