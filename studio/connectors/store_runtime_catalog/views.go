package store_runtime_catalog

import (
	json "encoding/json"
	time "time"
)

// StoredConnector is generated canonical view metadata for connector.
type StoredConnector struct {
	Name              string          `sqlx:"name"`
	Driver            string          `sqlx:"driver"`
	DsnTemplate       *string         `sqlx:"dsn_template"`
	SecretRef         *string         `sqlx:"secret_ref"`
	Description       *string         `sqlx:"description"`
	OwnerId           string          `sqlx:"owner_id"`
	Status            string          `sqlx:"status"`
	OptionsJson       json.RawMessage `sqlx:"options_json,enc=JSON"`
	LastTestStatus    *string         `sqlx:"last_test_status"`
	LastTestErrorCode *string         `sqlx:"last_test_error_code"`
	LastTestedAt      *time.Time      `sqlx:"last_tested_at"`
	Etag              int64           `sqlx:"etag"`
	CreatedAt         time.Time       `sqlx:"created_at"`
	UpdatedAt         time.Time       `sqlx:"updated_at"`
	IsLive            bool            `sqlx:"is_live"`
}
