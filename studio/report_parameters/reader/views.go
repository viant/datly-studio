package reader

import (
	json "encoding/json"
)

// ReportParameter is generated canonical view metadata for parameter.
type ReportParameter struct {
	QuerySelectorJson json.RawMessage `sqlx:"query_selector_json,enc=JSON"`
	CodecJson json.RawMessage `sqlx:"codec_json,enc=JSON"`
	ActivationJson json.RawMessage `sqlx:"activation_json,enc=JSON"`
	MetadataJson json.RawMessage `sqlx:"metadata_json,enc=JSON"`
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	ParameterId *string `sqlx:"parameter_id"`
	ParameterIdentity *string `sqlx:"parameter_identity"`
	Name *string `sqlx:"name"`
	SourceKind *string `sqlx:"source_kind"`
	SourceName *string `sqlx:"source_name"`
	TypeExpr *string `sqlx:"type_expr"`
	Required *int `sqlx:"required"`
	EmitOutput *int `sqlx:"emit_output"`
	Predicates []*ReportPredicate `view:"predicates,type=ReportPredicate,table=report_predicates" on:"ReportId:parameters.report_id=ReportId:predicates.report_id,VersionNo:parameters.version_no=VersionNo:predicates.version_no,ParameterId:parameters.parameter_id=ParameterId:predicates.parameter_id" sql:"uri=studio_report_parameters_reader_parameter:sql/predicates.sql"`
}

// ReportPredicate is generated canonical view metadata for parameter.
type ReportPredicate struct {
	ArgsJson json.RawMessage `sqlx:"args_json,enc=JSON"`
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	ParameterId *string `sqlx:"parameter_id"`
	PredicateIndex *int `sqlx:"predicate_index"`
	PredicateGroup *int `sqlx:"predicate_group"`
	Name *string `sqlx:"name"`
	ApplyWhenAbsent *int `sqlx:"apply_when_absent"`
}
