package reader

import (
	json "encoding/json"
	time "time"
)

// Connector is generated canonical view metadata for connector.
type Connector struct {
	SecretConfigured  bool            `sqlx:"secret_configured"`
	OptionsJson       json.RawMessage `sqlx:"options_json,enc=JSON"`
	Name              *string         `sqlx:"name"`
	Driver            *string         `sqlx:"driver"`
	Description       *string         `sqlx:"description"`
	OwnerId           *string         `sqlx:"owner_id"`
	Status            *string         `sqlx:"status"`
	LastTestStatus    *string         `sqlx:"last_test_status"`
	LastTestErrorCode *string         `sqlx:"last_test_error_code"`
	LastTestedAt      *time.Time      `sqlx:"last_tested_at"`
	Etag              *int            `sqlx:"etag"`
	CreatedAt         *time.Time      `sqlx:"created_at"`
	UpdatedAt         *time.Time      `sqlx:"updated_at"`
}
