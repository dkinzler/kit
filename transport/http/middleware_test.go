package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicCaught(t *testing.T) {
	a := assert.New(t)

	//onPanic func should not be called if there is no panic
	handlerWithoutPanic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	called := false
	panicHandler := PanicMiddleware(handlerWithoutPanic, func(e interface{}) {
		called = true
	})
	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	panicHandler.ServeHTTP(w, r)
	a.False(called)
	a.Equal(http.StatusOK, w.Result().StatusCode)

	//onPanic called when handler panics
	handlerWithPanic := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("xyz")
	})
	called = false
	panicHandler = PanicMiddleware(handlerWithPanic, func(e interface{}) {
		called = true
	})
	r = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	panicHandler.ServeHTTP(w, r)
	a.True(called)
	a.Equal(http.StatusInternalServerError, w.Result().StatusCode)
}
