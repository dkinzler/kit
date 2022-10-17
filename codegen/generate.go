package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/d39b/kit/codegen/annotations"
	"github.com/d39b/kit/codegen/gen"
	"github.com/d39b/kit/codegen/internal/kit"
	"github.com/d39b/kit/codegen/internal/mock"
	"github.com/d39b/kit/codegen/parse"

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
	var module parse.Module
	var err error
	if config.ModuleName != "" && config.ModulePath != "" {
		module = parse.Module{
			Path: config.ModulePath,
			Name: config.ModuleName,
		}
	} else {
		log.Println("searching for go module...")
		module, err = parse.NewModuleFromDir(config.InputDir)
		if err != nil {
			return err
		}
		log.Println("found go module", module.Name, "with root dir", module.Path)
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
				spec, err := kit.SpecFromAnnotations(i, module, annotations)
				if err != nil {
					if config.FailOnError {
						return err
					} else {
						//move to next interface
						continue
					}
				}

				files, err := kit.NewKitGenerator(spec).Generate()
				if err != nil {
					if config.FailOnError {
						return err
					} else {
						//move to next interface
						continue
					}
				}
				generatedCode = append(generatedCode, files...)
			} else if name == "Mock" {
				spec, err := mock.SpecFromAnnotations(i, module, annotations)
				if err != nil {
					if config.FailOnError {
						return err
					} else {
						//move to next interface
						continue
					}
				}

				files, err := mock.NewMockGenerator(spec).Generate()
				if err != nil {
					if config.FailOnError {
						return err
					} else {
						//move to next interface
						continue
					}
				}
				generatedCode = append(generatedCode, files...)
			} else {
				log.Printf("unknown annotation %v on interface %v\n", name, i.Name)
			}
		}
	}

	generatedFiles := gen.MergeResults(generatedCode)

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
