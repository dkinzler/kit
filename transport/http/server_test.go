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

	c := make(chan struct{})
	go func() {
		time.Sleep(10 * time.Millisecond)
		c <- struct{}{}
	}()
	shutdown := HandleShutdown(srv, c, func(err error) {
		called = true
	}, 5*time.Second)
	<-shutdown
	assert.True(t, called)
}

func TestDefaultServerShutdown(t *testing.T) {
	a := assert.New(t)

	// shutdown server by killing the process
	onShutdownCalled := false
	onShutdown := func(e error) {
		onShutdownCalled = true
	}

	config := NewServerConfig().WithPort(9001).WithOnShutdownFunc(onShutdown)
	go func() {
		time.Sleep(10 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()
	err := RunDefaultServer(nil, nil, config)
	a.Nil(err)
	a.True(onShutdownCalled)

	// shutdown server using a close channel
	onShutdownCalled = false

	c := make(chan struct{})
	go func() {
		time.Sleep(10 * time.Millisecond)
		c <- struct{}{}
	}()
	err = RunDefaultServer(nil, c, config)
	a.Nil(err)
	a.True(onShutdownCalled)

	onShutdownCalled = false
	c = make(chan struct{})
	// invalid address will cause an error to be returned from ListenAndServe()
	// but shutdown handler should still be called
	err = RunDefaultServer(nil, c, config.WithAddress("::::::"))
	a.NotNil(err)
	a.True(onShutdownCalled)
}
