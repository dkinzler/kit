package parse

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: We can use relative paths here, since go test will always run package tests with the working directory set to the directory that contains the package being tested.
//The testdata dir contains an example go project that we can parse.

func TestModuleCanBeFound(t *testing.T) {
	a := assert.New(t)

	m, err := NewModuleFromDir("testdata/exampleproject/internal/example")
	a.Nil(err)
	a.Equal("exampleproject", m.Name)
	a.True(strings.HasSuffix(m.Path, "testdata/exampleproject"))

	//using a parent dir should find the same module
	m1, err := NewModuleFromDir("testdata/exampleproject/internal")
	a.Nil(err)
	m2, err := NewModuleFromDir("testdata/exampleproject")
	a.Nil(err)
	a.Equal(m, m1)
	a.Equal(m, m2)
}

func TestModuleHelperFunctions(t *testing.T) {
	a := assert.New(t)

	m, err := NewModuleFromDir("testdata/exampleproject/internal/example")
	a.Nil(err)

	fn := m.FileName("internal/example", "example.go")
	expectedFn, err := filepath.Abs("testdata/exampleproject/internal/example/example.go")
	a.Nil(err)
	a.Equal(expectedFn, fn)

	pp, err := m.PackagePathFromFilePath(expectedFn)
	a.Nil(err)
	a.Equal("exampleproject/internal/example", pp)
	//this should fail, since file is not in module directory
	outsideFile, err := filepath.Abs("testdata/otherproject/internal/example/example.go")
	a.Nil(err)
	_, err = m.PackagePathFromFilePath(outsideFile)
	a.NotNil(err)

	a.Equal("exampleproject/internal/example", m.FullPackagePath("internal/example"))
	a.Equal("internal/example", m.PackagePathWithoutModule("exampleproject/internal/example"))
}

func TestParseDir(t *testing.T) {
	a := assert.New(t)

	m, err := NewModuleFromDir("testdata/exampleproject")
	a.Nil(err)

	is, err := ParseDir("testdata/exampleproject", m)
	a.Nil(err)
	a.Len(is, 2)

	fp1, err := filepath.Abs("testdata/exampleproject/internal/example/example.go")
	a.Nil(err)
	fp2, err := filepath.Abs("testdata/exampleproject/internal/other/other.go")
	a.Nil(err)

	var exampleInterface, otherInterface Interface
	if is[0].Name == "ExampleInterface" {
		exampleInterface = is[0]
		otherInterface = is[1]
	} else {
		exampleInterface = is[1]
		otherInterface = is[0]
	}

	a.Equal("ExampleInterface", exampleInterface.Name)
	a.Equal("exampleproject/internal/example", exampleInterface.Package)
	a.Equal(fp1, exampleInterface.File)
	a.ElementsMatch(exampleInterface.Methods, []Method{
		{
			Name: "Method1",
			Params: []Param{
				{Name: "ctx", Type: SimpleType{Type: "Context", Package: "context"}},
				{Name: "a", Type: SimpleType{Type: "string", Package: ""}},
				{Name: "x", Type: SimpleType{Type: "X", Package: "exampleproject/internal/example"}},
			},
			Returns: []Param{
				{Name: "", Type: SimpleType{Type: "Y", Package: "exampleproject/internal/example"}},
				{Name: "", Type: SimpleType{Type: "error", Package: ""}},
			},
		},
		{
			Name: "Method2",
			Params: []Param{
				{Name: "m", Type: MapType{
					KeyType: SimpleType{Type: "int"},
					ValueType: ArrayType{
						Type: MapType{
							KeyType:   SimpleType{Type: "string"},
							ValueType: SimpleType{Type: "int"},
						},
					},
				}},
			},
			Returns: []Param{
				{Name: "", Type: SimpleType{Type: "error", Package: ""}},
			},
		},
		{
			Name: "Method3",
		},
	})

	a.Equal("OtherInterface", otherInterface.Name)
	a.Equal("exampleproject/internal/other", otherInterface.Package)
	a.Equal(fp2, otherInterface.File)
	a.ElementsMatch(otherInterface.Methods, []Method{
		{
			Name: "OtherMethod1",
			Params: []Param{
				{Name: "a", Type: SimpleType{Type: "string"}},
				{Name: "b", Type: SimpleType{Type: "string"}},
				{Name: "c", Type: SimpleType{Type: "string"}},
			},
			Returns: []Param{
				{Name: "", Type: SimpleType{Type: "int"}},
				{Name: "", Type: SimpleType{Type: "int"}},
				{Name: "", Type: SimpleType{Type: "int"}},
			},
		},
	})
}
