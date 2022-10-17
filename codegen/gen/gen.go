// Package gen implements code generation functionality that work on interface specifications from the parse package.
// Code is generated using the package "github.com/dave/jennifer/jen".
package gen

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/d39b/kit/codegen/parse"

	"github.com/dave/jennifer/jen"
)

// A piece of code returned by a code generator, that is assigned to a particular output package/file.
// Multiple instances of GenResult can be merged into a single code file, since different code generators might provide parts of it.
type GenResult struct {
	// Piece of code, that does not include any package or import statements.
	Code *jen.Group
	// Full path of package this code belongs to, e.g. "example.com/abc/xyz".
	PackagePath string
	// Package name of package this code belongs to, usually the last section of the package path, e.g. "xyz".
	PackageName string
	// Define explicit aliases for imports used by this code.
	// Maps from package path to alias, e.g. "example.com/abc/xyz":"x"
	Imports map[string]string
	// Path of the file the code should ultimately be written to.
	OutputFile string
}

// The interface all code generators should implement.
// A code generator should be created with a specification that defines what exactly needs to be generated.
// Calling the Generate() method should then always yield the same result.
//
// A code generator should not actually write any files, instead the Generate() method should return a list of
// GenResult values that each define a piece of code belonging to an output file.
type Generator interface {
	Generate() ([]GenResult, error)
}

type GeneratedFile struct {
	File *jen.File
	Path string
}

// Generates a list of GeneratedFile values by merging together all the code pieces for the same output file path into a single code file.
func MergeResults(results []GenResult) []GeneratedFile {
	resultsByFile := make(map[string][]GenResult)
	for _, result := range results {
		resultsByFile[result.OutputFile] = append(resultsByFile[result.OutputFile], result)
	}

	result := make([]GeneratedFile, len(resultsByFile))
	i := 0
	for outputFile, r := range resultsByFile {
		rr := r[0]
		f := jen.NewFilePathName(rr.PackagePath, rr.PackageName)
		f.PackageComment("generated code, do not modify")
		for path, alias := range mergeImports(r) {
			f.ImportAlias(path, alias)
		}
		for _, part := range r {
			f.Add(part.Code)
		}
		result[i] = GeneratedFile{File: f, Path: outputFile}
		i++
	}
	return result
}

func mergeImports(r []GenResult) map[string]string {
	result := make(map[string]string)
	for _, x := range r {
		for path, name := range x.Imports {
			result[path] = name
		}
	}
	return result
}

// SimpleGenerator provides functions to make it easier to generate common code elements like parameters, struct types and functions.
type SimpleGenerator struct{}

func NewSimpleGenerator() *SimpleGenerator {
	return &SimpleGenerator{}
}

// Generates a type e.g. for a function parameter or return value.
func (s *SimpleGenerator) GenParamType(p parse.ParamType) jen.Code {
	switch t := p.(type) {
	case parse.SimpleType:
		if t.Package == "" {
			return jen.Id(t.Type)
		} else {
			return jen.Qual(t.Package, t.Type)
		}
	case parse.MapType:
		return jen.Map(s.GenParamType(t.KeyType)).Add(s.GenParamType(t.ValueType))
	case parse.ArrayType:
		return jen.Index().Add(s.GenParamType(t.Type))
	case parse.StarType:
		return jen.Op("*").Add(s.GenParamType(t.Type))
	default:
		panic("unimplemented parse.ParamType in GenParamType()")
	}
}

// Generates a struct type with the given name and fields.
func (s *SimpleGenerator) GenStructType(name string, fields []jen.Code) jen.Code {
	return jen.Type().Id(name).Struct(fields...)
}

// Generates a function with the given receiver, name, parameters, return values and statements in the body.
func (s *SimpleGenerator) GenFunction(receiver jen.Code, name string, params jen.Code, returns jen.Code, body []jen.Code) jen.Code {
	result := jen.Func()
	if receiver != nil {
		result = result.Params(receiver)
	}
	if name != "" {
		result = result.Id(name)
	}
	result = result.Add(params)
	result = result.Add(returns)
	result.Block(body...)
	return result
}

func (s *SimpleGenerator) GenReturnParams(params []parse.Param) jen.Code {
	var returnParams []jen.Code
	for _, param := range params {
		paramType := s.GenParamType(param.Type)
		returnParams = append(returnParams, paramType)
	}
	// if there is only a single return value, we do not need to wrap it in "()"
	if len(returnParams) == 1 {
		return returnParams[0]
	} else if len(returnParams) > 1 {
		return jen.Params(returnParams...)
	} else {
		return jen.Empty()
	}
}

func (s *SimpleGenerator) GenParamTypes(params []parse.Param) []jen.Code {
	result := make([]jen.Code, len(params))
	for i, param := range params {
		result[i] = s.GenParamType(param.Type)
	}
	return result
}

// Generatesa parameter list for the given parameter specification.
// E.g. (ctx context.Context, p1 string, p2 int)
// If parameter specifications do not contain a name, consecutive integers will be used, e.g. "p0", "p1", "p2".
func (s *SimpleGenerator) GenFunctionParams(params []parse.Param) jen.Code {
	funcParams := make([]jen.Code, len(params))
	paramNames := s.GenParamNames(params)

	for i, param := range params {
		paramName := paramNames[i]
		paramType := s.GenParamType(param.Type)
		funcParams[i] = jen.Id(paramName).Add(paramType)
	}

	return jen.Params(funcParams...)
}

// Returns a list of parameter names for the given parameter specification.
// If a parameter specification contains a name, that name will be used, otherwise a name is generated
// by using consecutive integers, i.e. "p0", "p1", "p2".
// The underscore in the generated param names is used to avoid any naming conflicts with existing code elements.
func (s *SimpleGenerator) GenParamNames(params []parse.Param) []string {
	result := make([]string, len(params))

	id := 0
	for i, param := range params {
		if param.Name == "" {
			result[i] = fmt.Sprintf("p%v", id)
			id++
		} else {
			result[i] = param.Name
		}
	}

	return result
}

// Lowercase first letter of string, useful for generating e.g. paramter or variable names.
func LowercaseFirst(s string) string {
	for i, char := range s {
		return string(unicode.ToLower(char)) + s[i+1:]
	}
	return ""
}

// Uppercase first letter of string, userful for generating e.g. exported struct or function names.
func UppercaseFirst(s string) string {
	return strings.Title(s)
}

// Adds the following package comment to the given code file:
// "generated code, do not edit"
func AddDefaultPackageComment(f *jen.File) {
	f.PackageComment("generated code, do not edit")
}
