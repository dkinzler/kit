package parse

// Interface represents an interface in a more compact way than a tree from the "go/ast" package,
// which makes it easier to work with for code generation.
type Interface struct {
	// Name of the interface
	Name string
	// Package the interface is defined in
	Package string
	// Methods of the interface
	Methods []Method
	// Comments belonging to this interface, i.e. the comments directly above the type definition in the source code.
	Comments []string

	//the file (path) this interface is defined in
	File string
}

// Method represents a method of an interface.
type Method struct {
	// Name of the method
	Name string
	// Parameters/Arguments
	Params []Param
	// Return values
	Returns []Param
	// Comments belonging to this method, i.e. the comments directly above the method definition in the source code.
	Comments []string
}

// Param represents a method paramter or return value
type Param struct {
	// Name of the parameter, e.g. "ctx" for the parameter definition "ctx context.Context".
	Name string
	// Type of the parameter
	Type ParamType
}

// Represents the type of a parameter.
// Note: this does not support some types like function types, channels and anonymous structs, since
// these types should rarely appear for the kinds of interfaces this package is used on.
type ParamType interface {
	// Returns a list of all the packages required by this type.
	// E.g. a type map[context.Context]*http.Request requires the pacakges "context" and "http".
	Packages() []string
}

// Simple types are: bool, string, error, int, float (and all the variations of the numeric types), interface{}, structs like http.Request.
// They consist of just the type name and possibly a package prefix.
// Note that pointers (*http.Request), slices, maps, function types, channels, anonymous structs, ... are composite types.
type SimpleType struct {
	// Name of the type, e.g. "string" or "Request" for "http.Request"
	Type string
	// Package the type is defined in, e.g. "http" for "http.Request"
	// Empty for built-in types.
	Package string
}

func (t SimpleType) Packages() []string {
	if t.Package == "" {
		return nil
	}
	return []string{t.Package}
}

// Represents a map type recursively.
type MapType struct {
	KeyType   ParamType
	ValueType ParamType
}

func (t MapType) Packages() []string {
	return append(t.KeyType.Packages(), t.ValueType.Packages()...)
}

type ArrayType struct {
	Type ParamType
}

func (at ArrayType) Packages() []string {
	return at.Type.Packages()
}

type StarType struct {
	Type ParamType
}

func (st StarType) Packages() []string {
	return st.Type.Packages()
}

var basicTypes = []string{
	"bool",
	"string",
	"error",
	"int",
	"int8",
	"int16",
	"int32",
	"int64",
	"uint",
	"uint8",
	"uint16",
	"uint32",
	"uint64",
	"uintptr",
	"byte",
	"rune",
	"float32",
	"float64",
	"complex64",
	"complex128",
}

func isBasicType(t string) bool {
	for _, bt := range basicTypes {
		if t == bt {
			return true
		}
	}
	return false
}

func IsSimpleType(p ParamType, typeName, packageName string) bool {
	if st, ok := p.(SimpleType); ok {
		if st.Type == typeName && st.Package == packageName {
			return true
		}
		return false
	}
	return false
}
