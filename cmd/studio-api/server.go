package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

func serveUntilStopped(parent context.Context, server *http.Server, listener net.Listener) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	shutdownDone := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		shutdownDone <- server.Shutdown(shutdownCtx)
	}()
	serveErr := server.Serve(listener)
	cancel()
	shutdownErr := <-shutdownDone // wait for in-flight handlers before closing dependencies
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return errors.Join(serveErr, shutdownErr)
	}
	return shutdownErr
}
