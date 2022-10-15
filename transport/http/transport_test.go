package http

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/d39b/kit/endpoint"
	"github.com/d39b/kit/errors"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	Hello string   `json:"hello"`
	This  string   `json:"this"`
	A     string   `json:"a"`
	X     float64  `json:"x"`
	Y     []string `json:"y"`
}

func TestDecodeJSONBodyWorks(t *testing.T) {
	a := assert.New(t)

	jsonContent := `{
	    "hello" : "there",
		"this" : "is",
		"a" : "test",
		"x" : 23.0,
		"y" : [
			"v1",
			"v2",
			"v3"
		]
	}`
	req := httptest.NewRequest("POST", "https://example.com/foo", strings.NewReader(jsonContent))
	var decoded map[string]interface{}
	err := DecodeJSONBody(req, &decoded)
	a.Nil(err)
	expected := map[string]interface{}{
		"hello": "there",
		"this":  "is",
		"a":     "test",
		"x":     23.0,
		"y": []interface{}{
			"v1",
			"v2",
			"v3",
		},
	}
	a.Equal(expected, decoded)

	// works with struct
	req = httptest.NewRequest("POST", "https://example.com/foo", strings.NewReader(jsonContent))
	var decodedStruct TestStruct
	err = DecodeJSONBody(req, &decodedStruct)
	a.Nil(err)
	expectedStruct := TestStruct{
		Hello: "there",
		This:  "is",
		A:     "test",
		X:     23,
		Y: []string{
			"v1",
			"v2",
			"v3",
		},
	}
	a.Equal(expectedStruct, decodedStruct)
}

func TestDecodeJSONBodyReturnsErrorIfDecodingFails(t *testing.T) {
	a := assert.New(t)
	// invalid json, because a "," is missing in the third line
	jsonContent := `{
	    "hello" : "there",
		"this" : "is"
		"a" : "test",
		"x" : 23,
		"y" : [
			"v1",
			"v2",
			"v3"
		]
	}`
	req := httptest.NewRequest("POST", "https://example.com/foo", strings.NewReader(jsonContent))
	var decoded TestStruct
	err := DecodeJSONBody(req, &decoded)
	a.NotNil(err)
	a.True(errors.IsInvalidArgumentError(err))
}

func TestEncodeJSONBody(t *testing.T) {
	a := assert.New(t)

	v := TestStruct{
		Hello: "hey",
		This:  "yes",
		A:     "b",
		X:     9001,
		Y:     []string{"oh", "no", "no", "no"},
	}
	w := httptest.NewRecorder()
	err := EncodeJSONBody(w, &v)
	a.Nil(err)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	var actual TestStruct
	err = json.Unmarshal(body, &actual)
	a.Nil(err)
	a.Equal(v, actual)
}

func TestEncodeJSONBodyReturnsErrorCorrectly(t *testing.T) {
	a := assert.New(t)

	w := httptest.NewRecorder()
	// json.Marshal does not support function values
	err := EncodeJSONBody(w, func() {})
	a.NotNil(err)
	a.True(errors.IsInternalError(err))
}

func TestDecodeURLParameter(t *testing.T) {
	a := assert.New(t)

	r := httptest.NewRequest("GET", "http://example.com/events/e-1234-5678", nil)
	w := httptest.NewRecorder()
	expected := "e-1234-5678"
	var actual string
	var err error
	router := mux.NewRouter()
	router.HandleFunc("/events/{eventid}", func(w http.ResponseWriter, r *http.Request) {
		actual, err = DecodeURLParameter(r, "eventid")
	})
	router.ServeHTTP(w, r)
	a.Nil(err)
	a.Equal(expected, actual)
}

func TestDecodeURLParameterReturnsErrorOnMissingParam(t *testing.T) {
	a := assert.New(t)

	r := httptest.NewRequest("GET", "http://example.com/events", nil)
	w := httptest.NewRecorder()
	var actual string
	var err error
	router := mux.NewRouter()
	router.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		actual, err = DecodeURLParameter(r, "eventid")
	})
	router.ServeHTTP(w, r)
	a.NotNil(err)
	a.True(errors.IsInternalError(err))
	a.Empty(actual)
}

type DecodeQueryStruct struct {
	From   string   `schema:"from"`
	To     int      `schema:"to"`
	Status []string `schema:"status"`
}

func TestDecodeQueryParameters(t *testing.T) {
	a := assert.New(t)

	r := httptest.NewRequest("GET", "https://example.com/foo/bar/yes?from=hello123&to=12345&status=open&status=kek", nil)
	var actual DecodeQueryStruct
	err := DecodeQueryParameters(r, &actual)
	a.Nil(err)
	expected := DecodeQueryStruct{
		From:   "hello123",
		To:     12345,
		Status: []string{"open", "kek"},
	}
	a.Equal(expected, actual)
}

func TestEncodeErrorWorks(t *testing.T) {
	a := assert.New(t)

	cases := []struct {
		E            error
		ExpectedCode int
		ExpectedBody interface{}
	}{
		{
			E:            errors.New(nil, "test", errors.NotFound),
			ExpectedCode: http.StatusNotFound,
			ExpectedBody: nil,
		},
		{
			E:            errors.New(nil, "test", errors.InvalidArgument),
			ExpectedCode: http.StatusBadRequest,
			ExpectedBody: nil,
		},
		{
			E:            errors.New(nil, "test", errors.FailedPrecondition),
			ExpectedCode: http.StatusBadRequest,
			ExpectedBody: nil,
		},
		{
			E:            errors.New(nil, "test", errors.PermissionDenied),
			ExpectedCode: http.StatusForbidden,
			ExpectedBody: nil,
		},
		{
			E:            errors.New(nil, "test", errors.Unauthenticated),
			ExpectedCode: http.StatusUnauthorized,
			ExpectedBody: nil,
		},
		{
			E:            errors.New(nil, "test", errors.Internal).WithInternalCode(42).WithInternalMessage("abc"),
			ExpectedCode: http.StatusInternalServerError,
			ExpectedBody: nil,
		},
		{
			E:            stderrors.New("test"),
			ExpectedCode: http.StatusInternalServerError,
			ExpectedBody: nil,
		},
		{
			E: errors.New(nil, "test", errors.NotFound).
				WithPublicCode(17).WithPublicMessage("justsomeerrormessage"),
			ExpectedCode: http.StatusNotFound,
			ExpectedBody: jsonErrorBody(17, "justsomeerrormessage"),
		},
		{
			E: errors.New(nil, "test", errors.InvalidArgument).
				WithPublicCode(1337).WithPublicMessage("anothermessage"),
			ExpectedCode: http.StatusBadRequest,
			ExpectedBody: jsonErrorBody(1337, "anothermessage"),
		},
	}

	for i, c := range cases {
		w := httptest.NewRecorder()
		EncodeError(nil, c.E, w)
		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		a.Equal(c.ExpectedCode, resp.StatusCode, "test case %v", i)
		if c.ExpectedBody != nil {
			var actual jsonErrorWrapper
			err := json.Unmarshal(body, &actual)
			a.Nil(err, "test case %v", i)
			a.Equal(c.ExpectedBody, actual, "test case %v", i)
		} else {
			a.Empty(body, "test case %v", i)
		}
	}
}

func TestGenericJSONEncodeFunc(t *testing.T) {
	a := assert.New(t)

	var x interface{}
	// if response does not implement endpoint.Responder should return status InternalServerError
	w := httptest.NewRecorder()
	x = map[string]interface{}{"test": "hello"}
	err := MakeGenericJSONEncodeFunc(http.StatusOK)(context.Background(), w, x)
	a.NotNil(err)
	a.True(errors.IsInternalError(err))
	a.Equal(http.StatusInternalServerError, w.Result().StatusCode)

	// if response contains an error the respective status code should be set
	w = httptest.NewRecorder()
	x = endpoint.Response{
		R:   nil,
		Err: errors.New(nil, "test", errors.InvalidArgument),
	}
	err = MakeGenericJSONEncodeFunc(http.StatusOK)(context.Background(), w, x)
	a.Nil(err)
	a.Equal(http.StatusBadRequest, w.Result().StatusCode)

	// response does not contain err
	w = httptest.NewRecorder()
	expected := map[string]interface{}{
		"test":      "sometestvalue",
		"othertest": "someothervalue",
	}
	x = endpoint.Response{
		R:   expected,
		Err: nil,
	}
	err = MakeGenericJSONEncodeFunc(http.StatusCreated)(context.Background(), w, x)
	a.Nil(err)
	a.Equal(http.StatusCreated, w.Result().StatusCode)
	body, err := io.ReadAll(w.Result().Body)
	a.Nil(err)
	var u map[string]interface{}
	err = json.Unmarshal(body, &u)
	a.Nil(err)
	a.Equal(expected, u)
}

func TestMaxRequestBodySizeHandler(t *testing.T) {
	a := assert.New(t)

	gotError := false
	var handler http.Handler
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var x map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&x)
		if err != nil {
			if err.Error() == "http: request body too large" {
				gotError = true
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	})
	handler = NewMaxRequestBodySizeHandler(handler, 1000)

	// this request body will be too large
	var longString string
	for i := 0; i < 2000; i++ {
		longString += "a"
	}
	bodyBytes, err := json.Marshal(map[string]interface{}{"x": longString})
	a.Nil(err)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	handler.ServeHTTP(w, r)
	a.True(gotError)
	a.Equal(http.StatusBadRequest, w.Result().StatusCode)

	// request body < 1000 bytes, request should work
	gotError = false
	bodyBytes, err = json.Marshal(map[string]interface{}{"x": "this is not that long"})
	a.Nil(err)
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	handler.ServeHTTP(w, r)
	a.False(gotError)
	a.Equal(http.StatusCreated, w.Result().StatusCode)
}
