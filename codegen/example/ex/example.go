package ex

import (
	"context"
)

type X struct {
	A []string
	B map[string]int
}

type Y struct {
	C float64
	D map[string][]int
}

type QueryParams struct {
	A int
	B string
}

/*
	@Kit{
		"endpointPackage": "endpoint",
		"endpointOutput": "endpoints.go",
		"httpPackage": "http",
		"httpOutput": "http.go"
	}

	@Mock{
		"package": "mock",
		"output": "mock.go"
	}
*/
type ExampleInterface interface {
	/*
		@Kit{
		  "endpoints": [
		    {
		      "http": {
		        "method": "POST",
		        "path": "/some/path/{a}",
		        "successCode": 201
		      }
		    },
		    {
		      "name": "AnotherOne",
		      "http": {
		        "method": "POST",
		        "path": "/other/path/{a}",
		        "successCode": 201
		      }
		    }
		  ],
		  "httpParams": ["url", "json"]
		}
	*/
	Method1(ctx context.Context, a string, x X) (Y, error)
	/*
		@Kit{
		  "endpoints": [
		    {
		      "http": {
		        "method": "GET",
		        "path": "/some/path/{a}"
		      }
		    }
		  ],
		  "httpParams": ["url", "query"]
		}
	*/
	Method2(ctx context.Context, a string, qp QueryParams) error
}
