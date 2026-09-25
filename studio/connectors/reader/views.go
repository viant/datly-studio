package reader

import (
	json "encoding/json"
	time "time"
)

// Connector is generated canonical view metadata for connector.
type Connector struct {
	Name              string          `sqlx:"name"`
	Driver            string          `sqlx:"driver"`
	DsnConfigured     bool            `sqlx:"dsn_configured"`
	SecretConfigured  bool            `sqlx:"secret_configured"`
	Description       *string         `json:"description,omitempty" sqlx:"description"`
	OwnerId           string          `sqlx:"owner_id"`
	Status            string          `sqlx:"status"`
	OptionsJson       json.RawMessage `json:"options,omitempty" sqlx:"options_json,enc=JSON"`
	LastTestStatus    *string         `json:"lastTestStatus,omitempty" sqlx:"last_test_status"`
	LastTestErrorCode *string         `json:"lastTestErrorCode,omitempty" sqlx:"last_test_error_code"`
	LastTestedAt      *time.Time      `json:"lastTestedAt,omitempty" sqlx:"last_tested_at"`
	Etag              int64           `sqlx:"etag"`
	CreatedAt         time.Time       `sqlx:"created_at"`
	UpdatedAt         time.Time       `sqlx:"updated_at"`
}
