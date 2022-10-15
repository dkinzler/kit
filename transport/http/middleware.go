package http

import (
	"net/http"
)

// Http middleware that recovers and calls the provided onPanic function if the next http handler panics.
// Status code 500 Internal Server Error is written to the response header.
func PanicMiddleware(next http.Handler, onPanic func(e interface{})) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				onPanic(e)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
