package annotations

import (
	"testing"

	"github.com/d39b/kit/tools/codegen/parse"

	"github.com/stretchr/testify/assert"
)

func TestParseAnnotations(t *testing.T) {
	a := assert.New(t)

	cases := []struct {
		Comments     []string
		Result       map[string]string
		ReturnsError bool
	}{
		{
			Comments: []string{
				`@Kit{"abc":"xyz",   "cde": 123, "efg": {"a": 123, "b": 345, "c": {"d": "e"}}}`,
			},
			Result: map[string]string{
				"Kit": `{"abc":"xyz",   "cde": 123, "efg": {"a": 123, "b": 345, "c": {"d": "e"}}}`,
			},
			ReturnsError: false,
		},
		{
			Comments: []string{
				`@Kit{"abc":"xyz",   "cde": 123, "efg": {"a": 123, "b": 345, "c": {"d": "e"}}`,
			},
			Result:       map[string]string{},
			ReturnsError: false,
		},
		{
			Comments: []string{
				"       some other stuff here, {}, {{{{}}}}}{}{}{}{}{}{}{}{}",
				//this one should be ignored because there is no name
				`@{"abc":123}`,
				`asdf   asdfadsf @Kit{"abc":"xyz",   "cde": 123, "efg": {"a": 123, "b": 345, "c": {"d": "e"}}}`,
			},
			Result: map[string]string{
				"Kit": `{"abc":"xyz",   "cde": 123, "efg": {"a": 123, "b": 345, "c": {"d": "e"}}}`,
			},
			ReturnsError: false,
		},
		{
			Comments: []string{
				`@Kit{"abc":"xyz",   "cde": 123, "efg": {"a": 123, "b": 345, "c": {"d": "e"}}}`,
				` aasdfasdfdsf`,
				`@Kit{"abc":"xyz"}`,
			},
			Result:       map[string]string{},
			ReturnsError: true,
		},
	}

	for i, c := range cases {
		annotations, err := parseAnnotations(c.Comments)
		a.Equal(c.ReturnsError, err != nil, "test case %v", i)
		if !c.ReturnsError {
			a.Equal(c.Result, annotations, "test case %v", i)
		}
	}
}

func TestParseInterfaceAnnotations(t *testing.T) {
	a := assert.New(t)

	i := parse.Interface{
		Comments: []string{
			`@Kit{`,
			`"abc":"xyz",`,
			`"cde": 123,`,
			`"efg": {"a": 123, "b": 345, "c": {"d": "e"}}`,
			`}`,
			`@Mock{}`,
		},
		Methods: []parse.Method{
			{
				Comments: []string{
					`@Kit{"abc":"xyz",   "cde": 123, "efg": {"a": 123, "b": 345, "c": {"d": "e"}}}`,
				},
			},
			{
				Comments: []string{},
			},
			{
				Comments: []string{
					`@Kit{"k1":"v1","k2":42,"k3":"v3"}`,
					`@Mock{"a1":"a2"}`,
				},
			},
		},
	}

	annotations, err := ParseInterfaceAnnotations(i)
	a.Nil(err)
	a.Len(annotations, 2)
	a.Contains(annotations, "Kit")
	a.Contains(annotations, "Mock")

	kit := annotations["Kit"]
	a.Equal(InterfaceAnnotation{
		Name:       "Kit",
		Annotation: `{"abc":"xyz","cde": 123,"efg": {"a": 123, "b": 345, "c": {"d": "e"}}}`,
		MethodAnnotations: []string{
			`{"abc":"xyz",   "cde": 123, "efg": {"a": 123, "b": 345, "c": {"d": "e"}}}`,
			"",
			`{"k1":"v1","k2":42,"k3":"v3"}`,
		},
	}, kit)

	mock := annotations["Mock"]
	a.Equal(InterfaceAnnotation{
		Name:       "Mock",
		Annotation: `{}`,
		MethodAnnotations: []string{
			"",
			"",
			`{"a1":"a2"}`,
		},
	}, mock)

	var kitParsed testKitAnnotation
	err = ParseJSONAnnotation(kit.Annotation, &kitParsed)
	a.Nil(err)
	a.Equal(testKitAnnotation{
		Abc: "xyz",
		Cde: 123,
		Efg: testKitAnnotationInner{
			A: 123,
			B: 345,
			C: testKitAnnotationInnerInner{
				D: "e",
			},
		},
	}, kitParsed)
}

type testKitAnnotation struct {
	Abc string
	Cde int
	Efg testKitAnnotationInner
}

type testKitAnnotationInner struct {
	A int
	B int
	C testKitAnnotationInnerInner
}

type testKitAnnotationInnerInner struct {
	D string
}
