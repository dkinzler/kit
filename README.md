# Kit

![GitHub Workflow Status](https://github.com/d39b/kit/actions/workflows/go.yml/badge.svg)
[![GoDev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/d39b/kit)
[![Go Report Card](https://goreportcard.com/badge/github.com/d39b/kit)](https://goreportcard.com/report/github.com/d39b/kit)

Packages and tools for building Go applications. 
While some of the packages provide stand-alone functionality, others build on top of existing packages to make them easier to use.

- Package [errors](https://pkg.go.dev/github.com/d39b/kit/errors) can be used to create and inspect structured errors that can hold error properties and context.
- Package [firebase](https://pkg.go.dev/github.com/d39b/kit/firebase) makes it easier to work with the [Firebase Admin SDK](https://pkg.go.dev/firebase.google.com/go/v4).
- Package [emulator](https://pkg.go.dev/github.com/d39b/kit/firebase/emulator) provides helpers to use Firebase emulators for testing.
- Package [endpoint](https://pkg.go.dev/github.com/d39b/kit/endpoint) and [http](https://pkg.go.dev/github.com/d39b/kit/transport/http) provide functionality on top of [Go kit](https://github.com/go-kit/kit) and the [Gorilla Web Toolkit](https://github.com/gorilla) to build services and APIs using JSON over HTTP. For convenience, package [codegen](https://pkg.go.dev/github.com/d39b/kit/codegen) provides a code generator that can generate all the necessary Go kit endpoint and transport boilerplate code.
- ...

For the complete list of packages, more detailed information and usage examples refer to the [documentation](https://pkg.go.dev/github.com/d39b/kit).


## License

[MIT](LICENSE)
