package store_status

import (
	time "time"
)

// StoredConnector is generated canonical view metadata for connector.
type StoredConnector struct {
	Name              string              `sqlx:"name,primaryKey"`
	Status            string              `sqlx:"status"`
	Etag              *int64              `writer:"concurrency" sqlx:"etag"`
	UpdatedAt         *time.Time          `sqlx:"updated_at"`
	LastTestStatus    *string             `sqlx:"last_test_status"`
	LastTestErrorCode *string             `sqlx:"last_test_error_code"`
	LastTestedAt      *time.Time          `sqlx:"last_tested_at"`
	DeletedAt         *time.Time          `sqlx:"deleted_at"`
	Has               *StoredConnectorHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredConnectorHas"`
}

type StoredConnectorHas struct {
	Name              bool
	Status            bool
	Etag              bool
	UpdatedAt         bool
	LastTestStatus    bool
	LastTestErrorCode bool
	LastTestedAt      bool
	DeletedAt         bool
}

// CurrentConnectorView is generated canonical view metadata for connector.
type CurrentConnectorView struct {
	Name              string     `sqlx:"name,primaryKey"`
	Status            string     `sqlx:"status"`
	Etag              *int64     `sqlx:"etag"`
	UpdatedAt         *time.Time `sqlx:"updated_at"`
	LastTestStatus    *string    `sqlx:"last_test_status"`
	LastTestErrorCode *string    `sqlx:"last_test_error_code"`
	LastTestedAt      *time.Time `sqlx:"last_tested_at"`
	DeletedAt         *time.Time `sqlx:"deleted_at"`
}

type ConnectorKeysRow struct {
	Name string `sqlx:"name"`
}
