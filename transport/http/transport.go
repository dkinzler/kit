// Package http includes functionality to decode and encode http requests/responses/parameters/errors, http middlewares and graceful shutdown handling.
// It is designed to work with Go kit (github.com/go-kit/kit/transport/http) and the Gorilla web toolkit (github.com/gorilla).
package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/d39b/kit/endpoint"
	"github.com/d39b/kit/errors"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

// TODO change this to http?
const errorOrigin = "transport"

func newPublicTransportError(inner error, code errors.ErrorCode, message string) error {
	return errors.New(inner, errorOrigin, code).WithPublicMessage(message)
}

func newInternalTransportError(inner error, code errors.ErrorCode, message string) error {
	return errors.New(inner, errorOrigin, code).WithInternalMessage(message)
}

// Tries to decode the body of the given request as JSON and store the result in target.
func DecodeJSONBody(r *http.Request, target interface{}) error {
	err := json.NewDecoder(r.Body).Decode(target)
	if err != nil {
		// Use a public error message, it might be send back in the body of the response.
		// If the request body could not be decoded there is probably a problem/bug in the client, so it makes sense to inform the client of the reason the request failed.
		return newPublicTransportError(err, errors.InvalidArgument, "could not decode json request body")
	}
	return nil
}

// Encode the value as JSON and write it to the given http response.
func EncodeJSONBody(w http.ResponseWriter, source interface{}) error {
	err := json.NewEncoder(w).Encode(source)
	if err != nil {
		// Use an internal error here, if the request could not be encoded this usually indicates a bug in the server application.
		// Clients do not need to know about this error.
		return newInternalTransportError(err, errors.Internal, "could not encode json response body")
	}
	return nil
}

func DecodeURLParameter(r *http.Request, name string) (string, error) {
	value, ok := mux.Vars(r)[name]
	//Note: normally this shouldn't happen since we should only call this method for routes that contain the corresponding parameter.
	//If we get this error, there is probably a bug in the code that uses this function.
	if !ok {
		return "", newInternalTransportError(nil, errors.Internal, "url parameter not found, this is probably a bug")
	}
	return value, nil
}

var schemaDecoder = schema.NewDecoder()

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

// Only use this function if the response value returned by the endpoint implements endpoint.Responder.
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

// Determines an appropriate response code for the given error.
// If the error is of type errors.Error, the response code is set based on the error code of the error.
// Otherwise returns http.StatusInternalServerError.
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

// Sends an appropriate status code and response body based on the error.
// If the error is of type errors.Error and contains a public error code or message, those will be encoded as json and sent in the response body.
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
// If request body is larger, reading beyond the limit will return an error.
func NewMaxRequestBodySizeHandler(next http.Handler, maxBytes int64) http.Handler {
	return &maxRequestBodySizeHandler{
		next: next,
		n:    maxBytes,
	}
}
