package store_draft_pointer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
)

type ReportHandlerCurrentReportSlice []*CurrentReportView

func ReportHandlerCurrentReportIndexByIdKey(value *CurrentReportView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Id, true
}

type ReportHandlerCurrentReportIndexedById map[string]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexById() (ReportHandlerCurrentReportIndexedById, error) {
	result := make(ReportHandlerCurrentReportIndexedById)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexById")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedById) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedById map[string][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupById() ReportHandlerCurrentReportGroupedById {
	result := make(ReportHandlerCurrentReportGroupedById)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedById) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ReportHandlerCurrentReportIndexByCurrentDraftVersionKey(value *CurrentReportView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.CurrentDraftVersion == nil {
		return zero, false
	}
	return *value.CurrentDraftVersion, true
}

type ReportHandlerCurrentReportIndexedByCurrentDraftVersion map[int]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByCurrentDraftVersion() (ReportHandlerCurrentReportIndexedByCurrentDraftVersion, error) {
	result := make(ReportHandlerCurrentReportIndexedByCurrentDraftVersion)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByCurrentDraftVersionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByCurrentDraftVersion")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByCurrentDraftVersion) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByCurrentDraftVersion map[int][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByCurrentDraftVersion() ReportHandlerCurrentReportGroupedByCurrentDraftVersion {
	result := make(ReportHandlerCurrentReportGroupedByCurrentDraftVersion)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByCurrentDraftVersionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByCurrentDraftVersion) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ReportHandlerCurrentReportIndexByEtagKey(value *CurrentReportView) (int64, bool) {
	var zero int64
	if value == nil {
		return zero, false
	}
	if value.Etag == nil {
		return zero, false
	}
	return *value.Etag, true
}

type ReportHandlerCurrentReportIndexedByEtag map[int64]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByEtag() (ReportHandlerCurrentReportIndexedByEtag, error) {
	result := make(ReportHandlerCurrentReportIndexedByEtag)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByEtagKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByEtag")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByEtag) Has(key int64) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByEtag map[int64][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByEtag() ReportHandlerCurrentReportGroupedByEtag {
	result := make(ReportHandlerCurrentReportGroupedByEtag)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByEtagKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByEtag) Has(key int64) bool {
	_, ok := index[key]
	return ok
}
func ReportHandlerCurrentReportIndexByUpdatedAtKey(value *CurrentReportView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.UpdatedAt == nil {
		return zero, false
	}
	return *value.UpdatedAt, true
}

type ReportHandlerCurrentReportIndexedByUpdatedAt map[time.Time]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByUpdatedAt() (ReportHandlerCurrentReportIndexedByUpdatedAt, error) {
	result := make(ReportHandlerCurrentReportIndexedByUpdatedAt)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByUpdatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByUpdatedAt map[time.Time][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByUpdatedAt() ReportHandlerCurrentReportGroupedByUpdatedAt {
	result := make(ReportHandlerCurrentReportGroupedByUpdatedAt)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerReadIndexes struct {
	CurrentReport     ReportHandlerCurrentReportSlice
	CurrentReportById ReportHandlerCurrentReportIndexedById
}

func BuildReportHandlerReadIndexes(ctx context.Context, input *Input) (*ReportHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &ReportHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentReport")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentReport")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentReport")
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
			if _, err := (xshape.Collection[CurrentReportView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentReportView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentReportView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Id") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.Id")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CurrentDraftVersion") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.CurrentDraftVersion")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Etag") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.Etag")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("UpdatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.UpdatedAt")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentReport = ReportHandlerCurrentReportSlice(cloned.([]*CurrentReportView))
		result.CurrentReportById, err = result.CurrentReport.IndexById()
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
	input._reportHandlerReadIndexes = nil
	indexes, err := BuildReportHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._reportHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*ReportHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._reportHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._reportHandlerReadIndexes, nil
}
