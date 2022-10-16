package http

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

/*
Sets up graceful shutdown for the given http server.
Server will be shutdown when a message is received on the given channel.
Call this function before invoking ListenAndServe(...) on the server and then read from the returned channel.

Example:

	srv := &http.Server{...}

	// Shutdown the server when a SIGINT or SIGTERM signal is received, e.g. when the process is killed
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	shutdown := HandleShutdown(srv, sig, func(err error) {
		// do something on shutdown, e.g. log the error if not nil
	})

	// When the shutdown handler calls srv.Shutdown(), ListenAndServe will immediately return with http.ErrServerClsoed.
	// If ListenAndServe returns another error, terminate the program.
	// Otherwise the read on the shutdown channel might block forever since ListenAndServe did not return because of a shutdown signal.
	// Alternatively one could pass another channel to HandleShutdown that can be send a value here.
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		os.Exit(0)
	}

	//wait for shutdown to complete
	<-shutdown
*/
func HandleShutdown(srv *http.Server, closeChan <-chan struct{}, onShutdown func(error), timeout time.Duration) <-chan struct{} {
	shutdown := make(chan struct{})
	go func() {
		<-closeChan

		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), timeout)
		defer cancelShutdown()
		// Wait for open connections to finish up
		// Note that calling Shutdown on a server that never started to listen or has already been shut down is fine.
		err := srv.Shutdown(ctxShutdown)
		if onShutdown != nil {
			onShutdown(err)
		}
		// close shutdown channel which will cause any channel reads to unblock
		close(shutdown)
	}()
	return shutdown
}

type ServerConfig struct {
	Address string
	// Defaults to 80
	Port int

	// Maximum size of request body in bytes, 0 = no limit, defaults to 128kb
	RequestMaxBodyBytes int
	// Defaults to 128kb
	RequestMaxHeaderBytes int
	// Defaults to 7s
	RequestTimeout time.Duration
	// Defaults to 10s
	WriteTimeout time.Duration
	// Defaults to 10s
	ReadTimeout time.Duration

	// Called when a panic is caught in a http handler
	OnPanicFunc func(interface{})
	// Called when the server is shut down with the error returned by the Shutdown() method
	OnShutdownFunc func(error)
}

func NewServerConfig() ServerConfig {
	return ServerConfig{
		Address:               "",
		Port:                  80,
		RequestMaxBodyBytes:   1024 * 128,
		RequestMaxHeaderBytes: 1024 * 128,
		RequestTimeout:        7 * time.Second,
		WriteTimeout:          10 * time.Second,
		ReadTimeout:           10 * time.Second,
	}
}

func (s ServerConfig) WithAddress(address string) ServerConfig {
	s.Address = address
	return s
}

func (s ServerConfig) WithPort(port int) ServerConfig {
	s.Port = port
	return s
}

func (s ServerConfig) WithRequestMaxBodyBytes(maxBytes int) ServerConfig {
	s.RequestMaxBodyBytes = maxBytes
	return s
}

func (s ServerConfig) WithRequestMaxHeaderBytes(maxBytes int) ServerConfig {
	s.RequestMaxHeaderBytes = maxBytes
	return s
}

func (s ServerConfig) WithRequestTimeout(timeout time.Duration) ServerConfig {
	s.RequestTimeout = timeout
	return s
}

func (s ServerConfig) WithWriteTimeout(timeout time.Duration) ServerConfig {
	s.WriteTimeout = timeout
	return s
}

func (s ServerConfig) WithReadTimeout(timeout time.Duration) ServerConfig {
	s.ReadTimeout = timeout
	return s
}

func (s ServerConfig) WithOnPanicFunc(onPanic func(interface{})) ServerConfig {
	s.OnPanicFunc = onPanic
	return s
}

func (s ServerConfig) WithOnShutdownFunc(onShutdown func(error)) ServerConfig {
	s.OnShutdownFunc = onShutdown
	return s
}

// Creates a new http server and starts listening with the given handler, config and useful defaults.
// Middlewares to catch panics and to timeout requests are added and server shutdown is handled gracefully.
//
// This function blocks until a signal to shutdown the server is received, it then tries
// to gracefully shutdown the server and eventually returns. We wait for open connections/requests to complete for 10 seconds.
// The server can be stopped/shut down using the following signals:
//   - the program receives a SIGINT signal (e.g. when pressing CTRL+c in a terminal)
//   - the program receives a SIGTERM signal (e.g. when the process is killed)
//   - a value is sent on closeChan
//
// When using a close channel, make sure to send any values in a non-blocking way.
//
// Returns any errors from ListenAndServer() that are not http.ErrServerClosed.
func RunDefaultServer(handler http.Handler, closeChan <-chan struct{}, config ServerConfig) error {
	var h http.Handler = handler

	if config.RequestMaxBodyBytes > 0 {
		h = NewMaxRequestBodySizeHandler(h, int64(config.RequestMaxBodyBytes))
	}

	// catch panics
	h = PanicMiddleware(h, config.OnPanicFunc)

	// request timeout
	if config.RequestTimeout > 0 {
		h = http.TimeoutHandler(h, config.RequestTimeout, "request timed out")
	}

	srv := &http.Server{
		Handler:      h,
		Addr:         config.Address + ":" + strconv.Itoa(config.Port),
		WriteTimeout: config.WriteTimeout,
		// this timeout also applies to reading the request header
		ReadTimeout: config.ReadTimeout,
		// default is 1MB, but this might be a bit large
		MaxHeaderBytes: config.RequestMaxHeaderBytes,
	}

	// Sending a value on this channel will shutdown the server.
	c := make(chan struct{}, 1)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		// There are multiple sources that can cause a shutdown,
		// a SIGINT or SIGTERM signal to the process or a value sent on closeChan.
		// Since the server can be shutdown only once, only the first shutdown event
		// needs to be processed.
		select {
		case <-sig:
			// Perfrom a non-blocking send.
			// If a send would block, that just means that a value has already been sent, which
			// will already cause the server to shutdown.
			select {
			case c <- struct{}{}:
			default:
			}
		case <-closeChan:
			select {
			case c <- struct{}{}:
			default:
			}
		}
	}()

	shutdown := HandleShutdown(srv, c, config.OnShutdownFunc, 10*time.Second)

	var returnError error

	// When Shutdown is called ListenAndServe returns immediately with http.ErrServerClosed.
	// However, open connections might still be running, therefore we wait below (with the shutdown channel) until the shutdown is complete.
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		returnError = err
		// Perform a non-blocking send to ensure that the shutdown handler runs.
		// When ListenAndServe returns with an error other than http.ErrServerClosed, Shutdown() has not been called on the server by our
		// shutdown handler. Therefore the read on the shutdown channel below would block forever.
		select {
		case c <- struct{}{}:
		default:
		}
	}

	// Wait for server shutdown to complete, there might still be open connections/requests.
	<-shutdown
	return returnError
}
