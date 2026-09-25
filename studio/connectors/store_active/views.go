package store_active

import (
	json "encoding/json"
)

// StoredConnector is generated canonical view metadata for connector.
type StoredConnector struct {
	Name        string          `sqlx:"name"`
	Driver      string          `sqlx:"driver"`
	SecretRef   string          `sqlx:"secret_ref"`
	OptionsJson json.RawMessage `sqlx:"options_json,enc=JSON"`
	DsnTemplate *string         `sqlx:"dsn_template"`
}
