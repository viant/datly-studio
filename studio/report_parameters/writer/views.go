package writer

import (
	json "encoding/json"
)

// ReportParameter is generated canonical view metadata for parameter.
type ReportParameter struct {
	QuerySelectorJson json.RawMessage `sqlx:"query_selector_json,enc=JSON"`
	CodecJson json.RawMessage `sqlx:"codec_json,enc=JSON"`
	ActivationJson json.RawMessage `sqlx:"activation_json,enc=JSON"`
	MetadataJson json.RawMessage `sqlx:"metadata_json,enc=JSON"`
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_versions,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_versions,refColumn=version_no,required=true"`
	ParameterId *string `sqlx:"parameter_id,primaryKey,required=true" validate:"required"`
	ParameterIdentity *string `validate:"required" sqlx:"parameter_identity,required=true"`
	Name *string `validate:"required" sqlx:"name,required=true"`
	SourceKind *string `validate:"required" sqlx:"source_kind,required=true"`
	ShouldDelete bool `sqlx:"-" writer:"delete"`
	SourceName *string `sqlx:"source_name"`
	TypeExpr *string `sqlx:"type_expr"`
	Required *int `sqlx:"required"`
	EmitOutput *int `sqlx:"emit_output,required=true"`
	Ordinal *int `sqlx:"ordinal,required=true"`
	Predicate []*ReportPredicate `view:"predicate,type=ReportPredicate,table=report_predicates" on:"ReportId:parameter.report_id=ReportId:predicate.report_id,VersionNo:parameter.version_no=VersionNo:predicate.version_no,ParameterId:parameter.parameter_id=ParameterId:predicate.parameter_id" sql:"uri=studio_report_parameters_writer_parameter:sql/predicate.sql"`
	Has *ReportParameterHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ReportParameterHas"`
}

type ReportParameterHas struct {
	QuerySelectorJson bool
	CodecJson bool
	ActivationJson bool
	MetadataJson bool
	ReportId bool
	VersionNo bool
	ParameterId bool
	ParameterIdentity bool
	Name bool
	SourceKind bool
	ShouldDelete bool
	SourceName bool
	TypeExpr bool
	Required bool
	EmitOutput bool
	Ordinal bool
	Predicate bool
}

// ReportPredicate is generated canonical view metadata for parameter.
type ReportPredicate struct {
	ArgsJson json.RawMessage `sqlx:"args_json,enc=JSON"`
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_parameters,refColumn=report_id,required=true"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_parameters,refColumn=version_no,required=true"`
	ParameterId *string `sqlx:"parameter_id,primaryKey,refTable=report_parameters,refColumn=parameter_id,required=true"`
	PredicateIndex *int `sqlx:"predicate_index,primaryKey,required=true"`
	Name *string `validate:"required" sqlx:"name,required=true"`
	ShouldDelete bool `sqlx:"-" writer:"delete"`
	PredicateGroup *int `sqlx:"predicate_group,required=true"`
	ApplyWhenAbsent *int `sqlx:"apply_when_absent,required=true"`
	Has *ReportPredicateHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ReportPredicateHas"`
}

type ReportPredicateHas struct {
	ArgsJson bool
	ReportId bool
	VersionNo bool
	ParameterId bool
	PredicateIndex bool
	Name bool
	ShouldDelete bool
	PredicateGroup bool
	ApplyWhenAbsent bool
}

// CurrentParameterView is generated canonical view metadata for parameter.
type CurrentParameterView struct {
	QuerySelectorJson json.RawMessage `sqlx:"query_selector_json,enc=JSON"`
	CodecJson json.RawMessage `sqlx:"codec_json,enc=JSON"`
	ActivationJson json.RawMessage `sqlx:"activation_json,enc=JSON"`
	MetadataJson json.RawMessage `sqlx:"metadata_json,enc=JSON"`
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_versions,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_versions,refColumn=version_no,required=true"`
	ParameterId *string `sqlx:"parameter_id,primaryKey,required=true" validate:"required"`
	ParameterIdentity *string `validate:"required" sqlx:"parameter_identity,required=true"`
	Name *string `validate:"required" sqlx:"name,required=true"`
	SourceKind *string `validate:"required" sqlx:"source_kind,required=true"`
	SourceName *string `sqlx:"source_name"`
	TypeExpr *string `sqlx:"type_expr"`
	Required *int `sqlx:"required"`
	EmitOutput *int `sqlx:"emit_output,required=true"`
	Ordinal *int `sqlx:"ordinal,required=true"`
}

// CurrentPredicateView is generated canonical view metadata for parameter.
type CurrentPredicateView struct {
	ArgsJson json.RawMessage `sqlx:"args_json,enc=JSON"`
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_parameters,refColumn=report_id,required=true"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_parameters,refColumn=version_no,required=true"`
	ParameterId *string `sqlx:"parameter_id,primaryKey,refTable=report_parameters,refColumn=parameter_id,required=true"`
	PredicateIndex *int `sqlx:"predicate_index,primaryKey,required=true"`
	Name *string `validate:"required" sqlx:"name,required=true"`
	PredicateGroup *int `sqlx:"predicate_group,required=true"`
	ApplyWhenAbsent *int `sqlx:"apply_when_absent,required=true"`
}

type ParameterKeysRow struct {
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	ParameterId *string `sqlx:"parameter_id"`
}
