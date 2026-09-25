package store_preview_scoped

import (
	json "encoding/json"
)

// PreviewConnector is generated canonical view metadata for connector.
type PreviewConnector struct {
	Name        string          `sqlx:"name"`
	Driver      string          `sqlx:"driver"`
	SecretRef   string          `sqlx:"secret_ref"`
	OptionsJson json.RawMessage `sqlx:"options_json,enc=JSON"`
	DsnTemplate *string         `sqlx:"dsn_template"`
}
