// Package mock provides a code generator to generate mocks using the "github.com/stretchr/testify/mock" package.
package mock

import (
	"errors"
	"fmt"

	"github.com/d39b/kit/tools/codegen/gen"
	"github.com/d39b/kit/tools/codegen/parse"

	"github.com/dave/jennifer/jen"
)

const testifyMockPackage = "github.com/stretchr/testify/mock"

type MockGenerator struct {
	Spec GenSpecification
	g    *gen.SimpleGenerator
}

func NewMockGenerator(spec GenSpecification) *MockGenerator {
	return &MockGenerator{
		Spec: spec,
	}
}

func (m *MockGenerator) Generate() (result []gen.GenResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = errors.New(fmt.Sprint(r))
		}
	}()

	code := m.genInterfaceMock(m.Spec.I)
	packagePath := m.Spec.Module.FullPackagePath(m.Spec.Package)
	packageName := m.Spec.PackageName()
	outputFile := m.Spec.Module.FileName(m.Spec.Package, m.Spec.Output)
	return []gen.GenResult{{
		Code:        code,
		PackagePath: packagePath,
		PackageName: packageName,
		OutputFile:  outputFile,
	}}, nil
}

func (m *MockGenerator) genInterfaceMock(i parse.Interface) *jen.Group {
	m.g = gen.NewSimpleGenerator()

	structType := m.g.GenStructType(mockStructName(i.Name), []jen.Code{jen.Qual(testifyMockPackage, "Mock")})

	var g *jen.Group = jen.NewFile("").Group

	g.Add(structType)
	g.Line()
	for _, method := range i.Methods {
		m.genMockFunc(g, method, i)
		g.Line()
	}

	return g
}

func mockStructName(name string) string {
	return "Mock" + gen.UppercaseFirst(name)
}

func (m *MockGenerator) genMockFunc(g *jen.Group, method parse.Method, i parse.Interface) {
	funcParams := m.g.GenFunctionParams(method.Params)
	returnParams := m.g.GenReturnParams(method.Returns)

	paramNames := m.g.GenParamNames(method.Params)
	//the call to m.Called(arg1, arg2), should not contain the first argument if it is context.Context
	if len(method.Params) > 0 {
		if st, ok := method.Params[0].Type.(parse.SimpleType); ok && st.Type == "Context" && st.Package == "context" {
			paramNames = paramNames[1:]
		}
	}
	paramIds := make([]jen.Code, len(paramNames))
	for i, paramName := range paramNames {
		paramIds[i] = jen.Id(paramName)
	}

	var stmts []jen.Code
	stmts = append(stmts, jen.Id("args").Op(":=").Id("m").Dot("Called").Call(paramIds...))
	if len(method.Returns) > 0 {
		returnTypes := m.g.GenParamTypes(method.Returns)
		returnElements := make([]jen.Code, len(returnTypes))
		for i, rt := range returnTypes {
			if isErrorParam(method.Returns[i]) {
				returnElements[i] = jen.Id("args").Dot("Error").Call(jen.Lit(i))
			} else {
				returnElements[i] = jen.Id("args").Dot("Get").Call(jen.Lit(i)).Assert(rt)
			}
		}
		stmts = append(stmts, jen.Return(returnElements...))
	}

	g.Add(m.g.GenFunction(
		jen.Id("m").Op("*").Id(mockStructName(i.Name)),
		method.Name,
		funcParams,
		returnParams,
		stmts,
	))
}

func isErrorParam(param parse.Param) bool {
	st, ok := (param.Type).(parse.SimpleType)
	if ok {
		if st.Type == "error" {
			return true
		}
		return false
	}
	return false
}
