package kit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/d39b/kit/tools/codegen/gen"
	"github.com/d39b/kit/tools/codegen/parse"

	"github.com/dave/jennifer/jen"
)

func (g *KitGenerator) generateHttp() gen.GenResult {
	g.g = gen.NewSimpleGenerator()

	var code *jen.Group = jen.NewFile("").Group

	for _, es := range g.Spec.Endpoints {
		if len(es.EndpointSpecs) > 0 {
			code.Add(g.generateMethodHttpDecodeFunc(es))
			code.Line()
		}
	}

	code.Add(g.generateHttpRegisterHandlersFunc())
	return gen.GenResult{
		Code:        code,
		PackagePath: g.Spec.HttpPackageFullPath,
		PackageName: g.Spec.httpPackageName(),
		Imports: map[string]string{
			kitEndpointPackage: "endpoint",
			localHttpPackage:   "t",
			kitHttpPackage:     "kithttp",
			gorillaMuxPackage:  "mux",
		},
		OutputFile: g.Spec.Module.FileName(g.Spec.HttpPackage, g.Spec.HttpOutput),
	}
}

func (g *KitGenerator) generateMethodHttpDecodeFunc(es EndpointSpecifications) jen.Code {
	//if method only takes context parameter, there is nothing to decode, instead use go-kit/kit/transport/http.NopRequestDecoder as decoder func
	m := es.Method

	if len(m.Params) <= 1 {
		return jen.Empty()
	}

	if len(m.Params)-1 != len(es.HttpParams) {
		panic(fmt.Sprintf("generateMethodHttpDecodeFunc: missing or too many http parameter annotations for method %v,", m.Name))
	}

	var stmts []jen.Code
	returnFields := make(jen.Dict)
	for i, p := range m.Params[1:] {
		hasError := false
		if i > 0 {
			hasError = true
		}
		httpParamType := es.HttpParams[i]
		if httpParamType == HttpTypeJson {
			stmts = append(stmts, g.generateHttpDecodeFuncJsonParam(p, hasError)...)
		} else if httpParamType == HttpTypeUrl {
			stmts = append(stmts, g.generateHttpDecodeFuncUrlParam(p)...)
		} else if httpParamType == HttpTypeQuery {
			stmts = append(stmts, g.generateHttpDecodeFuncQueryParam(p, hasError)...)
		}
		stmts = append(stmts, jen.Line())
		returnFields[jen.Id(es.endpointRequestTypeParamName(p.Name))] = jen.Id(p.Name)
	}
	stmts = append(stmts, jen.Return(
		jen.Qual(g.Spec.EndpointPackageFullPath, es.endpointRequestTypeName()).Values(returnFields),
		jen.Nil(),
	))

	return g.g.GenFunction(
		nil,
		es.httpDecodeFuncName(),
		jen.Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
		),
		jen.Params(
			jen.Interface(),
			jen.Error(),
		),
		stmts,
	)
}

// hasError indicates whether the "err" var has already been defined, if yes we use operator "=" instead of ":="
func (g *KitGenerator) generateHttpDecodeFuncJsonParam(p parse.Param, hasError bool) []jen.Code {
	op := ":="
	if hasError {
		op = "="
	}
	result := []jen.Code{
		jen.Var().Id(p.Name).Add(g.g.GenParamType(p.Type)),
		jen.Id("err").Op(op).Qual(localHttpPackage, "DecodeJSONBody").Call(jen.Id("r"), jen.Op("&").Id(p.Name)),
		jen.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Id("err")),
		),
	}
	return result
}

func (g *KitGenerator) generateHttpDecodeFuncUrlParam(p parse.Param) []jen.Code {
	result := []jen.Code{
		jen.List(jen.Id(p.Name), jen.Id("err")).Op(":=").Qual(localHttpPackage, "DecodeURLParameter").Call(jen.Id("r"), jen.Lit(p.Name)),
		jen.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Id("err")),
		),
	}
	return result
}

func (g *KitGenerator) generateHttpDecodeFuncQueryParam(p parse.Param, hasError bool) []jen.Code {
	op := ":="
	if hasError {
		op = "="
	}
	result := []jen.Code{
		jen.Var().Id(p.Name).Add(g.g.GenParamType(p.Type)),
		jen.Id("err").Op(op).Qual(localHttpPackage, "DecodeQueryParameters").Call(jen.Id("r"), jen.Op("&").Id(p.Name)),
		jen.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Id("err")),
		),
	}
	return result
}

type httpEndpointCodeStmts struct {
	Path  string
	Stmts []jen.Code
}

func (g *KitGenerator) generateHttpRegisterHandlersFunc() jen.Code {
	//generate code for each endpoint, we will then sort them by path afterwards
	stmts := []httpEndpointCodeStmts{}

	for _, es := range g.Spec.Endpoints {
		for _, spec := range es.EndpointSpecs {
			var decodeFuncName jen.Code
			if len(es.HttpParams) > 0 {
				decodeFuncName = jen.Id(es.httpDecodeFuncName())
			} else {
				decodeFuncName = jen.Qual(kitHttpPackage, "NopRequestDecoder")
			}
			stmts = append(stmts, httpEndpointCodeStmts{
				Path: spec.HttpSpec.Path,
				Stmts: []jen.Code{
					//TODO Fix all this, need to use endpoint name
					jen.Id(spec.httpHandlerVarName()).Op(":=").Qual(kitHttpPackage, "NewServer").Call(
						jen.Id("endpoints").Dot(spec.endpointSetFieldName()),
						decodeFuncName,
						jen.Qual(localHttpPackage, "MakeGenericJSONEncodeFunc").Call(jen.Lit(spec.HttpSpec.SuccessCode)),
						jen.Id("opts").Op("..."),
					),
					jen.Id("router").Dot("Handle").Call(
						jen.Lit(spec.HttpSpec.Path),
						jen.Id(spec.httpHandlerVarName()),
					).Dot("Methods").Call(
						jen.Lit(strings.ToUpper(spec.HttpSpec.Method)),
						jen.Lit("OPTIONS"),
					),
				},
			})
		}
	}

	sort.Slice(stmts, func(i, j int) bool {
		return sortEndpointsByHttpPath(stmts[i].Path, stmts[j].Path)
	})

	combinedStmts := []jen.Code{}
	for i, s := range stmts {
		combinedStmts = append(combinedStmts, s.Stmts...)
		if i < len(stmts)-1 {
			combinedStmts = append(combinedStmts, jen.Line())
		}
	}

	return g.g.GenFunction(
		nil,
		"RegisterHttpHandlers",
		jen.Params(
			jen.Id("endpoints").Qual(g.Spec.EndpointPackageFullPath, "EndpointSet"),
			jen.Id("router").Op("*").Qual(gorillaMuxPackage, "Router"),
			jen.Id("opts").Index().Qual(kitHttpPackage, "ServerOption"),
		),
		jen.Empty(),
		combinedStmts,
	)
}

func sortEndpointsByHttpPath(a, b string) bool {
	path1 := strings.TrimSpace(a)
	path1 = strings.TrimPrefix(path1, "/")
	path1 = strings.TrimSuffix(path1, "/")

	path2 := strings.TrimSpace(b)
	path2 = strings.TrimPrefix(path2, "/")
	path2 = strings.TrimSuffix(path2, "/")

	p1 := strings.Split(path1, "/")
	p2 := strings.Split(path2, "/")

	//minimum of len(p1) and len(p2)
	l := len(p1)
	if len(p2) < l {
		l = len(p2)
	}

	//we split the paths into the segments and decide based on the first segment that is different
	//if all the segments are equal, the shorter path comes first
	for i := 0; i < l; i++ {
		f1 := p1[i]
		f2 := p2[i]
		if f1 != f2 {
			if strings.HasPrefix(f1, "{") {
				return false
			} else if strings.HasPrefix(f2, "{") {
				return true
			} else if f1 < f2 {
				return true
			} else {
				return false
			}
		}
	}

	if len(p1) < len(p2) {
		return true
	}
	return false
}
