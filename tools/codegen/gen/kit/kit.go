// Package kit provides a code generator to generate endpoints and http handlers for an interface.
// For the code generator to work, the following requirements should be met:
//   - Interface methods do not contain function types, channel types or anonymous structs as parameter or return values.
//   - The source file that contains the interface should not import any types that are used in the interface definition using ".", i.e. (import them without a prefix/qualifier).
//   - Every interface method has a context.Context as the first parameter.
//   - Every interface method has 1 or 2 return values, where the last one is always "error".
package kit

import (
	"errors"
	"fmt"

	"github.com/d39b/kit/tools/codegen/gen"
)

const kitEndpointPackage = "github.com/go-kit/kit/endpoint"
const kitHttpPackage = "github.com/go-kit/kit/transport/http"
const localEndpointPackage = "go-sample/internal/pkg/endpoint"
const localHttpPackage = "go-sample/internal/pkg/transport/http"
const gorillaMuxPackage = "github.com/gorilla/mux"

type KitGenerator struct {
	Spec      KitGenSpecification
	g         *gen.SimpleGenerator
	nextVarId int
}

func NewKitGenerator(spec KitGenSpecification) *KitGenerator {
	return &KitGenerator{
		Spec: spec,
		g:    gen.NewSimpleGenerator(),
	}
}

func (g *KitGenerator) Generate() (result []gen.GenResult, err error) {
	//Instead of having an error return value on every function in the chain, just catch any panics and set the error
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = errors.New(fmt.Sprint(r))
		}
	}()

	// nothing to generate, can't generate http handlers without endpoints
	if !g.Spec.GenerateEndpoints {
		return nil, nil
	}

	endpoints := g.generateEndpoints()
	result = append(result, endpoints)

	if g.Spec.GenerateHttp {
		http := g.generateHttp()
		result = append(result, http)
	}
	return result, nil
}
