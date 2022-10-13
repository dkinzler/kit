package http

import (
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOnShutdownFunctionCalled(t *testing.T) {
	srv := &http.Server{Addr: ":12345"}
	called := false
	go func() {
		time.Sleep(10 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()
	shutdown := HandleShutdown(srv, func(err error) {
		called = true
	})
	<-shutdown
	assert.True(t, called)
}
