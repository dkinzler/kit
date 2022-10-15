package http

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-kit/log"
)

type ServerConfig struct {
	address string
	// Defaults to 80
	port int

	// Maximum size of request body in bytes, 0 = no limit, defaults to 128kb
	requestMaxBodyBytes int
	// Defaults to 128kb
	requestMaxHeaderBytes int
	// Defaults to 7s
	requestTimeout time.Duration
	// Defaults to 10s
	writeTimeout time.Duration
	// Defaults to 10s
	readTimeout time.Duration
}

func NewServerConfig() ServerConfig {
	return ServerConfig{
		address:               "",
		port:                  80,
		requestMaxBodyBytes:   1024 * 128,
		requestMaxHeaderBytes: 1024 * 128,
		requestTimeout:        7 * time.Second,
		writeTimeout:          10 * time.Second,
		readTimeout:           10 * time.Second,
	}
}

func (s ServerConfig) WithAddress(address string) ServerConfig {
	s.address = address
	return s
}

func (s ServerConfig) WithPort(port int) ServerConfig {
	s.port = port
	return s
}

func (s ServerConfig) WithRequestMaxBodyBytes(maxBytes int) ServerConfig {
	s.requestMaxBodyBytes = maxBytes
	return s
}

func (s ServerConfig) WithRequestMaxHeaderBytes(maxBytes int) ServerConfig {
	s.requestMaxHeaderBytes = maxBytes
	return s
}

func (s ServerConfig) WithRequestTimeout(timeout time.Duration) ServerConfig {
	s.requestTimeout = timeout
	return s
}

func (s ServerConfig) WithWriteTimeout(timeout time.Duration) ServerConfig {
	s.writeTimeout = timeout
	return s
}

func (s ServerConfig) WithReadTimeout(timeout time.Duration) ServerConfig {
	s.readTimeout = timeout
	return s
}

func RunDefaultServer(handler http.Handler, logger log.Logger, config ServerConfig) {
	var h http.Handler = handler

	if config.requestMaxBodyBytes > 0 {
		// limit max request size to 128kb
		h = NewMaxRequestBodySizeHandler(h, int64(config.requestMaxBodyBytes))
	}

	// catch and log panics
	h = PanicMiddleware(h, func(e interface{}) {
		logger.Log("msg", "caught panic", "error", e)
	})

	// request timeout
	if config.requestTimeout > 0 {
		h = http.TimeoutHandler(h, config.requestTimeout, "request timed out")
	}

	srv := &http.Server{
		Handler:      h,
		Addr:         config.address + ":" + strconv.Itoa(config.port),
		WriteTimeout: config.writeTimeout,
		// this timeout also applies to reading the request header
		ReadTimeout: config.readTimeout,
		// default is 1MB, but this might be a bit large
		MaxHeaderBytes: config.requestMaxHeaderBytes,
	}

	// set up graceful shutdown on SIGINT (e.g. when pressing CONTROL+c for terminal command) and SIGTERM (e.g. sent by kill to terminate a process)
	shutdown := HandleShutdown(srv, func(err error) {
		logger.Log("msg", "shutting down...")
		if err != nil {
			logger.Log("error", "error shutting down server")
		}
	})

	// TODO maybe don't terminate program here, but instead have another channel that we switch over?

	// when shutdown is called ListenAndServe returns immediately with http.ErrServerClosed
	// however, open connections might still be running, therefore we wait below (with the shutdown channel) until the shutdown is complete
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		//if the listen and serve throws an error we need to terminate the server, because otherwise the read on the shutdown channel below would block
		logger.Log("error", "http server ListenAndServe error")
		os.Exit(0)
	}

	<-shutdown
}
