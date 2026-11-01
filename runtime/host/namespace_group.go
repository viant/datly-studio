package host

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/viant/datly-studio/runtime/namespacemcp"
)

// NamespaceGroup owns the persisted-namespace MCP mode. It starts no global
// catalog. Shared listeners route only through explicit namespace paths.
type NamespaceGroup struct {
	DB          *sql.DB
	Listeners   *namespacemcp.Manager
	reconciler  *namespacemcp.Reconciler
	adminServer *http.Server
	failureMu   sync.RWMutex
	failures    map[string]error
}

func NewNamespaceGroup(ctx context.Context, config Config) (*NamespaceGroup, error) {
	if config.NamespaceID != "" {
		return nil, fmt.Errorf("namespace group configuration cannot pin one namespace")
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	db, err := sql.Open(config.Studio.Driver, config.Studio.DSN)
	if err != nil {
		return nil, err
	}
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	var manager *namespacemcp.Manager
	if config.NamespaceMCPMode == "shared" {
		manager, err = namespacemcp.NewShared(NamespaceFactory(config), config.MCP.Address)
	} else {
		manager, err = namespacemcp.New(NamespaceFactory(config))
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return &NamespaceGroup{DB: db, Listeners: manager, reconciler: &namespacemcp.Reconciler{Manager: manager, Definitions: namespacemcp.SQLDefinitions{DB: db}}}, nil
}
func (g *NamespaceGroup) StartAdmin(address, token string) error {
	if token == "" {
		return fmt.Errorf("namespace runtime admin token is required")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	server := hardenedHTTPServer(listener.Addr().String(), g.AdminHandler(token))
	g.adminServer = server
	go func() { _ = server.Serve(listener) }()
	return nil
}
func (g *NamespaceGroup) Reconcile(ctx context.Context) ([]namespacemcp.Change, error) {
	changes, err := g.reconciler.Apply(ctx)
	// Preserve the last known failures if reading the desired state itself failed.
	if changes != nil || err == nil {
		failures := make(map[string]error)
		for _, change := range changes {
			if change.Cause != nil {
				failures[change.NamespaceID] = change.Cause
			}
		}
		g.failureMu.Lock()
		g.failures = failures
		g.failureMu.Unlock()
	}
	return changes, err
}

// RuntimeFailure exposes the last reconciliation cause to trusted host control.
// Public callers must project it through the existing diagnostic policy.
func (g *NamespaceGroup) RuntimeFailure(namespaceID string) error {
	g.failureMu.RLock()
	defer g.failureMu.RUnlock()
	return g.failures[namespaceID]
}

// Run polls configuration after startup. Each read completes before listener
// changes, and transient failures leave existing namespace listeners available.
func (g *NamespaceGroup) Run(ctx context.Context, interval time.Duration, observe func([]namespacemcp.Change, error)) error {
	if interval <= 0 {
		return fmt.Errorf("namespace reconciliation interval must be positive")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return nil
		}
		changes, err := g.Reconcile(ctx)
		if observe != nil {
			observe(changes, err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (g *NamespaceGroup) Close(ctx context.Context) error {
	var adminErr error
	if g.adminServer != nil {
		adminErr = g.adminServer.Shutdown(ctx)
		if adminErr != nil {
			adminErr = errors.Join(adminErr, g.adminServer.Close())
		}
	}
	return errors.Join(adminErr, g.Listeners.Close(ctx), g.DB.Close())
}

// AdminAddress returns the reserved process-owned admin address after StartAdmin.
// Startup owns this value; callers must not race startup and shutdown.
func (g *NamespaceGroup) AdminAddress() string {
	if g == nil || g.adminServer == nil {
		return ""
	}
	return g.adminServer.Addr
}
