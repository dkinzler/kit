package http

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

/*
Sets up graceful shutdown for the given http server on SIGINT (e.g. when pressing CTRL+C in terminal) and SIGTERM (e.g. sent by kill to terminate a process).
Call this function before calling srv.ListenAndServe(...) and then read from the returned channel.

	Example:

	srv := &http.Server{...}

	shutdown := HandleShutdown(srv, func(err error) {
		//do something on shutdown, e.g. log the error if not nil
	})

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		//if the listen and serve throws an error we need to terminate the program, because otherwise the read on the shutdown channel below would block
		os.Exit(0)
	}

	//wait for signal from shutdown channel
	<-shutdown
*/
func HandleShutdown(srv *http.Server, onShutdown func(error)) chan struct{} {
	shutdown := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		//wait for interrupt/terminate signal
		<-sig

		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		//wait for open connections to finish up
		err := srv.Shutdown(ctxShutdown)
		onShutdown(err)
		//this will close the shutdown channel which will cause any channel reads to unblock
		close(shutdown)
	}()
	return shutdown
}
