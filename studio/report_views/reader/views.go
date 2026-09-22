package reader

import (
	json "encoding/json"
)

// ReportView is generated canonical view metadata for view.
type ReportView struct {
	MetadataJson json.RawMessage `sqlx:"metadata_json,enc=JSON"`
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	ViewId *string `sqlx:"view_id"`
	ViewIdentity *string `sqlx:"view_identity"`
	ParentViewId *string `sqlx:"parent_view_id"`
	RelationName *string `sqlx:"relation_name"`
	Name *string `sqlx:"name"`
	Namespace *string `sqlx:"namespace"`
	Role *string `sqlx:"role"`
	Cardinality *string `sqlx:"cardinality"`
	ConnectorName *string `sqlx:"connector_name"`
	SourceKind *string `sqlx:"source_kind"`
	SourceSql *string `sqlx:"source_sql"`
	SourceTable *string `sqlx:"source_table"`
	Fields []*ReportField `view:"fields,type=ReportField,table=report_fields" on:"ReportId:views.report_id=ReportId:fields.report_id,VersionNo:views.version_no=VersionNo:fields.version_no,ViewId:views.view_id=ViewId:fields.view_id" sql:"uri=studio_report_views_reader_view:sql/fields.sql"`
}

// ReportField is generated canonical view metadata for view.
type ReportField struct {
	MetadataJson json.RawMessage `sqlx:"metadata_json,enc=JSON"`
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	ViewId *string `sqlx:"view_id"`
	FieldName *string `sqlx:"field_name"`
	SourceColumn *string `sqlx:"source_column"`
	Expression *string `sqlx:"expression"`
	DatabaseType *string `sqlx:"database_type"`
	GoType *string `sqlx:"go_type"`
	Nullable *int `sqlx:"nullable"`
	Ordinal *int `sqlx:"ordinal"`
	Filterable *int `sqlx:"filterable"`
	Orderable *int `sqlx:"orderable"`
	Groupable *int `sqlx:"groupable"`
	Measurable *int `sqlx:"measurable"`
}
