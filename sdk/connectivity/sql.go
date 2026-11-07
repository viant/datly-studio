// Package connectivity contains host-neutral connector probes.
package connectivity

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/connectorinit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/constant"
)

// SQLProbe checks relational connectors with database/sql. The host must link
// the requested driver; no credential or driver implementation is embedded in
// Studio.
type SQLProbe struct {
	Secrets connectorsecret.Resolver
	Const   *constant.Values
}

func (p SQLProbe) Probe(ctx context.Context, connector *sdk.Connector) (result *sdk.ConnectorTestResult, probeErr error) {
	started := time.Now()
	stage, destination := "configuration", ""
	var details probeDetails
	defer func() {
		logProbe(ctx, connector, stage, destination, started, probeErr, details)
	}()
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
	stage = "secret"
	dsn, err := connectorsecret.Resolve(ctx, p.Secrets, connector.DSNTemplate, connector.SecretRef, p.Const)
	if err != nil {
		return nil, probeFailure("secret", err)
	}
	if strings.TrimSpace(dsn) == "" {
		return nil, &ProbeError{Code: "dsn_missing", Message: "The connector has no DSN. Enter a DSN or a complete DSN secret."}
	}
	destination = probeDestination(driver, dsn)
	details.ExpandedDSNFingerprint = expandedProbeFingerprint(dsn)
	stage = "open"
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, probeFailure("open", err)
	}
	defer db.Close()
	stage = "initialize"
	if err = connectorinit.Configure(ctx, db, driver, connector.Options); err != nil {
		return nil, probeFailure("initialize", err)
	}
	stage = "ping"
	if driver == "mysql" {
		details.Backend, err = probeMySQL(ctx, db)
	} else {
		err = db.PingContext(ctx)
	}
	if err != nil {
		return nil, probeFailure("ping", err)
	}
	return &sdk.ConnectorTestResult{Status: "passed"}, nil
}

type mysqlProbeBackend struct {
	Host, Version string
	ServerID      uint64
	Database      sql.NullString
	ErrorType     string
}

func probeMySQL(ctx context.Context, db *sql.DB) (*mysqlProbeBackend, error) {
	connection, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	if err = connection.PingContext(ctx); err != nil {
		return nil, err
	}

	// Read the server identity reported through the same authenticated session.
	// A proxy may answer the ping itself and route the diagnostic query onward.
	// Failure to read the identity does not change the connectivity result.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var backend mysqlProbeBackend
	err = connection.QueryRowContext(ctx, "SELECT @@hostname, @@server_id, VERSION(), DATABASE()").Scan(&backend.Host, &backend.ServerID, &backend.Version, &backend.Database)
	if err != nil {
		return &mysqlProbeBackend{ErrorType: fmt.Sprintf("%T", err)}, nil
	}
	return &backend, nil
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
