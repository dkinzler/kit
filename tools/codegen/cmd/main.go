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
