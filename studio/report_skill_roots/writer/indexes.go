package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
)

type SkillHandlerCurrentSkillSlice []*CurrentSkillView

func SkillHandlerCurrentSkillIndexByReportIdKey(value *CurrentSkillView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	return *value.ReportId, true
}

type SkillHandlerCurrentSkillIndexedByReportId map[string]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) IndexByReportId() (SkillHandlerCurrentSkillIndexedByReportId, error) {
	result := make(SkillHandlerCurrentSkillIndexedByReportId)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index SkillHandlerCurrentSkillSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index SkillHandlerCurrentSkillIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type SkillHandlerCurrentSkillGroupedByReportId map[string][]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) GroupByReportId() SkillHandlerCurrentSkillGroupedByReportId {
	result := make(SkillHandlerCurrentSkillGroupedByReportId)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index SkillHandlerCurrentSkillGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func SkillHandlerCurrentSkillIndexByVersionNoKey(value *CurrentSkillView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	return *value.VersionNo, true
}

type SkillHandlerCurrentSkillIndexedByVersionNo map[int]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) IndexByVersionNo() (SkillHandlerCurrentSkillIndexedByVersionNo, error) {
	result := make(SkillHandlerCurrentSkillIndexedByVersionNo)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index SkillHandlerCurrentSkillSlice.IndexByVersionNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index SkillHandlerCurrentSkillIndexedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type SkillHandlerCurrentSkillGroupedByVersionNo map[int][]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) GroupByVersionNo() SkillHandlerCurrentSkillGroupedByVersionNo {
	result := make(SkillHandlerCurrentSkillGroupedByVersionNo)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index SkillHandlerCurrentSkillGroupedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func SkillHandlerCurrentSkillIndexBySkillIdKey(value *CurrentSkillView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SkillId == nil {
		return zero, false
	}
	return *value.SkillId, true
}

type SkillHandlerCurrentSkillIndexedBySkillId map[string]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) IndexBySkillId() (SkillHandlerCurrentSkillIndexedBySkillId, error) {
	result := make(SkillHandlerCurrentSkillIndexedBySkillId)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexBySkillIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index SkillHandlerCurrentSkillSlice.IndexBySkillId")
		}
		result[key] = row
	}
	return result, nil
}
func (index SkillHandlerCurrentSkillIndexedBySkillId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type SkillHandlerCurrentSkillGroupedBySkillId map[string][]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) GroupBySkillId() SkillHandlerCurrentSkillGroupedBySkillId {
	result := make(SkillHandlerCurrentSkillGroupedBySkillId)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexBySkillIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index SkillHandlerCurrentSkillGroupedBySkillId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func SkillHandlerCurrentSkillIndexByFolderIdKey(value *CurrentSkillView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.FolderId == nil {
		return zero, false
	}
	return *value.FolderId, true
}

type SkillHandlerCurrentSkillIndexedByFolderId map[string]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) IndexByFolderId() (SkillHandlerCurrentSkillIndexedByFolderId, error) {
	result := make(SkillHandlerCurrentSkillIndexedByFolderId)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexByFolderIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index SkillHandlerCurrentSkillSlice.IndexByFolderId")
		}
		result[key] = row
	}
	return result, nil
}
func (index SkillHandlerCurrentSkillIndexedByFolderId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type SkillHandlerCurrentSkillGroupedByFolderId map[string][]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) GroupByFolderId() SkillHandlerCurrentSkillGroupedByFolderId {
	result := make(SkillHandlerCurrentSkillGroupedByFolderId)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexByFolderIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index SkillHandlerCurrentSkillGroupedByFolderId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func SkillHandlerCurrentSkillIndexBySkillRootKey(value *CurrentSkillView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SkillRoot == nil {
		return zero, false
	}
	return *value.SkillRoot, true
}

type SkillHandlerCurrentSkillIndexedBySkillRoot map[string]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) IndexBySkillRoot() (SkillHandlerCurrentSkillIndexedBySkillRoot, error) {
	result := make(SkillHandlerCurrentSkillIndexedBySkillRoot)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexBySkillRootKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index SkillHandlerCurrentSkillSlice.IndexBySkillRoot")
		}
		result[key] = row
	}
	return result, nil
}
func (index SkillHandlerCurrentSkillIndexedBySkillRoot) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type SkillHandlerCurrentSkillGroupedBySkillRoot map[string][]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) GroupBySkillRoot() SkillHandlerCurrentSkillGroupedBySkillRoot {
	result := make(SkillHandlerCurrentSkillGroupedBySkillRoot)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexBySkillRootKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index SkillHandlerCurrentSkillGroupedBySkillRoot) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func SkillHandlerCurrentSkillIndexByOrdinalKey(value *CurrentSkillView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.Ordinal == nil {
		return zero, false
	}
	return *value.Ordinal, true
}

type SkillHandlerCurrentSkillIndexedByOrdinal map[int]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) IndexByOrdinal() (SkillHandlerCurrentSkillIndexedByOrdinal, error) {
	result := make(SkillHandlerCurrentSkillIndexedByOrdinal)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexByOrdinalKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index SkillHandlerCurrentSkillSlice.IndexByOrdinal")
		}
		result[key] = row
	}
	return result, nil
}
func (index SkillHandlerCurrentSkillIndexedByOrdinal) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type SkillHandlerCurrentSkillGroupedByOrdinal map[int][]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) GroupByOrdinal() SkillHandlerCurrentSkillGroupedByOrdinal {
	result := make(SkillHandlerCurrentSkillGroupedByOrdinal)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexByOrdinalKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index SkillHandlerCurrentSkillGroupedByOrdinal) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type SkillHandlerCurrentSkillKey struct {
	ReportId  string
	VersionNo int
	SkillId   string
}

func SkillHandlerCurrentSkillIndexByKeyKey(value *CurrentSkillView) (SkillHandlerCurrentSkillKey, bool) {
	var zero SkillHandlerCurrentSkillKey
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	if value.SkillId == nil {
		return zero, false
	}
	return SkillHandlerCurrentSkillKey{ReportId: *value.ReportId, VersionNo: *value.VersionNo, SkillId: *value.SkillId}, true
}

type SkillHandlerCurrentSkillIndexedByKey map[SkillHandlerCurrentSkillKey]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) IndexByKey() (SkillHandlerCurrentSkillIndexedByKey, error) {
	result := make(SkillHandlerCurrentSkillIndexedByKey)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index SkillHandlerCurrentSkillSlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index SkillHandlerCurrentSkillIndexedByKey) Has(key SkillHandlerCurrentSkillKey) bool {
	_, ok := index[key]
	return ok
}

type SkillHandlerCurrentSkillGroupedByKey map[SkillHandlerCurrentSkillKey][]*CurrentSkillView

func (rows SkillHandlerCurrentSkillSlice) GroupByKey() SkillHandlerCurrentSkillGroupedByKey {
	result := make(SkillHandlerCurrentSkillGroupedByKey)
	for _, row := range rows {
		key, ok := SkillHandlerCurrentSkillIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index SkillHandlerCurrentSkillGroupedByKey) Has(key SkillHandlerCurrentSkillKey) bool {
	_, ok := index[key]
	return ok
}

type SkillHandlerReadIndexes struct {
	CurrentSkill      SkillHandlerCurrentSkillSlice
	CurrentSkillByKey SkillHandlerCurrentSkillIndexedByKey
}

func BuildSkillHandlerReadIndexes(ctx context.Context, input *Input) (*SkillHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &SkillHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentSkill")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentSkill")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentSkill")
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
			if _, err := (xshape.Collection[CurrentSkillView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentSkillView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentSkillView]{}).Pointers(value)
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
				return nil, fmt.Errorf("application index field was not loaded: CurrentSkill.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("VersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentSkill.VersionNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SkillId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentSkill.SkillId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("FolderId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentSkill.FolderId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SkillRoot") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentSkill.SkillRoot")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Ordinal") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentSkill.Ordinal")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentSkill = SkillHandlerCurrentSkillSlice(cloned.([]*CurrentSkillView))
		result.CurrentSkillByKey, err = result.CurrentSkill.IndexByKey()
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
	input._skillHandlerReadIndexes = nil
	indexes, err := BuildSkillHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._skillHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*SkillHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._skillHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._skillHandlerReadIndexes, nil
}
