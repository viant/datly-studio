package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
)

type ParameterHandlerCurrentParameterSlice []*CurrentParameterView

func ParameterHandlerCurrentParameterIndexByReportIdKey(value *CurrentParameterView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	return *value.ReportId, true
}

type ParameterHandlerCurrentParameterIndexedByReportId map[string]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexByReportId() (ParameterHandlerCurrentParameterIndexedByReportId, error) {
	result := make(ParameterHandlerCurrentParameterIndexedByReportId)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedByReportId map[string][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupByReportId() ParameterHandlerCurrentParameterGroupedByReportId {
	result := make(ParameterHandlerCurrentParameterGroupedByReportId)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentParameterIndexByVersionNoKey(value *CurrentParameterView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	return *value.VersionNo, true
}

type ParameterHandlerCurrentParameterIndexedByVersionNo map[int]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexByVersionNo() (ParameterHandlerCurrentParameterIndexedByVersionNo, error) {
	result := make(ParameterHandlerCurrentParameterIndexedByVersionNo)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexByVersionNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedByVersionNo map[int][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupByVersionNo() ParameterHandlerCurrentParameterGroupedByVersionNo {
	result := make(ParameterHandlerCurrentParameterGroupedByVersionNo)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentParameterIndexByParameterIdKey(value *CurrentParameterView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ParameterId == nil {
		return zero, false
	}
	return *value.ParameterId, true
}

type ParameterHandlerCurrentParameterIndexedByParameterId map[string]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexByParameterId() (ParameterHandlerCurrentParameterIndexedByParameterId, error) {
	result := make(ParameterHandlerCurrentParameterIndexedByParameterId)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByParameterIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexByParameterId")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedByParameterId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedByParameterId map[string][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupByParameterId() ParameterHandlerCurrentParameterGroupedByParameterId {
	result := make(ParameterHandlerCurrentParameterGroupedByParameterId)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByParameterIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedByParameterId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentParameterIndexByParameterIdentityKey(value *CurrentParameterView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ParameterIdentity == nil {
		return zero, false
	}
	return *value.ParameterIdentity, true
}

type ParameterHandlerCurrentParameterIndexedByParameterIdentity map[string]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexByParameterIdentity() (ParameterHandlerCurrentParameterIndexedByParameterIdentity, error) {
	result := make(ParameterHandlerCurrentParameterIndexedByParameterIdentity)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByParameterIdentityKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexByParameterIdentity")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedByParameterIdentity) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedByParameterIdentity map[string][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupByParameterIdentity() ParameterHandlerCurrentParameterGroupedByParameterIdentity {
	result := make(ParameterHandlerCurrentParameterGroupedByParameterIdentity)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByParameterIdentityKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedByParameterIdentity) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentParameterIndexByNameKey(value *CurrentParameterView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Name == nil {
		return zero, false
	}
	return *value.Name, true
}

type ParameterHandlerCurrentParameterIndexedByName map[string]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexByName() (ParameterHandlerCurrentParameterIndexedByName, error) {
	result := make(ParameterHandlerCurrentParameterIndexedByName)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexByName")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedByName map[string][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupByName() ParameterHandlerCurrentParameterGroupedByName {
	result := make(ParameterHandlerCurrentParameterGroupedByName)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentParameterIndexBySourceKindKey(value *CurrentParameterView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SourceKind == nil {
		return zero, false
	}
	return *value.SourceKind, true
}

type ParameterHandlerCurrentParameterIndexedBySourceKind map[string]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexBySourceKind() (ParameterHandlerCurrentParameterIndexedBySourceKind, error) {
	result := make(ParameterHandlerCurrentParameterIndexedBySourceKind)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexBySourceKindKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexBySourceKind")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedBySourceKind) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedBySourceKind map[string][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupBySourceKind() ParameterHandlerCurrentParameterGroupedBySourceKind {
	result := make(ParameterHandlerCurrentParameterGroupedBySourceKind)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexBySourceKindKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedBySourceKind) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentParameterIndexBySourceNameKey(value *CurrentParameterView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SourceName == nil {
		return zero, false
	}
	return *value.SourceName, true
}

type ParameterHandlerCurrentParameterIndexedBySourceName map[string]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexBySourceName() (ParameterHandlerCurrentParameterIndexedBySourceName, error) {
	result := make(ParameterHandlerCurrentParameterIndexedBySourceName)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexBySourceNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexBySourceName")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedBySourceName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedBySourceName map[string][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupBySourceName() ParameterHandlerCurrentParameterGroupedBySourceName {
	result := make(ParameterHandlerCurrentParameterGroupedBySourceName)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexBySourceNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedBySourceName) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentParameterIndexByTypeExprKey(value *CurrentParameterView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.TypeExpr == nil {
		return zero, false
	}
	return *value.TypeExpr, true
}

type ParameterHandlerCurrentParameterIndexedByTypeExpr map[string]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexByTypeExpr() (ParameterHandlerCurrentParameterIndexedByTypeExpr, error) {
	result := make(ParameterHandlerCurrentParameterIndexedByTypeExpr)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByTypeExprKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexByTypeExpr")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedByTypeExpr) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedByTypeExpr map[string][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupByTypeExpr() ParameterHandlerCurrentParameterGroupedByTypeExpr {
	result := make(ParameterHandlerCurrentParameterGroupedByTypeExpr)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByTypeExprKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedByTypeExpr) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentParameterIndexByRequiredKey(value *CurrentParameterView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.Required == nil {
		return zero, false
	}
	return *value.Required, true
}

type ParameterHandlerCurrentParameterIndexedByRequired map[int]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexByRequired() (ParameterHandlerCurrentParameterIndexedByRequired, error) {
	result := make(ParameterHandlerCurrentParameterIndexedByRequired)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByRequiredKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexByRequired")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedByRequired) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedByRequired map[int][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupByRequired() ParameterHandlerCurrentParameterGroupedByRequired {
	result := make(ParameterHandlerCurrentParameterGroupedByRequired)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByRequiredKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedByRequired) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentParameterIndexByEmitOutputKey(value *CurrentParameterView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.EmitOutput == nil {
		return zero, false
	}
	return *value.EmitOutput, true
}

type ParameterHandlerCurrentParameterIndexedByEmitOutput map[int]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexByEmitOutput() (ParameterHandlerCurrentParameterIndexedByEmitOutput, error) {
	result := make(ParameterHandlerCurrentParameterIndexedByEmitOutput)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByEmitOutputKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexByEmitOutput")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedByEmitOutput) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedByEmitOutput map[int][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupByEmitOutput() ParameterHandlerCurrentParameterGroupedByEmitOutput {
	result := make(ParameterHandlerCurrentParameterGroupedByEmitOutput)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByEmitOutputKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedByEmitOutput) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentParameterIndexByOrdinalKey(value *CurrentParameterView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.Ordinal == nil {
		return zero, false
	}
	return *value.Ordinal, true
}

type ParameterHandlerCurrentParameterIndexedByOrdinal map[int]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexByOrdinal() (ParameterHandlerCurrentParameterIndexedByOrdinal, error) {
	result := make(ParameterHandlerCurrentParameterIndexedByOrdinal)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByOrdinalKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexByOrdinal")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedByOrdinal) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedByOrdinal map[int][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupByOrdinal() ParameterHandlerCurrentParameterGroupedByOrdinal {
	result := make(ParameterHandlerCurrentParameterGroupedByOrdinal)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByOrdinalKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedByOrdinal) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterKey struct {
	ReportId    string
	VersionNo   int
	ParameterId string
}

func ParameterHandlerCurrentParameterIndexByKeyKey(value *CurrentParameterView) (ParameterHandlerCurrentParameterKey, bool) {
	var zero ParameterHandlerCurrentParameterKey
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	if value.ParameterId == nil {
		return zero, false
	}
	return ParameterHandlerCurrentParameterKey{ReportId: *value.ReportId, VersionNo: *value.VersionNo, ParameterId: *value.ParameterId}, true
}

type ParameterHandlerCurrentParameterIndexedByKey map[ParameterHandlerCurrentParameterKey]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) IndexByKey() (ParameterHandlerCurrentParameterIndexedByKey, error) {
	result := make(ParameterHandlerCurrentParameterIndexedByKey)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentParameterSlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentParameterIndexedByKey) Has(key ParameterHandlerCurrentParameterKey) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentParameterGroupedByKey map[ParameterHandlerCurrentParameterKey][]*CurrentParameterView

func (rows ParameterHandlerCurrentParameterSlice) GroupByKey() ParameterHandlerCurrentParameterGroupedByKey {
	result := make(ParameterHandlerCurrentParameterGroupedByKey)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentParameterIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentParameterGroupedByKey) Has(key ParameterHandlerCurrentParameterKey) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateSlice []*CurrentPredicateView

func ParameterHandlerCurrentPredicateIndexByReportIdKey(value *CurrentPredicateView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	return *value.ReportId, true
}

type ParameterHandlerCurrentPredicateIndexedByReportId map[string]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) IndexByReportId() (ParameterHandlerCurrentPredicateIndexedByReportId, error) {
	result := make(ParameterHandlerCurrentPredicateIndexedByReportId)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentPredicateSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentPredicateIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateGroupedByReportId map[string][]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) GroupByReportId() ParameterHandlerCurrentPredicateGroupedByReportId {
	result := make(ParameterHandlerCurrentPredicateGroupedByReportId)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentPredicateGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentPredicateIndexByVersionNoKey(value *CurrentPredicateView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	return *value.VersionNo, true
}

type ParameterHandlerCurrentPredicateIndexedByVersionNo map[int]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) IndexByVersionNo() (ParameterHandlerCurrentPredicateIndexedByVersionNo, error) {
	result := make(ParameterHandlerCurrentPredicateIndexedByVersionNo)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentPredicateSlice.IndexByVersionNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentPredicateIndexedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateGroupedByVersionNo map[int][]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) GroupByVersionNo() ParameterHandlerCurrentPredicateGroupedByVersionNo {
	result := make(ParameterHandlerCurrentPredicateGroupedByVersionNo)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentPredicateGroupedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentPredicateIndexByParameterIdKey(value *CurrentPredicateView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ParameterId == nil {
		return zero, false
	}
	return *value.ParameterId, true
}

type ParameterHandlerCurrentPredicateIndexedByParameterId map[string]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) IndexByParameterId() (ParameterHandlerCurrentPredicateIndexedByParameterId, error) {
	result := make(ParameterHandlerCurrentPredicateIndexedByParameterId)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByParameterIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentPredicateSlice.IndexByParameterId")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentPredicateIndexedByParameterId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateGroupedByParameterId map[string][]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) GroupByParameterId() ParameterHandlerCurrentPredicateGroupedByParameterId {
	result := make(ParameterHandlerCurrentPredicateGroupedByParameterId)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByParameterIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentPredicateGroupedByParameterId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentPredicateIndexByPredicateIndexKey(value *CurrentPredicateView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.PredicateIndex == nil {
		return zero, false
	}
	return *value.PredicateIndex, true
}

type ParameterHandlerCurrentPredicateIndexedByPredicateIndex map[int]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) IndexByPredicateIndex() (ParameterHandlerCurrentPredicateIndexedByPredicateIndex, error) {
	result := make(ParameterHandlerCurrentPredicateIndexedByPredicateIndex)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByPredicateIndexKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentPredicateSlice.IndexByPredicateIndex")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentPredicateIndexedByPredicateIndex) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateGroupedByPredicateIndex map[int][]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) GroupByPredicateIndex() ParameterHandlerCurrentPredicateGroupedByPredicateIndex {
	result := make(ParameterHandlerCurrentPredicateGroupedByPredicateIndex)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByPredicateIndexKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentPredicateGroupedByPredicateIndex) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentPredicateIndexByNameKey(value *CurrentPredicateView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Name == nil {
		return zero, false
	}
	return *value.Name, true
}

type ParameterHandlerCurrentPredicateIndexedByName map[string]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) IndexByName() (ParameterHandlerCurrentPredicateIndexedByName, error) {
	result := make(ParameterHandlerCurrentPredicateIndexedByName)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentPredicateSlice.IndexByName")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentPredicateIndexedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateGroupedByName map[string][]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) GroupByName() ParameterHandlerCurrentPredicateGroupedByName {
	result := make(ParameterHandlerCurrentPredicateGroupedByName)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentPredicateGroupedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentPredicateIndexByPredicateGroupKey(value *CurrentPredicateView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.PredicateGroup == nil {
		return zero, false
	}
	return *value.PredicateGroup, true
}

type ParameterHandlerCurrentPredicateIndexedByPredicateGroup map[int]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) IndexByPredicateGroup() (ParameterHandlerCurrentPredicateIndexedByPredicateGroup, error) {
	result := make(ParameterHandlerCurrentPredicateIndexedByPredicateGroup)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByPredicateGroupKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentPredicateSlice.IndexByPredicateGroup")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentPredicateIndexedByPredicateGroup) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateGroupedByPredicateGroup map[int][]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) GroupByPredicateGroup() ParameterHandlerCurrentPredicateGroupedByPredicateGroup {
	result := make(ParameterHandlerCurrentPredicateGroupedByPredicateGroup)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByPredicateGroupKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentPredicateGroupedByPredicateGroup) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ParameterHandlerCurrentPredicateIndexByApplyWhenAbsentKey(value *CurrentPredicateView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.ApplyWhenAbsent == nil {
		return zero, false
	}
	return *value.ApplyWhenAbsent, true
}

type ParameterHandlerCurrentPredicateIndexedByApplyWhenAbsent map[int]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) IndexByApplyWhenAbsent() (ParameterHandlerCurrentPredicateIndexedByApplyWhenAbsent, error) {
	result := make(ParameterHandlerCurrentPredicateIndexedByApplyWhenAbsent)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByApplyWhenAbsentKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentPredicateSlice.IndexByApplyWhenAbsent")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentPredicateIndexedByApplyWhenAbsent) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateGroupedByApplyWhenAbsent map[int][]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) GroupByApplyWhenAbsent() ParameterHandlerCurrentPredicateGroupedByApplyWhenAbsent {
	result := make(ParameterHandlerCurrentPredicateGroupedByApplyWhenAbsent)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByApplyWhenAbsentKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentPredicateGroupedByApplyWhenAbsent) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateKey struct {
	ReportId       string
	VersionNo      int
	ParameterId    string
	PredicateIndex int
}

func ParameterHandlerCurrentPredicateIndexByKeyKey(value *CurrentPredicateView) (ParameterHandlerCurrentPredicateKey, bool) {
	var zero ParameterHandlerCurrentPredicateKey
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	if value.ParameterId == nil {
		return zero, false
	}
	if value.PredicateIndex == nil {
		return zero, false
	}
	return ParameterHandlerCurrentPredicateKey{ReportId: *value.ReportId, VersionNo: *value.VersionNo, ParameterId: *value.ParameterId, PredicateIndex: *value.PredicateIndex}, true
}

type ParameterHandlerCurrentPredicateIndexedByKey map[ParameterHandlerCurrentPredicateKey]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) IndexByKey() (ParameterHandlerCurrentPredicateIndexedByKey, error) {
	result := make(ParameterHandlerCurrentPredicateIndexedByKey)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentPredicateSlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentPredicateIndexedByKey) Has(key ParameterHandlerCurrentPredicateKey) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateGroupedByKey map[ParameterHandlerCurrentPredicateKey][]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) GroupByKey() ParameterHandlerCurrentPredicateGroupedByKey {
	result := make(ParameterHandlerCurrentPredicateGroupedByKey)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentPredicateGroupedByKey) Has(key ParameterHandlerCurrentPredicateKey) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateReportIdAndVersionNoAndParameterIdKey struct {
	ReportId    string
	VersionNo   int
	ParameterId string
}

func ParameterHandlerCurrentPredicateIndexByReportIdAndVersionNoAndParameterIdKey(value *CurrentPredicateView) (ParameterHandlerCurrentPredicateReportIdAndVersionNoAndParameterIdKey, bool) {
	var zero ParameterHandlerCurrentPredicateReportIdAndVersionNoAndParameterIdKey
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	if value.ParameterId == nil {
		return zero, false
	}
	return ParameterHandlerCurrentPredicateReportIdAndVersionNoAndParameterIdKey{ReportId: *value.ReportId, VersionNo: *value.VersionNo, ParameterId: *value.ParameterId}, true
}

type ParameterHandlerCurrentPredicateIndexedByReportIdAndVersionNoAndParameterId map[ParameterHandlerCurrentPredicateReportIdAndVersionNoAndParameterIdKey]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) IndexByReportIdAndVersionNoAndParameterId() (ParameterHandlerCurrentPredicateIndexedByReportIdAndVersionNoAndParameterId, error) {
	result := make(ParameterHandlerCurrentPredicateIndexedByReportIdAndVersionNoAndParameterId)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByReportIdAndVersionNoAndParameterIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ParameterHandlerCurrentPredicateSlice.IndexByReportIdAndVersionNoAndParameterId")
		}
		result[key] = row
	}
	return result, nil
}
func (index ParameterHandlerCurrentPredicateIndexedByReportIdAndVersionNoAndParameterId) Has(key ParameterHandlerCurrentPredicateReportIdAndVersionNoAndParameterIdKey) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerCurrentPredicateGroupedByReportIdAndVersionNoAndParameterId map[ParameterHandlerCurrentPredicateReportIdAndVersionNoAndParameterIdKey][]*CurrentPredicateView

func (rows ParameterHandlerCurrentPredicateSlice) GroupByReportIdAndVersionNoAndParameterId() ParameterHandlerCurrentPredicateGroupedByReportIdAndVersionNoAndParameterId {
	result := make(ParameterHandlerCurrentPredicateGroupedByReportIdAndVersionNoAndParameterId)
	for _, row := range rows {
		key, ok := ParameterHandlerCurrentPredicateIndexByReportIdAndVersionNoAndParameterIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ParameterHandlerCurrentPredicateGroupedByReportIdAndVersionNoAndParameterId) Has(key ParameterHandlerCurrentPredicateReportIdAndVersionNoAndParameterIdKey) bool {
	_, ok := index[key]
	return ok
}

type ParameterHandlerReadIndexes struct {
	CurrentParameter                                            ParameterHandlerCurrentParameterSlice
	CurrentParameterByKey                                       ParameterHandlerCurrentParameterIndexedByKey
	CurrentParameterGroupedByKey                                ParameterHandlerCurrentParameterGroupedByKey
	CurrentPredicate                                            ParameterHandlerCurrentPredicateSlice
	CurrentPredicateByKey                                       ParameterHandlerCurrentPredicateIndexedByKey
	CurrentPredicateGroupedByReportIdAndVersionNoAndParameterId ParameterHandlerCurrentPredicateGroupedByReportIdAndVersionNoAndParameterId
}

func BuildParameterHandlerReadIndexes(ctx context.Context, input *Input) (*ParameterHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &ParameterHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentParameter")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentParameter")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentParameter")
		if err != nil {
			return nil, err
		}
		carrier, err := inputAccess.Get(input)
		if err != nil {
			return nil, err
		}
		value := carrier.Interface()
		if projection.DirectOutput() && projection.RootHolder() != "" {
			return nil, fmt.Errorf("direct application read cannot declare a root holder")
		}
		if !projection.DirectOutput() {
			holder := projection.RootHolder()
			if holder == "" {
				return nil, fmt.Errorf("wrapped read projection requires a canonical root holder")
			}
			accessor, err := xshape.Linked(carrier.Type()).Accessor(holder)
			if err != nil {
				return nil, err
			}
			if _, err := (xshape.Collection[CurrentParameterView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentParameterView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentParameterView]{}).Pointers(value)
		if err != nil {
			return nil, err
		}
		for ordinal, row := range rows {
			if row == nil {
				continue
			}
			loaded, err := projection.Fields(ordinal)
			if err != nil {
				return nil, err
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("QuerySelectorJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.QuerySelectorJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CodecJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.CodecJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ActivationJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.ActivationJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("MetadataJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.MetadataJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ReportId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("VersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.VersionNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ParameterId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.ParameterId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ParameterIdentity") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.ParameterIdentity")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Name") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.Name")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SourceKind") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.SourceKind")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SourceName") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.SourceName")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("TypeExpr") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.TypeExpr")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Required") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.Required")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("EmitOutput") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.EmitOutput")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Ordinal") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentParameter.Ordinal")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentParameter = ParameterHandlerCurrentParameterSlice(cloned.([]*CurrentParameterView))
		result.CurrentParameterByKey, err = result.CurrentParameter.IndexByKey()
		if err != nil {
			return nil, err
		}
		result.CurrentParameterGroupedByKey = result.CurrentParameter.GroupByKey()
	}
	{
		projection, err := metadata.Projection("CurrentPredicate")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentPredicate")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentPredicate")
		if err != nil {
			return nil, err
		}
		carrier, err := inputAccess.Get(input)
		if err != nil {
			return nil, err
		}
		value := carrier.Interface()
		if projection.DirectOutput() && projection.RootHolder() != "" {
			return nil, fmt.Errorf("direct application read cannot declare a root holder")
		}
		if !projection.DirectOutput() {
			holder := projection.RootHolder()
			if holder == "" {
				return nil, fmt.Errorf("wrapped read projection requires a canonical root holder")
			}
			accessor, err := xshape.Linked(carrier.Type()).Accessor(holder)
			if err != nil {
				return nil, err
			}
			if _, err := (xshape.Collection[CurrentPredicateView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentPredicateView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentPredicateView]{}).Pointers(value)
		if err != nil {
			return nil, err
		}
		for ordinal, row := range rows {
			if row == nil {
				continue
			}
			loaded, err := projection.Fields(ordinal)
			if err != nil {
				return nil, err
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ArgsJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPredicate.ArgsJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ReportId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPredicate.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("VersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPredicate.VersionNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ParameterId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPredicate.ParameterId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("PredicateIndex") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPredicate.PredicateIndex")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Name") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPredicate.Name")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("PredicateGroup") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPredicate.PredicateGroup")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ApplyWhenAbsent") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPredicate.ApplyWhenAbsent")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentPredicate = ParameterHandlerCurrentPredicateSlice(cloned.([]*CurrentPredicateView))
		result.CurrentPredicateByKey, err = result.CurrentPredicate.IndexByKey()
		if err != nil {
			return nil, err
		}
		result.CurrentPredicateGroupedByReportIdAndVersionNoAndParameterId = result.CurrentPredicate.GroupByReportIdAndVersionNoAndParameterId()
	}
	return result, nil
}
func (input *Input) PrepareReadIndexes(ctx context.Context) error {
	if input == nil {
		return fmt.Errorf("read indexes require input")
	}
	input._parameterHandlerReadIndexes = nil
	indexes, err := BuildParameterHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._parameterHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*ParameterHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._parameterHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._parameterHandlerReadIndexes, nil
}
