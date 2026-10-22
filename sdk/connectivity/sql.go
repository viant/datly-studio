// Package connectivity contains host-neutral connector probes.
package connectivity

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/datly/constant"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/connectorinit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/sdk"
)

// SQLProbe checks relational connectors with database/sql. The host must link
// the requested driver; no credential or driver implementation is embedded in
// Studio.
type SQLProbe struct {
	Secrets connectorsecret.Resolver
	Const   *constant.Values
}

func (p SQLProbe) Probe(ctx context.Context, connector *sdk.Connector) (*sdk.ConnectorTestResult, error) {
	if connector == nil {
		return nil, fmt.Errorf("connector is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	driver := resolveDriver(strings.ToLower(strings.TrimSpace(connector.Driver)))
	linked := false
	for _, candidate := range sql.Drivers() {
		if candidate == driver {
			linked = true
			break
		}
	}
	if !linked {
		return nil, &ProbeError{Code: "driver_not_linked", Message: "The selected SQL driver is not linked in this server executable."}
	}
	dsn, err := connectorsecret.Resolve(ctx, p.Secrets, connector.DSNTemplate, connector.SecretRef, p.Const)
	if err != nil {
		return nil, probeFailure("secret", err)
	}
	if strings.TrimSpace(dsn) == "" {
		return nil, &ProbeError{Code: "dsn_missing", Message: "The connector has no DSN. Enter a DSN or a complete DSN secret."}
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, probeFailure("open", err)
	}
	defer db.Close()
	if err = connectorinit.Configure(ctx, db, driver, connector.Options); err != nil {
		return nil, probeFailure("initialize", err)
	}
	if err = db.PingContext(ctx); err != nil {
		return nil, probeFailure("ping", err)
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
	// Honor the connector's requested implementation when both names are linked.
	// In particular, a transitive sqlite3 import may be only a CGO-disabled stub.
	if available[driver] {
		return driver
	}
	if available["sqlite"] {
		return "sqlite"
	}
	return "sqlite3"
}
