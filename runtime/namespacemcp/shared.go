package namespacemcp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// NewShared serves port-zero namespaces on one listener, retaining a separate
// handler, runtime source and authorization context per namespace. A positive
// per-namespace port remains an explicit dedicated-listener override.
func NewShared(factory Factory, address string) (*Manager, error) {
	manager, err := New(factory)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("reserve shared namespace MCP port: %w", err)
	}
	manager.sharedAddress = listener.Addr().String()
	manager.sharedPort = listener.Addr().(*net.TCPAddr).Port
	manager.sharedActive = true
	manager.sharedServer = &http.Server{Addr: manager.sharedAddress, Handler: http.HandlerFunc(manager.serveShared), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		_ = manager.sharedServer.Serve(listener)
		manager.mu.Lock()
		manager.sharedActive = false
		manager.mu.Unlock()
	}()
	return manager, nil
}
func (m *Manager) sharedAvailable() bool { m.mu.Lock(); defer m.mu.Unlock(); return m.sharedActive }
func (m *Manager) serveShared(w http.ResponseWriter, r *http.Request) {
	// Do not normalize ambiguous encodings or traversal into another namespace.
	path := r.URL.Path
	const metadataPrefix = "/.well-known/oauth-protected-resource"
	metadata := strings.HasPrefix(path, metadataPrefix+"/namespaces/")
	if metadata {
		path = strings.TrimPrefix(path, metadataPrefix)
	}
	parts := strings.Split(path, "/")
	if r.URL.RawPath != "" || len(parts) < 4 || parts[0] != "" || parts[1] != "namespaces" || len(parts[2]) != 64 {
		http.NotFound(w, r)
		return
	}
	id := parts[2]
	for _, part := range parts[3:] {
		if part == "." || part == ".." || part == "" {
			http.NotFound(w, r)
			return
		}
	}
	m.mu.Lock()
	entry := m.entries[id]
	if m.closed || entry == nil || !entry.shared {
		m.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	entry.requests.Add(1)
	m.mu.Unlock()
	defer entry.requests.Done()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	stop := context.AfterFunc(entry.lifetime, cancel)
	defer stop()
	request := r.Clone(ctx)
	url := *r.URL
	url.Path = "/" + strings.Join(parts[3:], "/")
	if metadata {
		url.Path = metadataPrefix + url.Path
	}
	url.RawPath = ""
	request.URL = &url
	request.RequestURI = url.RequestURI()
	entry.server.Handler.ServeHTTP(w, request)
}
