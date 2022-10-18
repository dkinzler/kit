/*
Package codegen provides a code generator that can generate:
  - Mock implementation of an interface using [Testify Mock] (package github.com/stretchr/testify/mock)
  - [Go kit] (package github.com/go-kit/kit) endpoints for an interface
  - [Go kit] http handlers for an interface

The code generator is configured by providing annotations in the comments of an interface and its methods.
An annotation has the format @Name{"abc":"xyz"} where:
  - Name denotes the type of code to generate, either Mock or Kit
  - name is followed by a JSON object which can be split across multiple comment lines

Run the generator with:

	go run github.com/d39b/kit/codegen@latest --inputDir xyz

This will generate code for any annotated interfaces found within directory xyz or (recursively) any subdirectories.
For the code generator to work, directory xyz must be part of a go module, i.e. xyz or one of its ancestor directories must contain a go.mod file.
Note also that code can be generated only for the same module as the annotated interfaces.

# Generating Mocks

To generate a mock implementation of an interface, add a @Mock{...} annotation to the interface comments.
See the [example project] for a complete example that also contains the generated code.

Example:

	// "package" defines the package the generated mock interface will belong to, must be relative to the module.
	// E.g. if the module is "example.com/abc" and the package for the generated code should be "example.com/abc/xyz/def" use "xyz/def" as the value for "package".
	// If "package" is empty the generated code will be placed in the same package as the interface.
	//
	// "output" defines the name of the output file that will contain the generated code.
	// If empty, defaults to "mock.go".
	//
	// @Mock{"package":"xyz", "output":"mock.go"}
	type ExampleInterface interface {
		Method1(ctx context.Context, a string, b int) error
	}

# Generating Go kit endpoints and http handlers

To generate Go kit endpoints and http handlers for an interface, add a @Kit{...} annotation to the comments of an interface.
See the [example project] for a complete example that also contains the generated code.

Example annotation on an interface, for better readability only the JSON annotation is shown:

	@Kit{
	  // Package the generated endpoints will belong to, must be relative to the full module path.
	  // E.g. if the output package should be "example.com/xyz/abc/def" and the full module path
	  // is "example.com/xyz" use "abc/def" as the value.
	  // If empty or not provided, nothing will be generated.
	  "endpointPackage": "endpoint",
	  // Name of output file for endpoint code, defaults to "endpoint.gen.go".
	  "endpointOutput": "endpoints.go",
	  // Package the generated http code will belong to, relative to the full module path.
	  // If empty or not provided nothing will be generated.
	  "httpPackage": "http",
	  // Name of output file for http code, defaults to "http.gen.go".
	  "httpOutput": "http.go"
	}

Example annotation on an interface method "Method(ctx context.Context, a string, b SomeType) error"

	@Kit{
	  // An array of json objects where each one describes an endpoint to be created,
	  // i.e. it is possible to create multiple endpoints for the same interface method.
	  "endpoints": [
	    {
	      // Name of the endpoint, used for function and field names in the generated code.
	      // Optional, defaults to the name of the method. However endpoint names must be unique
	      // and hence there can be at most one endpoint per interface method that does not define an explicit name.
	      "name":"ExampleEndpoint",
	      // Configures the http handler for the endpoint.
	      "http": {
	        // http method
	        "method": "POST",
	        // Path the endpoint will be reachable at.
	        // Can contain variables, the gorilla/mux package is used to decode them.
	        "path": "/some/path/{a}",
	        // http response code on success, defaults to 200
	        "successCode": 201
	      }
	    }
	  ],
	  // Configures how each method parameter (except first) is obtained from an incoming http request.
	  // Possible values are "url", "query", "json".
	  // In the example "a" will be obtained from the request url path, and "b" from the JSON request body.
	  "httpParams": ["url", "json"]
	}

Note that http handlers can be generated only if endpoints are generated.
Furthermore, it is possible to put generated endpoints and http handlers in the same output package.

For Go kit code generation to work, the following requirements should be met by the source interface:
  - Interface methods do not contain function types, channel types or anonymous structs as parameter or return values.
  - Interface method parameters should be named, avoid using names like "r" and "w" that are e.g. commonly used in http code.
  - The source file that contains the interface should not import any types that are used in the interface definition using ".", i.e. imported without a prefix/qualifier.
  - Every interface method has a context.Context as the first parameter.
  - Every interface method has 1 or 2 return values, where the last one is always "error".

[Go kit]: https://github.com/go-kit/kit
[Testify Mock]: https://github.com/stretchr/testify
[example project]: https://github.com/d39b/kit/tree/main/codegen/example
*/
package main

import (
	"log"
	"os"
	"path/filepath"

	cli "github.com/urfave/cli/v2"
)

const version string = "0.1"

func main() {
	app := &cli.App{
		Name:    "Codegen",
		Usage:   "generates code, how wonderful",
		Version: version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "fail-on-error",
				Value:   true,
				Aliases: []string{"e"},
				Usage:   "If true code generation is aborted on first error.",
			},
			&cli.StringFlag{
				Name:  "moduleName",
				Usage: "Name of the module the input directory belongs to, e.g. github.com/user/example .",
			},
			&cli.StringFlag{
				Name:  "modulePath",
				Usage: "Path to the root directory of the module the input directory belongs to. If empty will attempt to find the module by looking for a go.mod file in the input directory and its ancestors.",
			},
			&cli.StringFlag{
				Name:        "inputDir",
				Value:       ".",
				Usage:       "Directory to search for code generator annotations.",
				DefaultText: "default: current working directory",
			},
		},
		Action: func(ctx *cli.Context) error {
			inputDir, err := filepath.Abs(ctx.String("inputDir"))
			if err != nil {
				return err
			}

			modulePath := ctx.String("modulePath")
			if modulePath != "" {
				modulePath, err = filepath.Abs(modulePath)
				if err != nil {
					return err
				}
			}

			config := GeneratorConfig{
				InputDir:    inputDir,
				ModuleName:  ctx.String("moduleName"),
				ModulePath:  modulePath,
				FailOnError: ctx.Bool("fail-on-error"),
			}
			return generate(config)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
