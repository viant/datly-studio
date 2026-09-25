package store_edit

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

type VersionHandlerCurrentVersionSlice []*CurrentVersionView

func VersionHandlerCurrentVersionIndexByReportIdKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ReportId, true
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
	return value.VersionNo, true
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
	return value.State, true
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
func VersionHandlerCurrentVersionIndexBySpecHashKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.SpecHash, true
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
func VersionHandlerCurrentVersionIndexByCompileStatusKey(value *CurrentVersionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.CompileStatus, true
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
func VersionHandlerCurrentVersionIndexBySourceRevisionKey(value *CurrentVersionView) (int64, bool) {
	var zero int64
	if value == nil {
		return zero, false
	}
	if value.SourceRevision == nil {
		return zero, false
	}
	return *value.SourceRevision, true
}

type VersionHandlerCurrentVersionIndexedBySourceRevision map[int64]*CurrentVersionView

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
func (index VersionHandlerCurrentVersionIndexedBySourceRevision) Has(key int64) bool {
	_, ok := index[key]
	return ok
}

type VersionHandlerCurrentVersionGroupedBySourceRevision map[int64][]*CurrentVersionView

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
func (index VersionHandlerCurrentVersionGroupedBySourceRevision) Has(key int64) bool {
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
	return VersionHandlerCurrentVersionKey{ReportId: value.ReportId, VersionNo: value.VersionNo}, true
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
	metadata, ok := xhandler.ReadMetadataFromContext(ctx)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ReportId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("VersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.VersionNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("State") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.State")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("AuthoredSql") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.AuthoredSql")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("AuthoredDql") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.AuthoredDql")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ComponentSpecJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.ComponentSpecJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SpecHash") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.SpecHash")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("GeneratedDql") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.GeneratedDql")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CompileStatus") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.CompileStatus")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CompileDiagnosticsJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.CompileDiagnosticsJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SourceRevision") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentVersion.SourceRevision")
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
