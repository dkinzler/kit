// generated code, do not modify
package http

import (
	"context"
	endpoint "example/codegen/endpoint"
	example "example/codegen/example"
	"net/http"

	t "github.com/dkinzler/kit/transport/http"
	kithttp "github.com/go-kit/kit/transport/http"
	mux "github.com/gorilla/mux"
)

func decodeHttpMethod1Request(ctx context.Context, r *http.Request) (interface{}, error) {
	a, err := t.DecodeURLParameter(r, "a")
	if err != nil {
		return nil, err
	}

	var x example.X
	err = t.DecodeJSONBody(r, &x)
	if err != nil {
		return nil, err
	}

	return endpoint.Method1Request{
		A: a,
		X: x,
	}, nil
}

func decodeHttpMethod2Request(ctx context.Context, r *http.Request) (interface{}, error) {
	a, err := t.DecodeURLParameter(r, "a")
	if err != nil {
		return nil, err
	}

	var qp example.QueryParams
	err = t.DecodeQueryParameters(r, &qp)
	if err != nil {
		return nil, err
	}

	return endpoint.Method2Request{
		A:  a,
		Qp: qp,
	}, nil
}

func RegisterHttpHandlers(endpoints endpoint.EndpointSet, router *mux.Router, opts []kithttp.ServerOption) {
	anotherOneHandler := kithttp.NewServer(endpoints.AnotherOneEndpoint, decodeHttpMethod1Request, t.MakeGenericJSONEncodeFunc(201), opts...)
	router.Handle("/other/path/{a}", anotherOneHandler).Methods("POST", "OPTIONS")

	method1Handler := kithttp.NewServer(endpoints.Method1Endpoint, decodeHttpMethod1Request, t.MakeGenericJSONEncodeFunc(201), opts...)
	router.Handle("/some/path/{a}", method1Handler).Methods("POST", "OPTIONS")

	method2Handler := kithttp.NewServer(endpoints.Method2Endpoint, decodeHttpMethod2Request, t.MakeGenericJSONEncodeFunc(200), opts...)
	router.Handle("/some/path/{a}", method2Handler).Methods("GET", "OPTIONS")
}
