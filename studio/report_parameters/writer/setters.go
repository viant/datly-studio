package writer

import (
	json "encoding/json"
	fmt2 "fmt"
)

func (entity *ReportParameter) GetQuerySelectorJson() json.RawMessage {
	return entity.QuerySelectorJson
}
func (entity *ReportParameter) SetQuerySelectorJson(value json.RawMessage) {
	entity.QuerySelectorJson = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.QuerySelectorJson = true
}
func (entity *ReportParameter) GetCodecJson() json.RawMessage {
	return entity.CodecJson
}
func (entity *ReportParameter) SetCodecJson(value json.RawMessage) {
	entity.CodecJson = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.CodecJson = true
}
func (entity *ReportParameter) GetActivationJson() json.RawMessage {
	return entity.ActivationJson
}
func (entity *ReportParameter) SetActivationJson(value json.RawMessage) {
	entity.ActivationJson = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.ActivationJson = true
}
func (entity *ReportParameter) GetMetadataJson() json.RawMessage {
	return entity.MetadataJson
}
func (entity *ReportParameter) SetMetadataJson(value json.RawMessage) {
	entity.MetadataJson = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.MetadataJson = true
}
func (entity *ReportParameter) GetReportId() *string {
	return entity.ReportId
}
func (entity *ReportParameter) SetReportId(value *string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ReportParameter) GetVersionNo() *int {
	return entity.VersionNo
}
func (entity *ReportParameter) SetVersionNo(value *int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *ReportParameter) GetParameterId() *string {
	return entity.ParameterId
}
func (entity *ReportParameter) SetParameterId(value *string) {
	entity.ParameterId = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.ParameterId = true
}
func (entity *ReportParameter) GetParameterIdentity() *string {
	return entity.ParameterIdentity
}
func (entity *ReportParameter) SetParameterIdentity(value *string) {
	entity.ParameterIdentity = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.ParameterIdentity = true
}
func (entity *ReportParameter) GetName() *string {
	return entity.Name
}
func (entity *ReportParameter) SetName(value *string) {
	entity.Name = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.Name = true
}
func (entity *ReportParameter) GetSourceKind() *string {
	return entity.SourceKind
}
func (entity *ReportParameter) SetSourceKind(value *string) {
	entity.SourceKind = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.SourceKind = true
}
func (entity *ReportParameter) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *ReportParameter) GetSourceName() *string {
	return entity.SourceName
}
func (entity *ReportParameter) SetSourceName(value *string) {
	entity.SourceName = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.SourceName = true
}
func (entity *ReportParameter) GetTypeExpr() *string {
	return entity.TypeExpr
}
func (entity *ReportParameter) SetTypeExpr(value *string) {
	entity.TypeExpr = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.TypeExpr = true
}
func (entity *ReportParameter) GetRequired() *int {
	return entity.Required
}
func (entity *ReportParameter) SetRequired(value *int) {
	entity.Required = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.Required = true
}
func (entity *ReportParameter) GetEmitOutput() *int {
	return entity.EmitOutput
}
func (entity *ReportParameter) SetEmitOutput(value *int) {
	entity.EmitOutput = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.EmitOutput = true
}
func (entity *ReportParameter) GetOrdinal() *int {
	return entity.Ordinal
}
func (entity *ReportParameter) SetOrdinal(value *int) {
	entity.Ordinal = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.Ordinal = true
}
func (entity *ReportParameter) GetPredicate() []*ReportPredicate {
	return entity.Predicate
}
func (entity *ReportParameter) SetPredicate(value []*ReportPredicate) {
	entity.Predicate = value
	if entity.Has == nil {
		entity.Has = &ReportParameterHas{}
	}
	entity.Has.Predicate = true
}
func (entity *ReportPredicate) GetArgsJson() json.RawMessage {
	return entity.ArgsJson
}
func (entity *ReportPredicate) SetArgsJson(value json.RawMessage) {
	entity.ArgsJson = value
	if entity.Has == nil {
		entity.Has = &ReportPredicateHas{}
	}
	entity.Has.ArgsJson = true
}
func (entity *ReportPredicate) GetReportId() *string {
	return entity.ReportId
}
func (entity *ReportPredicate) SetReportId(value *string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ReportPredicateHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ReportPredicate) GetVersionNo() *int {
	return entity.VersionNo
}
func (entity *ReportPredicate) SetVersionNo(value *int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &ReportPredicateHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *ReportPredicate) GetParameterId() *string {
	return entity.ParameterId
}
func (entity *ReportPredicate) SetParameterId(value *string) {
	entity.ParameterId = value
	if entity.Has == nil {
		entity.Has = &ReportPredicateHas{}
	}
	entity.Has.ParameterId = true
}
func (entity *ReportPredicate) GetPredicateIndex() *int {
	return entity.PredicateIndex
}
func (entity *ReportPredicate) SetPredicateIndex(value *int) {
	entity.PredicateIndex = value
	if entity.Has == nil {
		entity.Has = &ReportPredicateHas{}
	}
	entity.Has.PredicateIndex = true
}
func (entity *ReportPredicate) GetName() *string {
	return entity.Name
}
func (entity *ReportPredicate) SetName(value *string) {
	entity.Name = value
	if entity.Has == nil {
		entity.Has = &ReportPredicateHas{}
	}
	entity.Has.Name = true
}
func (entity *ReportPredicate) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *ReportPredicate) GetPredicateGroup() *int {
	return entity.PredicateGroup
}
func (entity *ReportPredicate) SetPredicateGroup(value *int) {
	entity.PredicateGroup = value
	if entity.Has == nil {
		entity.Has = &ReportPredicateHas{}
	}
	entity.Has.PredicateGroup = true
}
func (entity *ReportPredicate) GetApplyWhenAbsent() *int {
	return entity.ApplyWhenAbsent
}
func (entity *ReportPredicate) SetApplyWhenAbsent(value *int) {
	entity.ApplyWhenAbsent = value
	if entity.Has == nil {
		entity.Has = &ReportPredicateHas{}
	}
	entity.Has.ApplyWhenAbsent = true
}
func (input *Input) ProjectCurrentPredicateParentKeys(previous []*CurrentParameterView) ([]struct {
	ReportId    *string "sqlx:\"report_id\""
	VersionNo   *int    "sqlx:\"version_no\""
	ParameterId *string "sqlx:\"parameter_id\""
}, error) {
	result := []struct {
		ReportId    *string "sqlx:\"report_id\""
		VersionNo   *int    "sqlx:\"version_no\""
		ParameterId *string "sqlx:\"parameter_id\""
	}{}
	for _, row := range previous {
		if row == nil {
			continue
		}
		if row.ReportId == nil || row.VersionNo == nil || row.ParameterId == nil {
			return nil, fmt2.Errorf("scoped parent identity is null")
		}
		result = append(result, struct {
			ReportId    *string "sqlx:\"report_id\""
			VersionNo   *int    "sqlx:\"version_no\""
			ParameterId *string "sqlx:\"parameter_id\""
		}{ReportId: row.ReportId, VersionNo: row.VersionNo, ParameterId: row.ParameterId})
	}
	return result, nil
}
