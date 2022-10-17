package kit

import (
	"fmt"

	"github.com/d39b/kit/codegen/gen"

	"github.com/dave/jennifer/jen"
)

func (g *KitGenerator) generateEndpoints() gen.GenResult {
	g.g = gen.NewSimpleGenerator()

	var code *jen.Group = jen.NewFile("").Group

	for _, es := range g.Spec.Endpoints {
		if len(es.EndpointSpecs) > 0 {
			code.Add(g.generateMethodEndpointRequestType(es))
			code.Line()
			for _, ess := range es.EndpointSpecs {
				code.Add(g.generateMethodEndpointMakeFunc(es, ess))
				code.Add()
			}
		}
	}

	code.Add(g.generateEndpointSetStruct())
	code.Line()
	code.Add(g.generateEndpointMiddlewaresStruct())
	code.Line()
	code.Add(g.generateNewEndpointsFunc())
	return gen.GenResult{
		Code:        code,
		PackagePath: g.Spec.EndpointPackageFullPath,
		PackageName: g.Spec.endpointPackageName(),
		Imports: map[string]string{
			kitEndpointPackage:   "endpoint",
			localEndpointPackage: "e",
		},
		OutputFile: g.Spec.Module.FileName(g.Spec.EndpointPackage, g.Spec.EndpointOutput),
	}
}

func (g *KitGenerator) generateMethodEndpointRequestType(es EndpointSpecifications) jen.Code {
	method := es.Method

	// first parameter should always be context.Context
	// if there are no other parameters, nothing needs to be generated
	if len(method.Params) <= 1 {
		return jen.Empty()
	}

	typeName := es.endpointRequestTypeName()
	var fields []jen.Code
	//first parameter is context, can be skipped
	for _, param := range method.Params[1:] {
		paramName := es.endpointRequestTypeParamName(param.Name)
		fields = append(fields, jen.Id(paramName).Add(g.g.GenParamType(param.Type)))
	}
	return g.g.GenStructType(typeName, fields)
}

func (g *KitGenerator) generateMethodEndpointMakeFunc(es EndpointSpecifications, spec EndpointSpecification) jen.Code {
	svcMethodCall, hasResult := g.generateInterfaceMethodCall(es)

	var resultValue jen.Code = jen.Nil()
	if hasResult {
		resultValue = jen.Id("r")
	}

	returnStmt := jen.Return(jen.Qual(localEndpointPackage, "Response").Values(
		jen.Dict{
			jen.Id("R"):   resultValue,
			jen.Id("Err"): jen.Id("err"),
		},
	), jen.Nil())

	m := es.Method

	var stmts []jen.Code
	//if method only has a context parameter we don't need a req variable
	if len(m.Params) > 1 {
		stmts = append(stmts, jen.Id("req").Op(":=").Id("request").Assert(jen.Id(es.endpointRequestTypeName())))
	}
	stmts = append(stmts, svcMethodCall, returnStmt)

	return g.g.GenFunction(
		nil,
		spec.makeEndpointFuncName(),
		jen.Params(jen.Id("svc").Qual(g.Spec.Interface.Package, g.Spec.Interface.Name)),
		jen.Qual(kitEndpointPackage, "Endpoint"),
		[]jen.Code{
			jen.Return(g.g.GenFunction(
				nil,
				"",
				jen.Params(
					jen.Id("ctx").Qual("context", "Context"),
					jen.Id("request").Interface(),
				),
				jen.Params(
					jen.Interface(),
					jen.Error(),
				),
				stmts,
			)),
		},
	)
}

// second parameter indicates whether there was just a single error return value (false) or two return values (true), second return value is still error
func (g *KitGenerator) generateInterfaceMethodCall(es EndpointSpecifications) (jen.Code, bool) {
	var result jen.Code
	m := es.Method

	params := []jen.Code{
		jen.Id("ctx"),
	}
	//ignore context parameter
	for _, p := range m.Params[1:] {
		params = append(params, jen.Id("req").Dot(es.endpointRequestTypeParamName(p.Name)))
	}

	if len(m.Returns) == 1 {
		//this should be error
		result = jen.Id("err").Op(":=").Id("svc").Dot(m.Name).Call(params...)
		return result, false
	} else if len(m.Returns) == 2 {
		result = jen.List(jen.Id("r"), jen.Id("err")).Op(":=").Id("svc").Dot(m.Name).Call(params...)
		return result, true
	} else {
		panic(fmt.Sprintf("method %v does not have 1 or 2 return values", m.Name))
	}
}

func (g *KitGenerator) generateEndpointSetStruct() jen.Code {
	var fields []jen.Code
	for _, es := range g.Spec.Endpoints {
		for _, ess := range es.EndpointSpecs {
			paramName := ess.endpointSetFieldName()
			fields = append(fields, jen.Id(paramName).Qual(kitEndpointPackage, "Endpoint"))
		}
	}
	return jen.Type().Id("EndpointSet").Struct(fields...)
}

func (g *KitGenerator) generateEndpointMiddlewaresStruct() jen.Code {
	var fields []jen.Code
	for _, es := range g.Spec.Endpoints {
		for _, ess := range es.EndpointSpecs {
			paramName := ess.endpointSetFieldName()
			fields = append(fields, jen.Id(paramName).Index().Qual(kitEndpointPackage, "Middleware"))
		}
	}
	return jen.Type().Id("Middlewares").Struct(fields...)
}

func (g *KitGenerator) generateNewEndpointsFunc() jen.Code {
	resultFields := make(jen.Dict)
	for _, es := range g.Spec.Endpoints {
		for _, ess := range es.EndpointSpecs {
			resultFields[jen.Id(ess.endpointSetFieldName())] = jen.Id(ess.endpointVarName())
		}
	}
	returnStmt := jen.Return(jen.Id("EndpointSet").Values(
		resultFields,
	))

	var stmts []jen.Code
	for _, es := range g.Spec.Endpoints {
		for _, ess := range es.EndpointSpecs {
			endpointVar := ess.endpointVarName()
			stmts = append(
				stmts,
				jen.Var().Id(endpointVar).Qual(kitEndpointPackage, "Endpoint"),
				jen.Block(
					jen.Id(endpointVar).Op("=").Id(ess.makeEndpointFuncName()).Call(jen.Id("svc")),
					jen.Id(endpointVar).Op("=").Qual(localEndpointPackage, "ApplyMiddlewares").Call(jen.Id(endpointVar), jen.Id("mws").Dot(ess.endpointSetFieldName()).Op("...")),
				),
				jen.Line(),
			)
		}
	}

	stmts = append(stmts, returnStmt)

	return jen.Func().Id("NewEndpoints").Params(
		jen.Id("svc").Qual(g.Spec.Interface.Package, g.Spec.Interface.Name),
		jen.Id("mws").Id("Middlewares"),
	).Id("EndpointSet").Block(
		stmts...,
	)
}
