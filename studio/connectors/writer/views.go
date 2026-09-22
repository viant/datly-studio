package writer

import (
	json "encoding/json"
	time "time"
)

// Connector is generated canonical view metadata for connector.
type Connector struct {
	OptionsJson json.RawMessage `sqlx:"options_json,enc=JSON"`
	Name *string `sqlx:"name,primaryKey" validate:"required"`
	Driver *string `validate:"required,choice(mysql,viant/bigquery,pg,sqlite,viant/aerospike)" invariant:"Connection" sqlx:"driver"`
	DsnTemplate *string `validate:"datly_connector_dsn(Driver,SecretRef)" invariant:"Connection" sqlx:"dsn_template"`
	SecretRef *string `invariant:"Connection" sqlx:"secret_ref"`
	ShouldDelete bool `sqlx:"-" writer:"delete"`
	Etag *int `writer:"concurrency" sqlx:"etag"`
	Description *string `sqlx:"description"`
	OwnerId *string `sqlx:"owner_id"`
	Status *string `sqlx:"status"`
	LastTestStatus *string `sqlx:"last_test_status"`
	LastTestErrorCode *string `sqlx:"last_test_error_code"`
	LastTestedAt *time.Time `sqlx:"last_tested_at"`
	CreatedAt *time.Time `sqlx:"created_at"`
	UpdatedAt *time.Time `sqlx:"updated_at"`
	DeletedAt *time.Time `sqlx:"deleted_at"`
	Has *ConnectorHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ConnectorHas"`
}

type ConnectorHas struct {
	OptionsJson bool
	Name bool
	Driver bool
	DsnTemplate bool
	SecretRef bool
	ShouldDelete bool
	Etag bool
	Description bool
	OwnerId bool
	Status bool
	LastTestStatus bool
	LastTestErrorCode bool
	LastTestedAt bool
	CreatedAt bool
	UpdatedAt bool
	DeletedAt bool
}

// CurrentConnectorView is generated canonical view metadata for connector.
type CurrentConnectorView struct {
	OptionsJson json.RawMessage `sqlx:"options_json,enc=JSON"`
	Name *string `sqlx:"name,primaryKey" validate:"required"`
	Driver *string `validate:"required,choice(mysql,viant/bigquery,pg,sqlite,viant/aerospike)" invariant:"Connection" sqlx:"driver"`
	DsnTemplate *string `validate:"datly_connector_dsn(Driver,SecretRef)" invariant:"Connection" sqlx:"dsn_template"`
	SecretRef *string `invariant:"Connection" sqlx:"secret_ref"`
	Etag *int `sqlx:"etag"`
	Description *string `sqlx:"description"`
	OwnerId *string `sqlx:"owner_id"`
	Status *string `sqlx:"status"`
	LastTestStatus *string `sqlx:"last_test_status"`
	LastTestErrorCode *string `sqlx:"last_test_error_code"`
	LastTestedAt *time.Time `sqlx:"last_tested_at"`
	CreatedAt *time.Time `sqlx:"created_at"`
	UpdatedAt *time.Time `sqlx:"updated_at"`
	DeletedAt *time.Time `sqlx:"deleted_at"`
}

type ConnectorKeysRow struct {
	Name *string `sqlx:"name"`
}
