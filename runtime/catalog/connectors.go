package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/viant/datly-studio/sdk"
)

type ConnectorCatalog struct{ db *sql.DB }

func NewConnectorCatalog(db *sql.DB) *ConnectorCatalog { return &ConnectorCatalog{db: db} }

func (c *ConnectorCatalog) LoadActiveConnectors(ctx context.Context) ([]*sdk.Connector, error) {
	rows, err := c.db.QueryContext(ctx, `
SELECT name, driver, dsn_template, description, owner_id, status, secret_ref, options_json, last_test_status, last_test_error_code, last_tested_at, etag, created_at, updated_at
FROM connectors WHERE deleted_at IS NULL AND status = 'active' ORDER BY updated_at DESC, name ASC`)
	if err != nil {
		return nil, fmt.Errorf("load active connectors: %w", err)
	}
	defer rows.Close()
	var result []*sdk.Connector
	for rows.Next() {
		connector, err := scanCatalogConnector(rows)
		if err != nil {
			return nil, fmt.Errorf("load active connectors scan: %w", err)
		}
		result = append(result, connector)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanCatalogConnector(scanner interface{ Scan(dest ...any) error }) (*sdk.Connector, error) {
	var value sdk.Connector
	var dsn, description, secretRef, options, testStatus, testCode sql.NullString
	var testedAt sql.NullTime
	if err := scanner.Scan(&value.Name, &value.Driver, &dsn, &description, &value.OwnerID, &value.Status, &secretRef, &options, &testStatus, &testCode, &testedAt, &value.ETag, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return nil, err
	}
	value.DSNTemplate, value.DSNConfigured, value.Description, value.SecretRef, value.LastTestStatus, value.LastTestErrorCode = dsn.String, strings.TrimSpace(dsn.String) != "", description.String, secretRef.String, testStatus.String, testCode.String
	if options.Valid && strings.TrimSpace(options.String) != "" {
		value.Options = json.RawMessage(options.String)
	}
	if testedAt.Valid {
		timestamp := testedAt.Time
		value.LastTestedAt = &timestamp
	}
	return &value, nil
}
