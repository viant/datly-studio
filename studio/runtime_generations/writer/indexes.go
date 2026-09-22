package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
)

type GenerationHandlerCurrentGenerationSlice []*CurrentGenerationView

func GenerationHandlerCurrentGenerationIndexByGenerationNoKey(value *CurrentGenerationView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.GenerationNo == nil {
		return zero, false
	}
	return *value.GenerationNo, true
}

type GenerationHandlerCurrentGenerationIndexedByGenerationNo map[int]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) IndexByGenerationNo() (GenerationHandlerCurrentGenerationIndexedByGenerationNo, error) {
	result := make(GenerationHandlerCurrentGenerationIndexedByGenerationNo)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByGenerationNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index GenerationHandlerCurrentGenerationSlice.IndexByGenerationNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index GenerationHandlerCurrentGenerationIndexedByGenerationNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type GenerationHandlerCurrentGenerationGroupedByGenerationNo map[int][]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) GroupByGenerationNo() GenerationHandlerCurrentGenerationGroupedByGenerationNo {
	result := make(GenerationHandlerCurrentGenerationGroupedByGenerationNo)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByGenerationNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index GenerationHandlerCurrentGenerationGroupedByGenerationNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func GenerationHandlerCurrentGenerationIndexBySourceRevisionKey(value *CurrentGenerationView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SourceRevision == nil {
		return zero, false
	}
	return *value.SourceRevision, true
}

type GenerationHandlerCurrentGenerationIndexedBySourceRevision map[string]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) IndexBySourceRevision() (GenerationHandlerCurrentGenerationIndexedBySourceRevision, error) {
	result := make(GenerationHandlerCurrentGenerationIndexedBySourceRevision)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexBySourceRevisionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index GenerationHandlerCurrentGenerationSlice.IndexBySourceRevision")
		}
		result[key] = row
	}
	return result, nil
}
func (index GenerationHandlerCurrentGenerationIndexedBySourceRevision) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type GenerationHandlerCurrentGenerationGroupedBySourceRevision map[string][]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) GroupBySourceRevision() GenerationHandlerCurrentGenerationGroupedBySourceRevision {
	result := make(GenerationHandlerCurrentGenerationGroupedBySourceRevision)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexBySourceRevisionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index GenerationHandlerCurrentGenerationGroupedBySourceRevision) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func GenerationHandlerCurrentGenerationIndexByStatusKey(value *CurrentGenerationView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Status == nil {
		return zero, false
	}
	return *value.Status, true
}

type GenerationHandlerCurrentGenerationIndexedByStatus map[string]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) IndexByStatus() (GenerationHandlerCurrentGenerationIndexedByStatus, error) {
	result := make(GenerationHandlerCurrentGenerationIndexedByStatus)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByStatusKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index GenerationHandlerCurrentGenerationSlice.IndexByStatus")
		}
		result[key] = row
	}
	return result, nil
}
func (index GenerationHandlerCurrentGenerationIndexedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type GenerationHandlerCurrentGenerationGroupedByStatus map[string][]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) GroupByStatus() GenerationHandlerCurrentGenerationGroupedByStatus {
	result := make(GenerationHandlerCurrentGenerationGroupedByStatus)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByStatusKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index GenerationHandlerCurrentGenerationGroupedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func GenerationHandlerCurrentGenerationIndexByRequestedByKey(value *CurrentGenerationView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.RequestedBy == nil {
		return zero, false
	}
	return *value.RequestedBy, true
}

type GenerationHandlerCurrentGenerationIndexedByRequestedBy map[string]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) IndexByRequestedBy() (GenerationHandlerCurrentGenerationIndexedByRequestedBy, error) {
	result := make(GenerationHandlerCurrentGenerationIndexedByRequestedBy)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByRequestedByKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index GenerationHandlerCurrentGenerationSlice.IndexByRequestedBy")
		}
		result[key] = row
	}
	return result, nil
}
func (index GenerationHandlerCurrentGenerationIndexedByRequestedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type GenerationHandlerCurrentGenerationGroupedByRequestedBy map[string][]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) GroupByRequestedBy() GenerationHandlerCurrentGenerationGroupedByRequestedBy {
	result := make(GenerationHandlerCurrentGenerationGroupedByRequestedBy)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByRequestedByKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index GenerationHandlerCurrentGenerationGroupedByRequestedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func GenerationHandlerCurrentGenerationIndexByRequestedAtKey(value *CurrentGenerationView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.RequestedAt == nil {
		return zero, false
	}
	return *value.RequestedAt, true
}

type GenerationHandlerCurrentGenerationIndexedByRequestedAt map[time.Time]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) IndexByRequestedAt() (GenerationHandlerCurrentGenerationIndexedByRequestedAt, error) {
	result := make(GenerationHandlerCurrentGenerationIndexedByRequestedAt)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByRequestedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index GenerationHandlerCurrentGenerationSlice.IndexByRequestedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index GenerationHandlerCurrentGenerationIndexedByRequestedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type GenerationHandlerCurrentGenerationGroupedByRequestedAt map[time.Time][]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) GroupByRequestedAt() GenerationHandlerCurrentGenerationGroupedByRequestedAt {
	result := make(GenerationHandlerCurrentGenerationGroupedByRequestedAt)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByRequestedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index GenerationHandlerCurrentGenerationGroupedByRequestedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func GenerationHandlerCurrentGenerationIndexByReportCountKey(value *CurrentGenerationView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.ReportCount == nil {
		return zero, false
	}
	return *value.ReportCount, true
}

type GenerationHandlerCurrentGenerationIndexedByReportCount map[int]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) IndexByReportCount() (GenerationHandlerCurrentGenerationIndexedByReportCount, error) {
	result := make(GenerationHandlerCurrentGenerationIndexedByReportCount)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByReportCountKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index GenerationHandlerCurrentGenerationSlice.IndexByReportCount")
		}
		result[key] = row
	}
	return result, nil
}
func (index GenerationHandlerCurrentGenerationIndexedByReportCount) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type GenerationHandlerCurrentGenerationGroupedByReportCount map[int][]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) GroupByReportCount() GenerationHandlerCurrentGenerationGroupedByReportCount {
	result := make(GenerationHandlerCurrentGenerationGroupedByReportCount)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByReportCountKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index GenerationHandlerCurrentGenerationGroupedByReportCount) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func GenerationHandlerCurrentGenerationIndexByActivatedAtKey(value *CurrentGenerationView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.ActivatedAt == nil {
		return zero, false
	}
	return *value.ActivatedAt, true
}

type GenerationHandlerCurrentGenerationIndexedByActivatedAt map[time.Time]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) IndexByActivatedAt() (GenerationHandlerCurrentGenerationIndexedByActivatedAt, error) {
	result := make(GenerationHandlerCurrentGenerationIndexedByActivatedAt)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByActivatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index GenerationHandlerCurrentGenerationSlice.IndexByActivatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index GenerationHandlerCurrentGenerationIndexedByActivatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type GenerationHandlerCurrentGenerationGroupedByActivatedAt map[time.Time][]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) GroupByActivatedAt() GenerationHandlerCurrentGenerationGroupedByActivatedAt {
	result := make(GenerationHandlerCurrentGenerationGroupedByActivatedAt)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByActivatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index GenerationHandlerCurrentGenerationGroupedByActivatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func GenerationHandlerCurrentGenerationIndexByRetiredAtKey(value *CurrentGenerationView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.RetiredAt == nil {
		return zero, false
	}
	return *value.RetiredAt, true
}

type GenerationHandlerCurrentGenerationIndexedByRetiredAt map[time.Time]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) IndexByRetiredAt() (GenerationHandlerCurrentGenerationIndexedByRetiredAt, error) {
	result := make(GenerationHandlerCurrentGenerationIndexedByRetiredAt)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByRetiredAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index GenerationHandlerCurrentGenerationSlice.IndexByRetiredAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index GenerationHandlerCurrentGenerationIndexedByRetiredAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type GenerationHandlerCurrentGenerationGroupedByRetiredAt map[time.Time][]*CurrentGenerationView

func (rows GenerationHandlerCurrentGenerationSlice) GroupByRetiredAt() GenerationHandlerCurrentGenerationGroupedByRetiredAt {
	result := make(GenerationHandlerCurrentGenerationGroupedByRetiredAt)
	for _, row := range rows {
		key, ok := GenerationHandlerCurrentGenerationIndexByRetiredAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index GenerationHandlerCurrentGenerationGroupedByRetiredAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type GenerationHandlerReadIndexes struct {
	CurrentGeneration               GenerationHandlerCurrentGenerationSlice
	CurrentGenerationByGenerationNo GenerationHandlerCurrentGenerationIndexedByGenerationNo
}

func BuildGenerationHandlerReadIndexes(ctx context.Context, input *Input) (*GenerationHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &GenerationHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentGeneration")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentGeneration")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentGeneration")
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
			if _, err := (xshape.Collection[CurrentGenerationView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentGenerationView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentGenerationView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("BuildManifestJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentGeneration.BuildManifestJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DiagnosticsJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentGeneration.DiagnosticsJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("GenerationNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentGeneration.GenerationNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SourceRevision") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentGeneration.SourceRevision")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Status") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentGeneration.Status")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RequestedBy") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentGeneration.RequestedBy")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RequestedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentGeneration.RequestedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ReportCount") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentGeneration.ReportCount")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ActivatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentGeneration.ActivatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RetiredAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentGeneration.RetiredAt")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentGeneration = GenerationHandlerCurrentGenerationSlice(cloned.([]*CurrentGenerationView))
		result.CurrentGenerationByGenerationNo, err = result.CurrentGeneration.IndexByGenerationNo()
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
	input._generationHandlerReadIndexes = nil
	indexes, err := BuildGenerationHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._generationHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*GenerationHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._generationHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._generationHandlerReadIndexes, nil
}
