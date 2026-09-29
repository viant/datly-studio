package host

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/viant/datly-studio/runtime/namespacemcp"
)

// NamespaceGroup owns the persisted-namespace MCP mode. It starts no global
// MCP listener, so a namespace catalog cannot be reached through an unscoped port.
type NamespaceGroup struct {
	DB          *sql.DB
	Listeners   *namespacemcp.Manager
	reconciler  *namespacemcp.Reconciler
	adminServer *http.Server
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
	manager, err := namespacemcp.New(NamespaceFactory(config))
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
	return g.reconciler.Apply(ctx)
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
