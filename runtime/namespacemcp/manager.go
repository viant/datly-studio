// Package namespacemcp owns independent Datly MCP HTTP listeners. Hosts supply
// a namespace-scoped catalog/source and authorization policy for each endpoint.
package namespacemcp

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	mcpserver "github.com/viant/datly/mcp/server"
)

type Endpoint struct {
	NamespaceID string `json:"namespaceId"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
}

func (e Endpoint) URL() string { return "http://" + e.Address + "/mcp" }

// Factory receives the reserved address, allowing OAuth resource metadata to
// use the actual port. The host must restrict the source to this namespace.
type Factory func(context.Context, Endpoint) (mcpserver.Config, error)

type instance struct {
	endpoint      Endpoint
	requestedPort int
	server        *http.Server
	done          chan struct{}
	cleanup       func(context.Context) error
	reload        func(context.Context, int64) error
	status        func() (string, int64)
}

type Manager struct {
	mu         sync.Mutex
	entries    map[string]*instance
	operations map[string]*sync.Mutex
	closed     bool
	factory    Factory
}

func New(factory Factory) (*Manager, error) {
	if factory == nil {
		return nil, fmt.Errorf("namespace MCP factory is required")
	}
	return &Manager{factory: factory, entries: map[string]*instance{}, operations: map[string]*sync.Mutex{}}, nil
}
func (m *Manager) operation(id string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	lock := m.operations[id]
	if lock == nil {
		lock = &sync.Mutex{}
		m.operations[id] = lock
	}
	return lock
}

// Start reserves a loopback port before constructing the Datly transport.
// Port zero asks the OS for a free port. Existing identical starts are idempotent;
// Start rejects port changes; Rebind prepares a replacement before draining the old listener.
func (m *Manager) Start(ctx context.Context, id string, port int) (Endpoint, error) {
	return m.start(ctx, id, port, false)
}

// Rebind constructs and reserves the new endpoint before replacing the old one.
// Failures before commitment preserve the current listener. A returned endpoint
// remains active even if an error reports failure to drain the previous server.
func (m *Manager) Rebind(ctx context.Context, id string, port int) (Endpoint, error) {
	return m.start(ctx, id, port, true)
}

func (m *Manager) start(ctx context.Context, id string, port int, replace bool) (Endpoint, error) {
	decoded, err := hex.DecodeString(id)
	if err != nil || len(decoded) != 32 || strings.ToLower(id) != id || port < 0 || port > 65535 {
		return Endpoint{}, fmt.Errorf("valid namespace ID and port 0–65535 are required")
	}
	if err = ctx.Err(); err != nil {
		return Endpoint{}, err
	}
	lock := m.operation(id)
	lock.Lock()
	defer lock.Unlock()
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return Endpoint{}, fmt.Errorf("namespace MCP manager is closed")
	}
	previous := m.entries[id]
	if previous != nil {
		if !replace && previous.requestedPort == port || replace && port != 0 && previous.endpoint.Port == port {
			m.mu.Unlock()
			return previous.endpoint, nil
		}
		if !replace {
			m.mu.Unlock()
			return Endpoint{}, fmt.Errorf("use Rebind to change a namespace MCP port")
		}
	}
	m.mu.Unlock()
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return Endpoint{}, fmt.Errorf("reserve namespace MCP port: %w", err)
	}
	registered := false
	defer func() {
		if !registered {
			_ = listener.Close()
		}
	}()
	endpoint := Endpoint{NamespaceID: id, Address: listener.Addr().String(), Port: listener.Addr().(*net.TCPAddr).Port}
	config, err := m.factory(ctx, endpoint)
	if err != nil {
		return Endpoint{}, err
	}
	var cleanup func(context.Context) error
	var reload func(context.Context, int64) error
	if source, ok := config.Source.(interface{ Close(context.Context) error }); ok {
		cleanup = source.Close
	}
	if source, ok := config.Source.(interface {
		Reload(context.Context, int64) error
	}); ok {
		reload = source.Reload
	}
	defer func() {
		if !registered && cleanup != nil {
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = cleanup(closeCtx)
		}
	}()
	config.Transport.Kind = mcpserver.TransportStreamable
	config.Transport.Address = endpoint.Address
	transport, err := mcpserver.New(config)
	if err != nil {
		return Endpoint{}, err
	}
	server, err := transport.HTTP()
	if err != nil {
		return Endpoint{}, err
	}
	if err = ctx.Err(); err != nil {
		return Endpoint{}, err
	}
	if wrapper, ok := config.Source.(interface {
		WrapHTTP(http.Handler) http.Handler
	}); ok {
		server.Handler = wrapper.WrapHTTP(server.Handler)
	}
	var status func() (string, int64)
	if source, ok := config.Source.(interface{ RuntimeStatus() (string, int64) }); ok {
		status = source.RuntimeStatus
	}
	entry := &instance{endpoint: endpoint, requestedPort: port, server: server, done: make(chan struct{}), cleanup: cleanup, reload: reload, status: status}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return Endpoint{}, fmt.Errorf("namespace MCP manager is closed")
	}
	m.entries[id] = entry
	registered = true
	go func() {
		defer close(entry.done)
		_ = server.Serve(listener)
		m.mu.Lock()
		if m.entries[id] == entry {
			delete(m.entries, id)
		}
		m.mu.Unlock()
	}()
	m.mu.Unlock()
	if previous != nil {
		drainCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := stop(drainCtx, previous); err != nil {
			return endpoint, fmt.Errorf("new namespace endpoint is active; previous listener drain failed: %w", err)
		}
	}
	return endpoint, nil
}
func (m *Manager) Get(id string) (Endpoint, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := m.entries[id]
	if entry == nil {
		return Endpoint{}, false
	}
	return entry.endpoint, true
}
func stop(ctx context.Context, entry *instance) error {
	if entry == nil {
		return nil
	}
	err := entry.server.Shutdown(ctx)
	if err != nil {
		err = errors.Join(err, entry.server.Close())
	}
	select {
	case <-entry.done:
	case <-ctx.Done():
		err = errors.Join(err, ctx.Err())
	}
	if entry.cleanup != nil {
		err = errors.Join(err, entry.cleanup(ctx))
	}
	return err
}
func (m *Manager) Reload(ctx context.Context, id string, generation int64) error {
	if generation < 0 {
		return fmt.Errorf("generation must be zero or positive")
	}
	lock := m.operation(id)
	lock.Lock()
	defer lock.Unlock()
	m.mu.Lock()
	entry := m.entries[id]
	m.mu.Unlock()
	if entry == nil || entry.reload == nil {
		return fmt.Errorf("namespace runtime reload is unavailable")
	}
	return entry.reload(ctx, generation)
}
func (m *Manager) Stop(ctx context.Context, id string) error {
	lock := m.operation(id)
	lock.Lock()
	defer lock.Unlock()
	m.mu.Lock()
	entry := m.entries[id]
	delete(m.entries, id)
	m.mu.Unlock()
	return stop(ctx, entry)
}
func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	m.closed = true
	entries := m.entries
	m.entries = map[string]*instance{}
	m.mu.Unlock()
	var result error
	for _, entry := range entries {
		result = errors.Join(result, stop(ctx, entry))
	}
	return result
}

func (m *Manager) RuntimeStatus(id string) (Endpoint, string, int64, bool) {
	m.mu.Lock()
	entry := m.entries[id]
	m.mu.Unlock()
	if entry == nil || entry.status == nil {
		return Endpoint{}, "", 0, false
	}
	mode, revision := entry.status()
	return entry.endpoint, mode, revision, true
}
