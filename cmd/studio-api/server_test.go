package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServeUntilStoppedWaitsForInFlightRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered, release := make(chan struct{}), make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		_, _ = w.Write([]byte("complete"))
	})}
	served := make(chan error, 1)
	go func() { served <- serveUntilStopped(ctx, server, listener) }()
	response := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 2 * time.Second}
		result, err := client.Get("http://" + listener.Addr().String())
		if err == nil {
			_, err = io.ReadAll(result.Body)
			result.Body.Close()
		}
		response <- err
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not enter the server")
	}
	cancel()
	select {
	case err := <-served:
		t.Fatalf("server stopped before the request completed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-response:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request did not complete")
	}
	select {
	case err := <-served:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server shutdown did not complete")
	}
}
