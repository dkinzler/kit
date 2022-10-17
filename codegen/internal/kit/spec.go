package kit

import (
	"errors"
	"fmt"
	"path"

	"github.com/d39b/kit/codegen/annotations"
	"github.com/d39b/kit/codegen/gen"
	"github.com/d39b/kit/codegen/parse"
)

// KitGenSpecification defines what the code generator should generate.
// It is typically constructed by reading source code annotations on e.g. an interface or method
// and configuration from the command line or a configuration file.
type KitGenSpecification struct {
	//the interface for which code should be generated
	Interface parse.Interface
	//go module the interface belongs to
	Module parse.Module

	// if false, nothing will be generated
	GenerateEndpoints bool
	// if false, will not generate http handlers for endpoints
	GenerateHttp bool

	// package name used for generated endpoints
	// can be a full package path or relative to the module name
	// If empty will not generate endpoints.
	EndpointPackage string `json:"endpointPackage"`
	// full path of the package that will contain the generated endpoint code
	// we need this to later import this package to e.g. generate http handlers
	EndpointPackageFullPath string
	// output file for endpoint code
	EndpointOutput string `json:"endpointOutput"`
	// Package name used for generated http handlers.
	// If empty will not generate anything.
	HttpPackage         string `json:"httpPackage"`
	HttpPackageFullPath string
	// output file for http code
	HttpOutput string `json:"httpOutput"`

	// each element specifies the endpoints to generate for an interface method
	Endpoints []EndpointSpecifications
}

func (g KitGenSpecification) endpointPackageName() string {
	return path.Base(g.EndpointPackage)
}

func (g KitGenSpecification) httpPackageName() string {
	return path.Base(g.HttpPackage)
}

// Checks if a given specification is valid.
// A specification is not valid if one of the following conditions is not satisfied:
//   - there cannot be two endpoints with the same name
//   - any interface method (for which at least one endpoint is defined) must have a context.Context value as first parameter
//   - any interface method (for which at least one endpoint is defined) must have at most two return values and last return value must be of type error
//   - if http code is generated, endpoints should have http method, path and success code set
func (spec KitGenSpecification) IsValid() error {
	// check that there are no duplicate endpoint names
	names := make(map[string]bool)
	for _, es := range spec.Endpoints {
		for _, e := range es.EndpointSpecs {
			if _, ok := names[e.Name]; ok {
				return errors.New(fmt.Sprintf("kit specification contains duplicate endpoint name: %v", e.Name))
			}
			names[e.Name] = true
		}
	}

	// check that interface methods have context.Context as first parameter
	for _, e := range spec.Endpoints {
		m := e.Method
		if len(m.Params) == 0 {
			return errors.New(fmt.Sprintf("interface method %v doesn't have context.Context parameter", m.Name))
		} else {
			firstParam := m.Params[0]
			if !parse.IsSimpleType(firstParam.Type, "Context", "context") {
				return errors.New(fmt.Sprintf("interface method %v doesn't have context.Context as first parameter", m.Name))
			}
		}
	}

	// check that interface methods have either 1 or 2 return values and last one is error
	for _, e := range spec.Endpoints {
		m := e.Method
		if len(m.Returns) < 1 || len(m.Returns) > 2 {
			return errors.New(fmt.Sprintf("interface method %v has invalid amount of return values", m.Name))
		}
		lastReturnValue := m.Returns[len(m.Returns)-1]
		if !parse.IsSimpleType(lastReturnValue.Type, "error", "") {
			return errors.New(fmt.Sprintf("interface method %v does not have error as last return value", m.Name))
		}
	}

	// If GenerateHttp is true, every endpoint should have a http method, path and path set,
	// and success code should be set to a default value.
	if spec.GenerateHttp {
		for _, e := range spec.Endpoints {
			for _, es := range e.EndpointSpecs {
				if es.HttpSpec.Method == "" {
					return errors.New(fmt.Sprintf("http method is empty for endpoint %v", es.Name))
				}
				if es.HttpSpec.Path == "" {
					return errors.New(fmt.Sprintf("http path is empty for endpoint %v", es.Name))
				}
				if es.HttpSpec.SuccessCode == 0 {
					return errors.New(fmt.Sprintf("success code not set for endpoint %v", es.Name))
				}
			}
		}
	}

	return nil
}

// Defines the endpoints created for a single interface method.
// We can generate multiple endpoints for the same interface method, e.g. to use different permission/authentication middlewares or different http urls.
type EndpointSpecifications struct {
	// the interface method to generate endpoints for
	Method parse.Method

	EndpointSpecs []EndpointSpecification `json:"endpoints"`

	// How the values from the interface method are obtained from an http request,
	// only used if http handlers are generated.
	// Since we expect the first parameter of an interface method to be a "context.Context", the length of this slice
	// should equal len(Method.Params) - 1.
	// TODO we could let every endpoint for this method define their own http params, which would result in multiple http decode funcs, but this is not necessary for now
	HttpParams []HttpParamType `json:"httpParams"`
}

func (e EndpointSpecifications) endpointRequestTypeName() string {
	return gen.UppercaseFirst(e.Method.Name) + "Request"
}

func (e EndpointSpecifications) endpointRequestTypeParamName(paramName string) string {
	return gen.UppercaseFirst(paramName)
}

func (e EndpointSpecifications) httpDecodeFuncName() string {
	return "decodeHttp" + gen.UppercaseFirst(e.Method.Name) + "Request"
}

// Specification to create a single endpoint for an inteface method.
type EndpointSpecification struct {
	// name of this endpoint, defaults to the name of the method
	Name string `json:"name"`

	// specifies how the http handler for this endpoint is generated
	HttpSpec HttpSpec `json:"http"`
}

// the name used for the function that creates the endpoint.Endpoint
func (e EndpointSpecification) makeEndpointFuncName() string {
	return "Make" + gen.UppercaseFirst(e.Name) + "Endpoint"
}

func (e EndpointSpecification) endpointSetFieldName() string {
	return gen.UppercaseFirst(e.Name) + "Endpoint"
}

func (e EndpointSpecification) endpointVarName() string {
	return gen.LowercaseFirst(e.Name) + "Endpoint"
}

func (e EndpointSpecification) httpHandlerVarName() string {
	return gen.LowercaseFirst(e.Name) + "Handler"
}

// Defines how a http handler is generated for an endpoint.
type HttpSpec struct {
	// url
	Path string `json:"path"`
	// http method, e.g. "GET", "POST", ...
	Method string `json:"method"`
	// http code for the response on success
	SuccessCode int `json:"successCode"`
	// how the values from the interface method are obtained from the http request
}

// HttpParamType represents how the parameters of an interface method should be obtained from a http request.
// E.g. by parsing the request body as json or extracting the parameter from the url path or query parameters.
type HttpParamType string

const HttpTypeJson HttpParamType = "json"
const HttpTypeUrl HttpParamType = "url"
const HttpTypeQuery HttpParamType = "query"

func SpecFromAnnotations(i parse.Interface, m parse.Module, a annotations.InterfaceAnnotation) (KitGenSpecification, error) {
	var spec KitGenSpecification

	err := annotations.ParseJSONAnnotation(a.Annotation, &spec)
	if err != nil {
		return spec, errors.New(fmt.Sprintf("could not parse interface annotation for interface %v, error: %v", i.Name, err))
	}

	spec.Interface = i
	spec.Module = m

	var endpointsForMethods []EndpointSpecifications

	for j, m := range i.Methods {
		if a.MethodAnnotations[j] != "" {
			var es EndpointSpecifications
			err := annotations.ParseJSONAnnotation(a.MethodAnnotations[j], &es)
			if err != nil {
				return spec, errors.New(fmt.Sprintf("could not parse method annotation for method %v in interface %v, error: %v", m.Name, i.Name, err))
			}
			es.Method = m
			for k, endpoint := range es.EndpointSpecs {
				//set default endpoint name if empty
				if endpoint.Name == "" {
					endpoint.Name = m.Name
				}
				//set default http success code
				if endpoint.HttpSpec.SuccessCode == 0 {
					endpoint.HttpSpec.SuccessCode = 200
				}
				es.EndpointSpecs[k] = endpoint
			}
			endpointsForMethods = append(endpointsForMethods, es)
		}
	}

	spec.Endpoints = endpointsForMethods

	if spec.EndpointPackage != "" {
		spec.GenerateEndpoints = true
		spec.EndpointPackageFullPath = m.FullPackagePath(spec.EndpointPackage)
	}
	if spec.EndpointOutput == "" {
		spec.EndpointOutput = "endpoint.gen.go"
	}

	if spec.HttpPackage != "" {
		spec.HttpPackageFullPath = m.FullPackagePath(spec.HttpPackage)
		if spec.GenerateEndpoints {
			spec.GenerateHttp = true
		}
	}
	if spec.HttpOutput == "" {
		spec.HttpOutput = "http.gen.go"
	}

	err = spec.IsValid()
	if err != nil {
		return spec, err
	}

	return spec, nil
}
