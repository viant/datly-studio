package store_config

import (
	json "encoding/json"
	time "time"
)

// StoredConnector is generated canonical view metadata for connector.
type StoredConnector struct {
	Name              string              `sqlx:"name,primaryKey"`
	Driver            string              `sqlx:"driver"`
	DsnTemplate       *string             `sqlx:"dsn_template"`
	SecretRef         *string             `sqlx:"secret_ref"`
	Description       *string             `sqlx:"description"`
	OptionsJson       json.RawMessage     `sqlx:"options_json,enc=JSON"`
	Status            string              `sqlx:"status"`
	LastTestStatus    *string             `sqlx:"last_test_status"`
	LastTestErrorCode *string             `sqlx:"last_test_error_code"`
	LastTestedAt      *time.Time          `sqlx:"last_tested_at"`
	Etag              *int64              `writer:"concurrency" sqlx:"etag"`
	UpdatedAt         *time.Time          `sqlx:"updated_at"`
	DeletedAt         *time.Time          `sqlx:"deleted_at"`
	Has               *StoredConnectorHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredConnectorHas"`
}

type StoredConnectorHas struct {
	Name              bool
	Driver            bool
	DsnTemplate       bool
	SecretRef         bool
	Description       bool
	OptionsJson       bool
	Status            bool
	LastTestStatus    bool
	LastTestErrorCode bool
	LastTestedAt      bool
	Etag              bool
	UpdatedAt         bool
	DeletedAt         bool
}

// CurrentConnectorView is generated canonical view metadata for connector.
type CurrentConnectorView struct {
	Name              string          `sqlx:"name,primaryKey"`
	Driver            string          `sqlx:"driver"`
	DsnTemplate       *string         `sqlx:"dsn_template"`
	SecretRef         *string         `sqlx:"secret_ref"`
	Description       *string         `sqlx:"description"`
	OptionsJson       json.RawMessage `sqlx:"options_json,enc=JSON"`
	Status            string          `sqlx:"status"`
	LastTestStatus    *string         `sqlx:"last_test_status"`
	LastTestErrorCode *string         `sqlx:"last_test_error_code"`
	LastTestedAt      *time.Time      `sqlx:"last_tested_at"`
	Etag              *int64          `sqlx:"etag"`
	UpdatedAt         *time.Time      `sqlx:"updated_at"`
	DeletedAt         *time.Time      `sqlx:"deleted_at"`
}

type ConnectorKeysRow struct {
	Name string `sqlx:"name"`
}
