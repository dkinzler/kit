package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/dkinzler/kit/endpoint"
	"github.com/dkinzler/kit/errors"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

const errorOrigin = "transport/http"

func newPublicTransportError(inner error, code errors.ErrorCode, message string) error {
	return errors.New(inner, errorOrigin, code).WithPublicMessage(message)
}

func newInternalTransportError(inner error, code errors.ErrorCode, message string) error {
	return errors.New(inner, errorOrigin, code).WithInternalMessage(message)
}

// Tries to decode the body of the given http request into target.
func DecodeJSONBody(r *http.Request, target interface{}) error {
	err := json.NewDecoder(r.Body).Decode(target)
	if err != nil {
		// If the request body could not be decoded, there is probably a problem/bug in the client that made the request.
		// It makes sense to inform the client of the reason the request failed.
		// Therefore we return an error with a public error message that can be sent back to the client in the http response.
		return newPublicTransportError(err, errors.InvalidArgument, "could not decode json request body")
	}
	return nil
}

// Encodes the given value as JSON and writes it to the http response.
func EncodeJSONBody(w http.ResponseWriter, source interface{}) error {
	err := json.NewEncoder(w).Encode(source)
	if err != nil {
		// Use an internal error message here, clients do not need to know about this error.
		// If the response could not be encoded this usually indicates a bug in the application using this package.
		return newInternalTransportError(err, errors.Internal, "could not encode json response body")
	}
	return nil
}

// Returns the value of the given url parameter.
// This is designed to work with path variables of the "github.com/gorilla/mux" package.
//
// Example:
//
//	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
//	  // If e.g. the url of the request is "/somepath/123abc", then v will equal "123abc".
//	  v, ok := DecodeURLParameter(r, "xyz")
//	  ...
//	}
//
//	r := mux.r := mux.NewRouter()
//	r.HandleFunc("/somepath/{xyz}", handlerFunc)
func DecodeURLParameter(r *http.Request, name string) (string, error) {
	value, ok := mux.Vars(r)[name]
	// This shouldn't happen since we should only call this function in handlers for routes that contain the corresponding parameter.
	// If we get this error, there is probably a bug in the code that uses this function.
	if !ok {
		return "", newInternalTransportError(nil, errors.Internal, "url parameternot found, this is probably a bug")
	}
	return value, nil
}

var schemaDecoder = schema.NewDecoder()

// Decodes the query parameters in the url of the given request into v, which should be a pointer to a struct.
// Struct tags can be used to define custom field names or ignore struct fields (see the "github.com/gorilla/schema" package for more information).
//
// Example:
//
//	type X struct {
//	  A string `schema:"a1"`
//	  B int `schema:"a2"`
//	  // Ignore this field
//	  C int `schema:"-"`
//	}
func DecodeQueryParameters(r *http.Request, v interface{}) error {
	err := r.ParseForm()
	if err != nil {
		return newPublicTransportError(nil, errors.InvalidArgument, "could not parse query parameters")
	}
	err = schemaDecoder.Decode(v, r.Form)
	if err != nil {
		return newPublicTransportError(err, errors.InvalidArgument, "could not decode query parameters")
	}
	return nil
}

// A generic response encoder function for Go kit (github.com/go-kit/kit).
// Use this function only if the response value returned by the endpoint implements the Responder interface from package "github.com/dkinzler/kit/endpoint".
func MakeGenericJSONEncodeFunc(status int) kithttp.EncodeResponseFunc {
	return func(ctx context.Context, w http.ResponseWriter, response interface{}) error {
		resp, ok := response.(endpoint.Responder)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return newInternalTransportError(nil, errors.Internal, "generic http encode func used with response type that does not implement Responder, this is probably a bug")
		}
		if resp.Error() != nil {
			return EncodeError(ctx, resp.Error(), w)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if resp.Response() != nil {
			return EncodeJSONBody(w, resp.Response())
		}
		return nil
	}
}

// Determines an appropriate http response code for the given error.
// If the error is of type Error from package "github.com/dkinzler/kit/errors", the response code is based on the error code of the error.
// Otherwise http.StatusInternalServerError is returned.
func ErrToCode(err error) int {
	if e, ok := err.(errors.Error); ok {
		switch e.Code {
		case errors.InvalidArgument:
			return http.StatusBadRequest
		case errors.FailedPrecondition:
			return http.StatusBadRequest
		case errors.PermissionDenied:
			return http.StatusForbidden
		case errors.Unauthenticated:
			return http.StatusUnauthorized
		case errors.NotFound:
			return http.StatusNotFound
		default:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}

// Sends an appropriate http status code and response body based on the error.
// If the error is of type Error from package "github.com/dkinzler/kit/errors" and contains a public error code or message,
// that information will be encoded as json and sent in the response body.
//
// The json body has the following format:
//
//	{
//	  "error": {
//	    "code": 42,
//		"message": "this is an example error message"
//	  }
//	}
func EncodeError(_ context.Context, err error, w http.ResponseWriter) error {
	w.WriteHeader(ErrToCode(err))
	if e, ok := err.(errors.Error); ok {
		if e.PublicCode != 0 || e.PublicMessage != "" {
			return json.NewEncoder(w).Encode(jsonErrorBody(e.PublicCode, e.PublicMessage))
		}
	}
	return nil
}

type jsonErrorWrapper struct {
	Error jsonError `json:"error"`
}

type jsonError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func jsonErrorBody(code int, message string) interface{} {
	return jsonErrorWrapper{
		Error: jsonError{
			Code:    code,
			Message: message,
		},
	}
}

// LogErrorHandler logs errors that occur while processing a http request.
// It can be passed as a ServerOption when creating a new http server handler with the "github.com/go-kit/kit/transport/http" package.
type LogErrorHandler struct {
	logger log.Logger
}

func NewLogErrorHandler(logger log.Logger) *LogErrorHandler {
	return &LogErrorHandler{
		logger: logger,
	}
}

func (h *LogErrorHandler) Handle(ctx context.Context, err error) {
	e, ok := err.(errors.Error)
	if ok {
		h.logger.Log("error", e.ToMap())
	} else {
		h.logger.Log("error", err)
	}
}

type maxRequestBodySizeHandler struct {
	next http.Handler
	n    int64
}

func (m *maxRequestBodySizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, m.n)
	m.next.ServeHTTP(w, r)
}

// Limit the request body size of the given http handler to the specified number of bytes.
// If the request body is larger, reading beyond the limit will return an error.
func NewMaxRequestBodySizeHandler(next http.Handler, maxBytes int64) http.Handler {
	return &maxRequestBodySizeHandler{
		next: next,
		n:    maxBytes,
	}
}
