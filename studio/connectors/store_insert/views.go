package store_insert

import (
	json "encoding/json"
	time "time"
)

// StoredConnector is generated canonical view metadata for connector.
type StoredConnector struct {
	Name        string              `sqlx:"name,primaryKey"`
	Driver      string              `sqlx:"driver"`
	DsnTemplate *string             `sqlx:"dsn_template"`
	SecretRef   *string             `sqlx:"secret_ref"`
	Description *string             `sqlx:"description"`
	OwnerId     string              `sqlx:"owner_id"`
	Status      string              `sqlx:"status"`
	OptionsJson json.RawMessage     `sqlx:"options_json,enc=JSON"`
	Etag        int64               `sqlx:"etag"`
	CreatedAt   time.Time           `sqlx:"created_at"`
	UpdatedAt   time.Time           `sqlx:"updated_at"`
	Has         *StoredConnectorHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredConnectorHas"`
}

type StoredConnectorHas struct {
	Name        bool
	Driver      bool
	DsnTemplate bool
	SecretRef   bool
	Description bool
	OwnerId     bool
	Status      bool
	OptionsJson bool
	Etag        bool
	CreatedAt   bool
	UpdatedAt   bool
}
