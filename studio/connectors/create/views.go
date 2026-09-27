package create

import (
	json "encoding/json"
	time "time"
)

// ConnectorRecord is generated canonical view metadata for connector.
type ConnectorRecord struct {
	Name        string              `sqlx:"name,primaryKey" json:"name"`
	Driver      string              `json:"driver" sqlx:"driver"`
	DsnTemplate *string             `json:"dsnTemplate,omitempty" sqlx:"dsn_template"`
	SecretRef   *string             `json:"secretRef,omitempty" sqlx:"secret_ref"`
	Description *string             `json:"description,omitempty" sqlx:"description"`
	OwnerId     string              `json:"ownerId,omitempty" sqlx:"owner_id"`
	Status      string              `json:"-" sqlx:"status"`
	OptionsJson json.RawMessage     `sqlx:"options_json,enc=JSON" json:"options,omitempty"`
	Etag        int64               `json:"-" sqlx:"etag"`
	CreatedAt   time.Time           `json:"-" sqlx:"created_at"`
	UpdatedAt   time.Time           `json:"-" sqlx:"updated_at"`
	Has         *ConnectorRecordHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ConnectorRecordHas"`
}

type ConnectorRecordHas struct {
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
