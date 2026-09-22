// Package connectivity contains host-neutral connector probes.
package connectivity

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/viant/datly-studio/internal/connectorinit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/sdk"
)

// SQLProbe checks relational connectors with database/sql. The host must link
// the requested driver; no credential or driver implementation is embedded in
// Studio.
type SQLProbe struct{ Secrets connectorsecret.Resolver }

func (p SQLProbe) Probe(ctx context.Context, connector *sdk.Connector) (*sdk.ConnectorTestResult, error) {
	if connector == nil {
		return nil, fmt.Errorf("connector is required")
	}
	driver := resolveDriver(strings.ToLower(strings.TrimSpace(connector.Driver)))
	switch driver {
	case "sqlite", "sqlite3", "mysql", "postgres", "pgx", "bigquery":
	default:
		return nil, fmt.Errorf("unsupported SQL connector driver %q", connector.Driver)
	}
	dsn, err := connectorsecret.Resolve(ctx, p.Secrets, connector.DSNTemplate, connector.SecretRef)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("connector %q has no DSN template", connector.Name)
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err = connectorinit.Configure(ctx, db, driver, connector.Options); err != nil {
		return nil, err
	}
	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}
	return &sdk.ConnectorTestResult{Status: "passed"}, nil
}

func resolveDriver(driver string) string {
	switch driver {
	case "viant/bigquery":
		return "bigquery"
	case "pg", "pgx":
		return "postgres"
	}
	if driver != "sqlite" && driver != "sqlite3" {
		return driver
	}
	available := map[string]bool{}
	for _, candidate := range sql.Drivers() {
		available[candidate] = true
	}
	if available["sqlite3"] {
		return "sqlite3"
	}
	return "sqlite"
}
