// Package namespacemcp owns independent Datly MCP HTTP listeners. Hosts supply
// a namespace-scoped catalog/source and authorization policy for each endpoint.
package namespacemcp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	dexec "github.com/viant/datly/exec"
	mcpserver "github.com/viant/datly/mcp/server"
)

type Endpoint struct {
	NamespaceID string `json:"namespaceId"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
	Path        string `json:"path,omitempty"`
}

func (e Endpoint) URL() string {
	path := e.Path
	if path == "" {
		path = "/mcp"
	}
	return "http://" + e.Address + path
}

// Factory receives the reserved address, allowing OAuth resource metadata to
// use the actual port. The host must restrict the source to this namespace.
type Factory func(context.Context, Endpoint) (mcpserver.Config, error)

// HTTPFactory supplies an already scoped and protected MCP handler. It lets
// embedding applications reuse listener lifecycle without a Datly catalog.
// The handler must enforce authentication and namespace authorization itself.
// Optional Close/Reload/RuntimeStatus/RuntimeConnection hooks are honored just
// as they are for a Datly source.
type HTTPFactory func(context.Context, Endpoint) (http.Handler, error)

type instance struct {
	endpoint      Endpoint
	requestedPort int
	server        *http.Server
	native        http.Handler
	done          chan struct{}
	cleanup       func(context.Context) error
	reload        func(context.Context, int64) error
	status        func() (string, int64)
	connection    func(context.Context) (json.RawMessage, error)
	invalidate    func(context.Context, string, int, string, string) ([]dexec.CacheInvalidation, error)
	shared        bool
	requests      sync.WaitGroup
	cancel        context.CancelFunc
	lifetime      context.Context
	drain         sync.Once
	drained       chan struct{}
}

type Manager struct {
	mu            sync.Mutex
	entries       map[string]*instance
	operations    map[string]*sync.Mutex
	closed        bool
	factory       Factory
	httpFactory   HTTPFactory
	sharedServer  *http.Server
	sharedAddress string
	sharedPort    int
	sharedActive  bool
}

func New(factory Factory) (*Manager, error) {
	if factory == nil {
		return nil, fmt.Errorf("namespace MCP factory is required")
	}
	return &Manager{factory: factory, entries: map[string]*instance{}, operations: map[string]*sync.Mutex{}}, nil
}

// NewHTTP reuses namespace listener lifecycle with host-owned MCP handlers.
func NewHTTP(factory HTTPFactory) (*Manager, error) {
	if factory == nil {
		return nil, fmt.Errorf("namespace MCP HTTP factory is required")
	}
	return &Manager{httpFactory: factory, entries: map[string]*instance{}, operations: map[string]*sync.Mutex{}}, nil
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
	return m.start(ctx, id, port, false, nil)
}

// Rebind constructs and reserves the new endpoint before replacing the old one.
// Failures before commitment preserve the current listener. A returned endpoint
// remains active even if an error reports failure to drain the previous server.
func (m *Manager) Rebind(ctx context.Context, id string, port int) (Endpoint, error) {
	return m.start(ctx, id, port, true, nil)
}

// RebindWithCommit reserves and constructs the replacement before committing
// caller-owned state. A failed commit preserves the old endpoint and releases
// the candidate. The callback also runs for an unchanged endpoint. It must not
// call Manager methods. As with Rebind, drain failure can follow activation.
func (m *Manager) RebindWithCommit(ctx context.Context, id string, port int, commit func(Endpoint) error) (Endpoint, error) {
	if commit == nil {
		return Endpoint{}, fmt.Errorf("namespace MCP commit callback is required")
	}
	return m.start(ctx, id, port, true, commit)
}

func (m *Manager) start(ctx context.Context, id string, port int, replace bool, commit func(Endpoint) error) (Endpoint, error) {
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
		if !replace && previous.requestedPort == port || replace && previous.requestedPort == port && previous.shared || replace && port != 0 && previous.endpoint.Port == port && !previous.shared {
			if commit != nil {
				err := func() error {
					defer m.mu.Unlock()
					return commitEndpoint(ctx, previous.endpoint, commit)
				}()
				if err != nil {
					return Endpoint{}, err
				}
				return previous.endpoint, nil
			}
			m.mu.Unlock()
			return previous.endpoint, nil
		}
		if !replace {
			m.mu.Unlock()
			return Endpoint{}, fmt.Errorf("use Rebind to change a namespace MCP port")
		}
	}
	m.mu.Unlock()
	useShared := m.sharedServer != nil && port == 0
	var listener net.Listener
	endpoint := Endpoint{NamespaceID: id}
	if useShared {
		m.mu.Lock()
		active := m.sharedActive
		m.mu.Unlock()
		if !active {
			return Endpoint{}, fmt.Errorf("shared MCP listener is unavailable")
		}
		endpoint.Address = m.sharedAddress
		endpoint.Port = m.sharedPort
		endpoint.Path = "/namespaces/" + id + "/mcp"
	} else {
		listener, err = net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err != nil {
			return Endpoint{}, fmt.Errorf("reserve namespace MCP port: %w", err)
		}
		endpoint.Address = listener.Addr().String()
		endpoint.Port = listener.Addr().(*net.TCPAddr).Port
	}
	registered := false
	defer func() {
		if !registered && listener != nil {
			_ = listener.Close()
		}
	}()
	var config mcpserver.Config
	var handler http.Handler
	var owner any
	if m.httpFactory != nil {
		handler, err = m.httpFactory(ctx, endpoint)
		if err == nil && handler == nil {
			err = fmt.Errorf("namespace MCP HTTP factory returned no handler")
		}
		owner = handler
	} else {
		config, err = m.factory(ctx, endpoint)
		owner = config.Source
	}
	if err != nil {
		return Endpoint{}, err
	}
	var cleanup func(context.Context) error
	var reload func(context.Context, int64) error
	if source, ok := owner.(interface{ Close(context.Context) error }); ok {
		cleanup = source.Close
	}
	if source, ok := owner.(interface {
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
	var server *http.Server
	if m.httpFactory != nil {
		server = &http.Server{Addr: endpoint.Address, Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	} else {
		config.Transport.Kind = mcpserver.TransportStreamable
		config.Transport.Address = endpoint.Address
		transport, e := mcpserver.New(config)
		if e != nil {
			return Endpoint{}, e
		}
		server, err = transport.HTTP()
		if err != nil {
			return Endpoint{}, err
		}
	}
	if err = ctx.Err(); err != nil {
		return Endpoint{}, err
	}
	var native http.Handler
	if wrapper, ok := owner.(interface {
		WrapNativeHTTP(http.Handler) http.Handler
	}); ok {
		native = wrapper.WrapNativeHTTP(server.Handler)
	}
	if wrapper, ok := owner.(interface {
		WrapHTTP(http.Handler) http.Handler
	}); ok {
		server.Handler = wrapper.WrapHTTP(server.Handler)
	}
	var status func() (string, int64)
	if source, ok := owner.(interface{ RuntimeStatus() (string, int64) }); ok {
		status = source.RuntimeStatus
	}
	var connection func(context.Context) (json.RawMessage, error)
	if source, ok := owner.(interface {
		RuntimeConnection(context.Context) (json.RawMessage, error)
	}); ok {
		connection = source.RuntimeConnection
	}
	var invalidate func(context.Context, string, int, string, string) ([]dexec.CacheInvalidation, error)
	if source, ok := owner.(interface {
		InvalidatePublishedCacheView(context.Context, string, int, string, string) ([]dexec.CacheInvalidation, error)
	}); ok {
		invalidate = source.InvalidatePublishedCacheView
	}
	entry := &instance{endpoint: endpoint, requestedPort: port, server: server, done: make(chan struct{}), cleanup: cleanup, reload: reload, status: status, connection: connection, invalidate: invalidate, shared: useShared, native: native, drained: make(chan struct{})}
	entry.lifetime, entry.cancel = context.WithCancel(context.Background())
	err = func() error {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.closed {
			return fmt.Errorf("namespace MCP manager is closed")
		}
		if err := commitEndpoint(ctx, endpoint, commit); err != nil {
			return err
		}
		m.entries[id] = entry
		registered = true
		if !useShared {
			go func() {
				defer close(entry.done)
				_ = server.Serve(listener)
				m.mu.Lock()
				if m.entries[id] == entry {
					delete(m.entries, id)
				}
				m.mu.Unlock()
			}()
		}
		return nil
	}()
	if err != nil {
		return Endpoint{}, err
	}
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
	entry.cancel()
	entry.drain.Do(func() { go func() { entry.requests.Wait(); close(entry.drained) }() })
	if entry.shared {
		return drainAndCleanup(ctx, entry)
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
	return errors.Join(err, drainAndCleanup(ctx, entry))
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
	return m.stopManaged(ctx, id, nil)
}

// StopWithCommit commits caller-owned withdrawal state before removing the
// endpoint. A failed callback leaves it active; a later drain error does not
// undo the committed withdrawal. The callback must not call Manager methods.
func (m *Manager) StopWithCommit(ctx context.Context, id string, commit func(Endpoint) error) error {
	if commit == nil {
		return fmt.Errorf("namespace MCP commit callback is required")
	}
	return m.stopManaged(ctx, id, commit)
}

func (m *Manager) stopManaged(ctx context.Context, id string, commit func(Endpoint) error) error {
	lock := m.operation(id)
	lock.Lock()
	defer lock.Unlock()
	var entry *instance
	err := func() error {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.closed && commit != nil {
			return fmt.Errorf("namespace MCP manager is closed")
		}
		entry = m.entries[id]
		endpoint := Endpoint{NamespaceID: id}
		if entry != nil {
			endpoint = entry.endpoint
		}
		if err := commitEndpoint(ctx, endpoint, commit); err != nil {
			return err
		}
		delete(m.entries, id)
		return nil
	}()
	if err != nil {
		return err
	}
	return stop(ctx, entry)
}

func commitEndpoint(ctx context.Context, endpoint Endpoint, commit func(Endpoint) error) error {
	if commit == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return commit(endpoint)
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
	if m.sharedServer != nil {
		err := m.sharedServer.Shutdown(ctx)
		if err != nil {
			err = errors.Join(err, m.sharedServer.Close())
		}
		result = errors.Join(result, err)
	}
	return result
}

func (m *Manager) RuntimeStatus(id string) (Endpoint, string, int64, bool) {
	m.mu.Lock()
	entry := m.entries[id]
	m.mu.Unlock()
	if entry == nil || entry.status == nil || entry.shared && !m.sharedAvailable() {
		return Endpoint{}, "", 0, false
	}
	mode, revision := entry.status()
	return entry.endpoint, mode, revision, true
}

// Connection returns only the deployed source's configuration references. The
// caller owns visibility authorization; this is used behind the deployment admin
// token, never as an unauthenticated directory.
func (m *Manager) Connection(ctx context.Context, id string) (json.RawMessage, error) {
	decoded, err := hex.DecodeString(id)
	if err != nil || len(decoded) != 32 || strings.ToLower(id) != id {
		return nil, fmt.Errorf("canonical namespace ID is required")
	}
	lock := m.operation(id)
	lock.Lock()
	defer lock.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	entry, closed := m.entries[id], m.closed
	m.mu.Unlock()
	if closed || entry == nil || entry.connection == nil || entry.shared && !m.sharedAvailable() {
		return nil, fmt.Errorf("namespace connection is unavailable")
	}
	return entry.connection(ctx)
}

// InvalidatePublishedCacheView invokes only the current namespace source's
// trusted control capability. It never sends an MCP data call or trusts a
// namespace ID from the source implementation.
func (m *Manager) InvalidatePublishedCacheView(ctx context.Context, id, reportID string, version int, routePath, view string) ([]dexec.CacheInvalidation, error) {
	decoded, err := hex.DecodeString(id)
	if err != nil || len(decoded) != 32 || id != strings.ToLower(id) {
		return nil, fmt.Errorf("canonical namespace ID is required")
	}
	lock := m.operation(id)
	lock.Lock()
	defer lock.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	entry, closed := m.entries[id], m.closed
	m.mu.Unlock()
	if closed || entry == nil || entry.invalidate == nil || entry.shared && !m.sharedAvailable() {
		return nil, fmt.Errorf("namespace cache administration is unavailable")
	}
	return entry.invalidate(ctx, reportID, version, routePath, view)
}

// drainAndCleanup prevents source disposal while an embedded request still owns it.
func drainAndCleanup(ctx context.Context, entry *instance) error {
	select {
	case <-entry.drained:
		if entry.cleanup != nil {
			return entry.cleanup(ctx)
		}
		return nil
	case <-ctx.Done():
		go func() {
			<-entry.drained
			if entry.cleanup != nil {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = entry.cleanup(cleanupCtx)
			}
		}()
		return ctx.Err()
	}
}

// NativeHandler forwards to the currently active namespace transport without
// invoking host composition middleware. Only sources explicitly exposing a
// protected native handler support this API. Authentication remains mandatory
// according to that source's configuration. No runtime or listener is created.
func (m *Manager) NativeHandler(id string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		entry := m.entries[id]
		if m.closed || entry == nil || entry.native == nil {
			m.mu.Unlock()
			http.Error(w, "namespace runtime unavailable", http.StatusServiceUnavailable)
			return
		}
		entry.requests.Add(1)
		m.mu.Unlock()
		defer entry.requests.Done()
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		detach := context.AfterFunc(entry.lifetime, cancel)
		defer detach()
		entry.native.ServeHTTP(w, r.Clone(ctx))
	})
}
