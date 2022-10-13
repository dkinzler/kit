package mock

import (
	"errors"
	"fmt"
	"path"

	"github.com/d39b/kit/tools/codegen/annotations"
	"github.com/d39b/kit/tools/codegen/parse"
)

// GenSpecification defines the input data and configuration for MockGenerator.
type GenSpecification struct {
	// the interface to generate a mock for
	I parse.Interface
	// module the interface belongs to, can be used to determine package names and file paths
	Module parse.Module
	// Package the file will be created in.
	// Path must be relative to the module, e.g. if the module is "example.com/abc" and the package is "example.com/abc/xyz/def" use "xyz/def".
	// If empty use the same package as the interface.
	Package string `json:"package"`
	// Filename of the output, defaults to "mock.go".
	Output string `json:"output"`
}

// The package name to use in a source file, the last element of the full package path.
// E.g. for the package "example.com/abc/xyz" the package name would be "xyz".
func (g GenSpecification) PackageName() string {
	return path.Base(g.Module.FullPackagePath(g.Package))
}

func SpecFromAnnotations(i parse.Interface, m parse.Module, a annotations.InterfaceAnnotation) (GenSpecification, error) {
	var spec GenSpecification

	err := annotations.ParseJSONAnnotation(a.Annotation, &spec)
	if err != nil {
		return spec, errors.New(fmt.Sprintf("could not parse annotation for interface %v, error: %v", i.Name, err))
	}

	spec.I = i
	spec.Module = m

	if spec.Package == "" {
		spec.Package = m.PackagePathWithoutModule(i.Package)
	}
	if spec.Output == "" {
		spec.Output = "mock.go"
	}

	return spec, nil
}
