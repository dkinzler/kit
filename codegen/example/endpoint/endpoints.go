// generated code, do not modify
package endpoint

import (
	"context"
	example "example/codegen/example"

	e "github.com/dkinzler/kit/endpoint"
	endpoint "github.com/go-kit/kit/endpoint"
)

type Method1Request struct {
	A string
	X example.X
}

func MakeMethod1Endpoint(svc example.ExampleInterface) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(Method1Request)
		r, err := svc.Method1(ctx, req.A, req.X)
		return e.Response{
			Err: err,
			R:   r,
		}, nil
	}
}
func MakeAnotherOneEndpoint(svc example.ExampleInterface) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(Method1Request)
		r, err := svc.Method1(ctx, req.A, req.X)
		return e.Response{
			Err: err,
			R:   r,
		}, nil
	}
}

type Method2Request struct {
	A  string
	Qp example.QueryParams
}

func MakeMethod2Endpoint(svc example.ExampleInterface) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(Method2Request)
		err := svc.Method2(ctx, req.A, req.Qp)
		return e.Response{
			Err: err,
			R:   nil,
		}, nil
	}
}

type EndpointSet struct {
	Method1Endpoint    endpoint.Endpoint
	AnotherOneEndpoint endpoint.Endpoint
	Method2Endpoint    endpoint.Endpoint
}

type Middlewares struct {
	Method1Endpoint    []endpoint.Middleware
	AnotherOneEndpoint []endpoint.Middleware
	Method2Endpoint    []endpoint.Middleware
}

func NewEndpoints(svc example.ExampleInterface, mws Middlewares) EndpointSet {
	var method1Endpoint endpoint.Endpoint
	{
		method1Endpoint = MakeMethod1Endpoint(svc)
		method1Endpoint = e.ApplyMiddlewares(method1Endpoint, mws.Method1Endpoint...)
	}

	var anotherOneEndpoint endpoint.Endpoint
	{
		anotherOneEndpoint = MakeAnotherOneEndpoint(svc)
		anotherOneEndpoint = e.ApplyMiddlewares(anotherOneEndpoint, mws.AnotherOneEndpoint...)
	}

	var method2Endpoint endpoint.Endpoint
	{
		method2Endpoint = MakeMethod2Endpoint(svc)
		method2Endpoint = e.ApplyMiddlewares(method2Endpoint, mws.Method2Endpoint...)
	}

	return EndpointSet{
		AnotherOneEndpoint: anotherOneEndpoint,
		Method1Endpoint:    method1Endpoint,
		Method2Endpoint:    method2Endpoint,
	}
}
