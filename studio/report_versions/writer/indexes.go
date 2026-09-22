package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
)

type VersionHandlerCurrentVersionSlice []*CurrentVersionView

func VersionHandlerCurrentVersionIndexByReportIdKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	return *value.ReportId, true
}

type VersionHandlerCurrentVersionIndexedByReportId map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByReportId() (VersionHandlerCurrentVersionIndexedByReportId, error) {
	result := make(VersionHandlerCurrentVersionIndexedByReportId)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByReportId map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByReportId() VersionHandlerCurrentVersionGroupedByReportId {
	result := make(VersionHandlerCurrentVersionGroupedByReportId)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByVersionNoKey(value *CurrentVersionView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	return *value.VersionNo, true
}

type VersionHandlerCurrentVersionIndexedByVersionNo map[int]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByVersionNo() (VersionHandlerCurrentVersionIndexedByVersionNo, error) {
	result := make(VersionHandlerCurrentVersionIndexedByVersionNo)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByVersionNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByVersionNo map[int][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByVersionNo() VersionHandlerCurrentVersionGroupedByVersionNo {
	result := make(VersionHandlerCurrentVersionGroupedByVersionNo)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByStateKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.State == nil {
		return zero, false
	}
	return *value.State, true
}

type VersionHandlerCurrentVersionIndexedByState map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByState() (VersionHandlerCurrentVersionIndexedByState, error) {
	result := make(VersionHandlerCurrentVersionIndexedByState)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByStateKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByState")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByState) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByState map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByState() VersionHandlerCurrentVersionGroupedByState {
	result := make(VersionHandlerCurrentVersionGroupedByState)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByStateKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByState) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByAuthoringModeKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.AuthoringMode == nil {
		return zero, false
	}
	return *value.AuthoringMode, true
}

type VersionHandlerCurrentVersionIndexedByAuthoringMode map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByAuthoringMode() (VersionHandlerCurrentVersionIndexedByAuthoringMode, error) {
	result := make(VersionHandlerCurrentVersionIndexedByAuthoringMode)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByAuthoringModeKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByAuthoringMode")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByAuthoringMode) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByAuthoringMode map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByAuthoringMode() VersionHandlerCurrentVersionGroupedByAuthoringMode {
	result := make(VersionHandlerCurrentVersionGroupedByAuthoringMode)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByAuthoringModeKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByAuthoringMode) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexBySpecFormatVersionKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SpecFormatVersion == nil {
		return zero, false
	}
	return *value.SpecFormatVersion, true
}

type VersionHandlerCurrentVersionIndexedBySpecFormatVersion map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexBySpecFormatVersion() (VersionHandlerCurrentVersionIndexedBySpecFormatVersion, error) {
	result := make(VersionHandlerCurrentVersionIndexedBySpecFormatVersion)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexBySpecFormatVersionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexBySpecFormatVersion")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedBySpecFormatVersion) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedBySpecFormatVersion map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupBySpecFormatVersion() VersionHandlerCurrentVersionGroupedBySpecFormatVersion {
	result := make(VersionHandlerCurrentVersionGroupedBySpecFormatVersion)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexBySpecFormatVersionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedBySpecFormatVersion) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexBySpecHashKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SpecHash == nil {
		return zero, false
	}
	return *value.SpecHash, true
}

type VersionHandlerCurrentVersionIndexedBySpecHash map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexBySpecHash() (VersionHandlerCurrentVersionIndexedBySpecHash, error) {
	result := make(VersionHandlerCurrentVersionIndexedBySpecHash)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexBySpecHashKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexBySpecHash")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedBySpecHash) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedBySpecHash map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupBySpecHash() VersionHandlerCurrentVersionGroupedBySpecHash {
	result := make(VersionHandlerCurrentVersionGroupedBySpecHash)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexBySpecHashKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedBySpecHash) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByCompileStatusKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.CompileStatus == nil {
		return zero, false
	}
	return *value.CompileStatus, true
}

type VersionHandlerCurrentVersionIndexedByCompileStatus map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByCompileStatus() (VersionHandlerCurrentVersionIndexedByCompileStatus, error) {
	result := make(VersionHandlerCurrentVersionIndexedByCompileStatus)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByCompileStatusKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByCompileStatus")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByCompileStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByCompileStatus map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByCompileStatus() VersionHandlerCurrentVersionGroupedByCompileStatus {
	result := make(VersionHandlerCurrentVersionGroupedByCompileStatus)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByCompileStatusKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByCompileStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByDatlyVersionKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.DatlyVersion == nil {
		return zero, false
	}
	return *value.DatlyVersion, true
}

type VersionHandlerCurrentVersionIndexedByDatlyVersion map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByDatlyVersion() (VersionHandlerCurrentVersionIndexedByDatlyVersion, error) {
	result := make(VersionHandlerCurrentVersionIndexedByDatlyVersion)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByDatlyVersionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByDatlyVersion")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByDatlyVersion) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByDatlyVersion map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByDatlyVersion() VersionHandlerCurrentVersionGroupedByDatlyVersion {
	result := make(VersionHandlerCurrentVersionGroupedByDatlyVersion)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByDatlyVersionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByDatlyVersion) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByCompilerVersionKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.CompilerVersion == nil {
		return zero, false
	}
	return *value.CompilerVersion, true
}

type VersionHandlerCurrentVersionIndexedByCompilerVersion map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByCompilerVersion() (VersionHandlerCurrentVersionIndexedByCompilerVersion, error) {
	result := make(VersionHandlerCurrentVersionIndexedByCompilerVersion)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByCompilerVersionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByCompilerVersion")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByCompilerVersion) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByCompilerVersion map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByCompilerVersion() VersionHandlerCurrentVersionGroupedByCompilerVersion {
	result := make(VersionHandlerCurrentVersionGroupedByCompilerVersion)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByCompilerVersionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByCompilerVersion) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByCreatedByKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.CreatedBy == nil {
		return zero, false
	}
	return *value.CreatedBy, true
}

type VersionHandlerCurrentVersionIndexedByCreatedBy map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByCreatedBy() (VersionHandlerCurrentVersionIndexedByCreatedBy, error) {
	result := make(VersionHandlerCurrentVersionIndexedByCreatedBy)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByCreatedByKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByCreatedBy")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByCreatedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByCreatedBy map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByCreatedBy() VersionHandlerCurrentVersionGroupedByCreatedBy {
	result := make(VersionHandlerCurrentVersionGroupedByCreatedBy)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByCreatedByKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByCreatedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexBySourceRevisionKey(value *CurrentVersionView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.SourceRevision == nil {
		return zero, false
	}
	return *value.SourceRevision, true
}

type VersionHandlerCurrentVersionIndexedBySourceRevision map[int]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexBySourceRevision() (VersionHandlerCurrentVersionIndexedBySourceRevision, error) {
	result := make(VersionHandlerCurrentVersionIndexedBySourceRevision)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexBySourceRevisionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexBySourceRevision")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedBySourceRevision) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedBySourceRevision map[int][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupBySourceRevision() VersionHandlerCurrentVersionGroupedBySourceRevision {
	result := make(VersionHandlerCurrentVersionGroupedBySourceRevision)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexBySourceRevisionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedBySourceRevision) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByAuthoredSqlKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.AuthoredSql == nil {
		return zero, false
	}
	return *value.AuthoredSql, true
}

type VersionHandlerCurrentVersionIndexedByAuthoredSql map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByAuthoredSql() (VersionHandlerCurrentVersionIndexedByAuthoredSql, error) {
	result := make(VersionHandlerCurrentVersionIndexedByAuthoredSql)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByAuthoredSqlKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByAuthoredSql")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByAuthoredSql) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByAuthoredSql map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByAuthoredSql() VersionHandlerCurrentVersionGroupedByAuthoredSql {
	result := make(VersionHandlerCurrentVersionGroupedByAuthoredSql)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByAuthoredSqlKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByAuthoredSql) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByAuthoredDqlKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.AuthoredDql == nil {
		return zero, false
	}
	return *value.AuthoredDql, true
}

type VersionHandlerCurrentVersionIndexedByAuthoredDql map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByAuthoredDql() (VersionHandlerCurrentVersionIndexedByAuthoredDql, error) {
	result := make(VersionHandlerCurrentVersionIndexedByAuthoredDql)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByAuthoredDqlKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByAuthoredDql")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByAuthoredDql) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByAuthoredDql map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByAuthoredDql() VersionHandlerCurrentVersionGroupedByAuthoredDql {
	result := make(VersionHandlerCurrentVersionGroupedByAuthoredDql)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByAuthoredDqlKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByAuthoredDql) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByGeneratedDqlKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.GeneratedDql == nil {
		return zero, false
	}
	return *value.GeneratedDql, true
}

type VersionHandlerCurrentVersionIndexedByGeneratedDql map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByGeneratedDql() (VersionHandlerCurrentVersionIndexedByGeneratedDql, error) {
	result := make(VersionHandlerCurrentVersionIndexedByGeneratedDql)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByGeneratedDqlKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByGeneratedDql")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByGeneratedDql) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByGeneratedDql map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByGeneratedDql() VersionHandlerCurrentVersionGroupedByGeneratedDql {
	result := make(VersionHandlerCurrentVersionGroupedByGeneratedDql)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByGeneratedDqlKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByGeneratedDql) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByNotesKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Notes == nil {
		return zero, false
	}
	return *value.Notes, true
}

type VersionHandlerCurrentVersionIndexedByNotes map[string]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByNotes() (VersionHandlerCurrentVersionIndexedByNotes, error) {
	result := make(VersionHandlerCurrentVersionIndexedByNotes)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByNotesKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByNotes")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByNotes) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByNotes map[string][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByNotes() VersionHandlerCurrentVersionGroupedByNotes {
	result := make(VersionHandlerCurrentVersionGroupedByNotes)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByNotesKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByNotes) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByCreatedAtKey(value *CurrentVersionView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.CreatedAt == nil {
		return zero, false
	}
	return *value.CreatedAt, true
}

type VersionHandlerCurrentVersionIndexedByCreatedAt map[time.Time]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByCreatedAt() (VersionHandlerCurrentVersionIndexedByCreatedAt, error) {
	result := make(VersionHandlerCurrentVersionIndexedByCreatedAt)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByCreatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByCreatedAt map[time.Time][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByCreatedAt() VersionHandlerCurrentVersionGroupedByCreatedAt {
	result := make(VersionHandlerCurrentVersionGroupedByCreatedAt)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByValidatedAtKey(value *CurrentVersionView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.ValidatedAt == nil {
		return zero, false
	}
	return *value.ValidatedAt, true
}

type VersionHandlerCurrentVersionIndexedByValidatedAt map[time.Time]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByValidatedAt() (VersionHandlerCurrentVersionIndexedByValidatedAt, error) {
	result := make(VersionHandlerCurrentVersionIndexedByValidatedAt)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByValidatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByValidatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByValidatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByValidatedAt map[time.Time][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByValidatedAt() VersionHandlerCurrentVersionGroupedByValidatedAt {
	result := make(VersionHandlerCurrentVersionGroupedByValidatedAt)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByValidatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByValidatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func VersionHandlerCurrentVersionIndexByPublishedAtKey(value *CurrentVersionView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.PublishedAt == nil {
		return zero, false
	}
	return *value.PublishedAt, true
}

type VersionHandlerCurrentVersionIndexedByPublishedAt map[time.Time]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByPublishedAt() (VersionHandlerCurrentVersionIndexedByPublishedAt, error) {
	result := make(VersionHandlerCurrentVersionIndexedByPublishedAt)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByPublishedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByPublishedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByPublishedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByPublishedAt map[time.Time][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByPublishedAt() VersionHandlerCurrentVersionGroupedByPublishedAt {
	result := make(VersionHandlerCurrentVersionGroupedByPublishedAt)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByPublishedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByPublishedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionKey struct {
	ReportId  string
	VersionNo int
}

func VersionHandlerCurrentVersionIndexByKeyKey(value *CurrentVersionView) (VersionHandlerCurrentVersionKey, bool) {
	var zero VersionHandlerCurrentVersionKey
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	return VersionHandlerCurrentVersionKey{ReportId: *value.ReportId, VersionNo: *value.VersionNo}, true
}

type VersionHandlerCurrentVersionIndexedByKey map[VersionHandlerCurrentVersionKey]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) IndexByKey() (VersionHandlerCurrentVersionIndexedByKey, error) {
	result := make(VersionHandlerCurrentVersionIndexedByKey)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index VersionHandlerCurrentVersionSlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index VersionHandlerCurrentVersionIndexedByKey) Has(key VersionHandlerCurrentVersionKey) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedByKey map[VersionHandlerCurrentVersionKey][]*CurrentVersionView

func (rows VersionHandlerCurrentVersionSlice) GroupByKey() VersionHandlerCurrentVersionGroupedByKey {
	result := make(VersionHandlerCurrentVersionGroupedByKey)
	for _, row := range rows {
		key, ok := VersionHandlerCurrentVersionIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index VersionHandlerCurrentVersionGroupedByKey) Has(key VersionHandlerCurrentVersionKey) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerReadIndexes struct {
	CurrentVersion      VersionHandlerCurrentVersionSlice
	CurrentVersionByKey VersionHandlerCurrentVersionIndexedByKey
}

func BuildVersionHandlerReadIndexes(ctx context.Context, input *Input) (*VersionHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &VersionHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentVersion")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentVersion")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentVersion")
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
			if _, err := (xshape.Collection[CurrentVersionView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentVersionView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentVersionView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ComponentSpecJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.ComponentSpecJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DqlExportLimitsJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.DqlExportLimitsJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("TypeManifestJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.TypeManifestJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ResourceManifestJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.ResourceManifestJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ComponentDescriptorJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.ComponentDescriptorJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CompileDiagnosticsJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.CompileDiagnosticsJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ReportId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("VersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.VersionNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("State") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.State")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("AuthoringMode") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.AuthoringMode")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SpecFormatVersion") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.SpecFormatVersion")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SpecHash") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.SpecHash")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CompileStatus") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.CompileStatus")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DatlyVersion") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.DatlyVersion")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CompilerVersion") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.CompilerVersion")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CreatedBy") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.CreatedBy")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SourceRevision") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.SourceRevision")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("AuthoredSql") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.AuthoredSql")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("AuthoredDql") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.AuthoredDql")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("GeneratedDql") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.GeneratedDql")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Notes") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.Notes")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CreatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.CreatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ValidatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.ValidatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("PublishedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.PublishedAt")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentVersion = VersionHandlerCurrentVersionSlice(cloned.([]*CurrentVersionView))
		result.CurrentVersionByKey, err = result.CurrentVersion.IndexByKey()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
func (input *Input) PrepareReadIndexes(ctx context.Context) error {
	if input == nil {
		return fmt.Errorf("read indexes require input")
	}
	input._versionHandlerReadIndexes = nil
	indexes, err := BuildVersionHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._versionHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*VersionHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._versionHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._versionHandlerReadIndexes, nil
}
