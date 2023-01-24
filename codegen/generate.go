package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/dkinzler/kit/codegen/annotations"
	"github.com/dkinzler/kit/codegen/gen"
	"github.com/dkinzler/kit/codegen/internal/kit"
	"github.com/dkinzler/kit/codegen/internal/mock"
	"github.com/dkinzler/kit/codegen/parse"

	"github.com/dave/jennifer/jen"
)

type GeneratorConfig struct {
	InputDir string

	ModuleName string
	ModulePath string

	//whether or not stop generating on first error or continue
	FailOnError bool
}

func generate(config GeneratorConfig) error {
	module, err := getModule(config)
	if err != nil {
		return err
	}

	is, err := parse.ParseDir(config.InputDir, module)
	if err != nil {
		return err
	}

	var generatedCode []gen.GenResult

	for _, i := range is {
		a, err := annotations.ParseInterfaceAnnotations(i)
		if err != nil {
			if config.FailOnError {
				return err
			} else {
				//move to next interface
				continue
			}
		}

		for name, annotations := range a {
			if name == "Kit" {
				files, err := generateKit(i, module, annotations)
				if err != nil {
					if config.FailOnError {
						return err
					}
				} else {
					generatedCode = append(generatedCode, files...)
				}
			} else if name == "Mock" {
				files, err := generateMock(i, module, annotations)
				if err != nil {
					if config.FailOnError {
						return err
					}
				} else {
					generatedCode = append(generatedCode, files...)
				}
			} else {
				log.Printf("unknown annotation %v on interface %v\n", name, i.Name)
			}
		}
	}

	return outputGeneratedCode(generatedCode)
}

func getModule(config GeneratorConfig) (parse.Module, error) {
	if config.ModuleName != "" && config.ModulePath != "" {
		return parse.Module{
			Path: config.ModulePath,
			Name: config.ModuleName,
		}, nil
	} else {
		log.Println("searching for go module...")
		module, err := parse.NewModuleFromDir(config.InputDir)
		if err != nil {
			return parse.Module{}, err
		}
		log.Println("found go module", module.Name, "with root dir", module.Path)
		return module, nil
	}
}

func generateKit(i parse.Interface, module parse.Module, annotations annotations.InterfaceAnnotation) ([]gen.GenResult, error) {
	spec, err := kit.SpecFromAnnotations(i, module, annotations)
	if err != nil {
		return nil, err
	}

	files, err := kit.NewKitGenerator(spec).Generate()
	return files, err
}

func generateMock(i parse.Interface, module parse.Module, annotations annotations.InterfaceAnnotation) ([]gen.GenResult, error) {
	spec, err := mock.SpecFromAnnotations(i, module, annotations)
	if err != nil {
		return nil, err
	}

	files, err := mock.NewMockGenerator(spec).Generate()
	return files, err
}

func outputGeneratedCode(c []gen.GenResult) error {
	generatedFiles := gen.MergeResults(c)

	for _, gf := range generatedFiles {
		err := saveFile(gf.File, gf.Path)
		if err != nil {
			log.Printf("could not save file %v, got error: %v\n", gf.Path, err)
		}
	}

	return nil
}

func saveFile(f *jen.File, filename string) error {
	dir := filepath.Dir(filename)
	err := makeDir(dir)
	if err != nil {
		log.Println(err)
		return err
	}
	err = f.Save(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func makeDir(d string) error {
	return os.MkdirAll(d, os.ModePerm)
}
