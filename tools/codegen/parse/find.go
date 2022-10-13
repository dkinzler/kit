package parse

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"strings"
)

// Returns all the interfaces in the file.
func findInterfacesInFile(file *ast.File, packagePath string) (result []Interface, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()
	visitor := &Visitor{
		PackagePath: packagePath,
		Imports:     importsFromFile(file),
	}
	ast.Walk(visitor, file)
	return visitor.Interfaces, nil
}

// Visitor implements the ast.Visitor interface.
// It can be used with the ast.Walk function to traverse an AST and find all the interfaces.
type Visitor struct {
	// Result, i.e. the list of interfaces found so var in the current walk
	Interfaces []Interface
	// Full package path for the current file
	PackagePath string
	// map local package name to full package path, e.g. for
	//
	// import (
	//   p "some/random/package"
	//   "another/pkg"
	// )
	//
	// the map would contain p -> "some/random/package" and pkg -> "another/pkg"
	Imports map[string]string
}

// TODO a package might be imported with the short ".", i.e the file uses the exported identifiers from that package without a qualifier.
// This is not supported and might cause problems later on, since we won't know if a type was declared locally or in the imported package.
// TODO imports with versions, e.g. "abc/xyz/v2" might also cause problems, should use an alias.
func importsFromFile(f *ast.File) map[string]string {
	result := make(map[string]string)
	for _, imp := range f.Imports {
		var importPath = imp.Path.Value
		//remove leading and trailing "
		importPath = strings.Trim(importPath, "\"")

		//this will be "p" for "import p "some/package""
		//this is nil e.g. if no local name is specified e.g. "import "context"" or "import "some/sub/package""
		var importName string
		if imp.Name != nil {
			importName = imp.Name.Name
		} else {
			//by default the import qualifier/name is the last part of the import path
			importName = path.Base(importPath)
		}

		result[importName] = importPath
	}
	return result
}

// Checks if the current node represents an interface type definition.
// If so, attempts to parse the ast and create an instance of Interface.
func (v *Visitor) Visit(node ast.Node) ast.Visitor {
	gd, ok := node.(*ast.GenDecl)
	if !ok {
		return v
	}
	//check if a type is declared
	if gd.Tok != token.TYPE {
		return v
	}
	if len(gd.Specs) == 0 {
		return v
	}
	ts, ok := gd.Specs[0].(*ast.TypeSpec)
	if !ok {
		return v
	}
	//need interface type
	it, ok := ts.Type.(*ast.InterfaceType)
	if !ok {
		return v
	}
	//skip empty interfaces
	if it.Methods == nil || len(it.Methods.List) == 0 {
		return v
	}

	//found an interface type
	result := Interface{
		Name:     ts.Name.Name,
		Package:  v.PackagePath,
		Comments: v.parseComments(gd.Doc),
		Methods:  v.parseMethods(it),
	}
	v.Interfaces = append(v.Interfaces, result)

	return v
}

func (v *Visitor) parseComments(cg *ast.CommentGroup) []string {
	if cg == nil || len(cg.List) == 0 {
		return nil
	}
	var result []string
	for _, c := range cg.List {
		result = append(result, trimComment(c.Text)...)
	}
	return result
}

// Trims any leading/trailing white space and the comment characters, i.e. leading // or /* and trailing */
// Also splits multi line comments into multiple strings, one for each line
// We do not trim anything else, note that // and /* might also appear as part of the comment, i.e. not as the first characters.
func trimComment(comment string) []string {
	//remove leading and trailing white space
	comment = strings.TrimSpace(comment)
	comment = strings.TrimRight(comment, "\n")

	if strings.HasPrefix(comment, "//") {
		//this is a single line comment, only need to trim the prefix
		return []string{strings.TrimPrefix(comment, "//")}
	} else if strings.HasPrefix(comment, "/*") {
		//this is a multi line comment, trim the prefix and suffix
		result := strings.TrimPrefix(comment, "/*")
		result = strings.TrimSuffix(result, "*/")
		//remove trailing newline
		result = strings.TrimRight(result, "\n")
		return strings.Split(result, "\n")
	} else {
		return []string{comment}
	}
}

func (v *Visitor) parseMethods(it *ast.InterfaceType) []Method {
	if it.Methods == nil || len(it.Methods.List) == 0 {
		return nil
	}

	var methods []Method
	for _, m := range it.Methods.List {
		method, ok := v.parseMethod(m)
		if ok {
			methods = append(methods, method)
		}
	}
	return methods
}

func (v Visitor) parseMethod(method *ast.Field) (Method, bool) {
	var result Method
	//field type must be function type
	ft, ok := method.Type.(*ast.FuncType)
	if !ok {
		return result, false
	}

	if len(method.Names) == 0 {
		return result, false
	}

	result.Name = method.Names[0].Name
	result.Comments = v.parseComments(method.Doc)

	var params []Param
	nextParamId := 0
	if ft.Params != nil {
		for _, field := range ft.Params.List {
			r, ok := v.parseParams(field)
			if ok {
				//add default parameter names
				for i, p := range r {
					if p.Name == "" {
						r[i].Name = defaultParamName(nextParamId)
						nextParamId++
					}
				}
				params = append(params, r...)
			}
		}
	}

	var returns []Param
	if ft.Results != nil {
		for _, field := range ft.Results.List {
			r, ok := v.parseParams(field)
			if ok {
				returns = append(returns, r...)
			}
		}
	}

	result.Params = params
	result.Returns = returns

	return result, true
}

func defaultParamName(i int) string {
	return fmt.Sprintf("p%v", i)
}

// A single field can declare multiple parameters, e.g. for the method declaration
// Method(a, b, c int) the three parameters would be represented by a single ast.Field.
func (v Visitor) parseParams(param *ast.Field) ([]Param, bool) {
	var result []Param

	paramType := v.parseParamType(param.Type)
	//param.Names is nil for unnamed parameters, e.g. in return values
	if len(param.Names) == 0 {
		result = []Param{{Type: paramType}}
	} else {
		result = make([]Param, len(param.Names))
		for i, name := range param.Names {
			result[i] = Param{
				Name: name.Name,
				Type: paramType,
			}
		}
	}

	return result, true
}

// TODO there are some other possible types that we don't handle like
// functions, channels, anonymous interfaces or structs, ...
func (v Visitor) parseParamType(t ast.Expr) ParamType {
	switch pt := t.(type) {
	case *ast.SelectorExpr:
		typeName := pt.Sel.Name
		var typePackageFull string
		if p, ok := pt.X.(*ast.Ident); ok {
			typePackageShort := p.Name
			typePackageFull, ok = v.Imports[typePackageShort]
			if !ok {
				panic(fmt.Sprintf("visitor does not contain full package name for package alias: %v", typePackageShort))
			}
		}
		return SimpleType{Type: typeName, Package: typePackageFull}
	case *ast.Ident:
		if isBasicType(pt.Name) {
			return SimpleType{Type: pt.Name}
		} else {
			//a type that is not a built-in type but has no package qualifier is defined in the current package
			return SimpleType{Type: pt.Name, Package: v.PackagePath}
		}
	case *ast.ArrayType:
		inner := v.parseParamType(pt.Elt)
		return ArrayType{Type: inner}
	case *ast.MapType:
		kpt := v.parseParamType(pt.Key)
		vpt := v.parseParamType(pt.Value)
		return MapType{KeyType: kpt, ValueType: vpt}
	case *ast.StarExpr:
		inner := v.parseParamType(pt.X)
		return StarType{Type: inner}
	case *ast.InterfaceType:
		return SimpleType{Type: "interface{}"}
	default:
		panic("tried to parase unimplemented parameter type")
	}
}
