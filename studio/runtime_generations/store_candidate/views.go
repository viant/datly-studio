package store_candidate

// PublishedDefinition is generated canonical view metadata for definition.
type PublishedDefinition struct {
	ReportId             string  `sqlx:"report_id"`
	VersionNo            int     `sqlx:"version_no"`
	ComponentScope       string  `sqlx:"component_scope"`
	ComponentName        string  `sqlx:"component_name"`
	DefaultConnectorName string  `sqlx:"default_connector_name"`
	Driver               string  `sqlx:"driver"`
	SecretRef            string  `sqlx:"secret_ref"`
	GeneratedDql         string  `sqlx:"generated_dql"`
	AuthoredDql          string  `sqlx:"authored_dql"`
	DsnTemplate          *string `sqlx:"dsn_template"`
}
