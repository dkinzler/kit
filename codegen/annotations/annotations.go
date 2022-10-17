// Package annotations finds and parses annotations in the comments to an interface or method.
// An annotation must have the following format: @Name{"key1":"value1", "key2": "value2", ...}, i.e.
// it begins with an @ characater, followed by a name followed by a valid json object.
package annotations

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/d39b/kit/codegen/parse"
)

// An annotation on an interface.
type InterfaceAnnotation struct {
	// Name of the annotation
	Name string
	// The annotation on the interface
	Annotation string
	// Annotations on the interface methods.
	// Slice has the same length as parse.Interface.Methods.
	// Contain empty string for methods without an annotation.
	MethodAnnotations []string
}

// Returns a map of annotations found in the comments of the given interface.
// The map keys are the annotation names and the values are instances of InterfaceAnnotation.
func ParseInterfaceAnnotations(i parse.Interface) (map[string]InterfaceAnnotation, error) {
	annotationsOnInterface, err := parseAnnotations(i.Comments)
	if err != nil {
		return nil, err
	}

	annotationsOnMethods := make([]map[string]string, len(i.Methods))
	for j, method := range i.Methods {
		annotationsOnMethod, err := parseAnnotations(method.Comments)
		if err != nil {
			return nil, err
		}
		annotationsOnMethods[j] = annotationsOnMethod
	}

	result := make(map[string]InterfaceAnnotation)
	for name, annot := range annotationsOnInterface {
		x := InterfaceAnnotation{
			Name:       name,
			Annotation: annot,
		}

		methodAnnotations := make([]string, len(i.Methods))
		for j, annotationsOnMethod := range annotationsOnMethods {
			if ma, ok := annotationsOnMethod[name]; ok {
				methodAnnotations[j] = ma
			}
		}
		x.MethodAnnotations = methodAnnotations
		result[name] = x
	}

	return result, nil
}

// Parses the annotation as JSON and stores the result in the "result" parameter, which should usually be a pointer to a struct or map.
func ParseJSONAnnotation(annotation string, result interface{}) error {
	err := json.Unmarshal([]byte(annotation), result)
	if err != nil {
		return err
	}
	return nil
}

func parseAnnotations(comments []string) (map[string]string, error) {
	result := make(map[string]string)

	name := ""
	json := ""
	readingName := false
	readingJson := false
	openBrackets := 0
	closedBrackets := 0

	//iterate over every character, whenever we encouter "@", start parsing a new annotation
	for _, comment := range comments {
		for _, c := range comment {
			if c == '@' {
				if !readingName && !readingJson {
					readingName = true
				} else if readingJson {
					json += string(c)
				}
			} else if c == '{' {
				if readingName {
					readingName = false
					if name != "" {
						readingJson = true
						json = "{"
						openBrackets = 1
					}
				} else if readingJson {
					json += "{"
					openBrackets++
				}
			} else if c == '}' {
				if readingJson {
					json += "}"
					closedBrackets++
					if openBrackets == closedBrackets {
						if _, ok := result[name]; ok {
							return nil, errors.New(fmt.Sprintf("multiple annotations found with name: %v", name))
						}
						result[name] = json
						name = ""
						json = ""
						readingName = false
						readingJson = false
						openBrackets = 0
						closedBrackets = 0
					}
				}
			} else {
				if readingName {
					name += string(c)
				} else if readingJson {
					json += string(c)
				}
			}
		}
	}

	return result, nil
}
