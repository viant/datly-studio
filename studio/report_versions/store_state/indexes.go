package store_state

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
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
