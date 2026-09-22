package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
)

type PublicationHandlerCurrentPublicationSlice []*CurrentPublicationView

func PublicationHandlerCurrentPublicationIndexByReportIdKey(value *CurrentPublicationView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	return *value.ReportId, true
}

type PublicationHandlerCurrentPublicationIndexedByReportId map[string]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) IndexByReportId() (PublicationHandlerCurrentPublicationIndexedByReportId, error) {
	result := make(PublicationHandlerCurrentPublicationIndexedByReportId)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PublicationHandlerCurrentPublicationSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index PublicationHandlerCurrentPublicationIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerCurrentPublicationGroupedByReportId map[string][]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) GroupByReportId() PublicationHandlerCurrentPublicationGroupedByReportId {
	result := make(PublicationHandlerCurrentPublicationGroupedByReportId)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PublicationHandlerCurrentPublicationGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PublicationHandlerCurrentPublicationIndexByActiveVersionNoKey(value *CurrentPublicationView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.ActiveVersionNo == nil {
		return zero, false
	}
	return *value.ActiveVersionNo, true
}

type PublicationHandlerCurrentPublicationIndexedByActiveVersionNo map[int]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) IndexByActiveVersionNo() (PublicationHandlerCurrentPublicationIndexedByActiveVersionNo, error) {
	result := make(PublicationHandlerCurrentPublicationIndexedByActiveVersionNo)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByActiveVersionNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PublicationHandlerCurrentPublicationSlice.IndexByActiveVersionNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index PublicationHandlerCurrentPublicationIndexedByActiveVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerCurrentPublicationGroupedByActiveVersionNo map[int][]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) GroupByActiveVersionNo() PublicationHandlerCurrentPublicationGroupedByActiveVersionNo {
	result := make(PublicationHandlerCurrentPublicationGroupedByActiveVersionNo)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByActiveVersionNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PublicationHandlerCurrentPublicationGroupedByActiveVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func PublicationHandlerCurrentPublicationIndexByDesiredGenerationKey(value *CurrentPublicationView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.DesiredGeneration == nil {
		return zero, false
	}
	return *value.DesiredGeneration, true
}

type PublicationHandlerCurrentPublicationIndexedByDesiredGeneration map[int]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) IndexByDesiredGeneration() (PublicationHandlerCurrentPublicationIndexedByDesiredGeneration, error) {
	result := make(PublicationHandlerCurrentPublicationIndexedByDesiredGeneration)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByDesiredGenerationKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PublicationHandlerCurrentPublicationSlice.IndexByDesiredGeneration")
		}
		result[key] = row
	}
	return result, nil
}
func (index PublicationHandlerCurrentPublicationIndexedByDesiredGeneration) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerCurrentPublicationGroupedByDesiredGeneration map[int][]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) GroupByDesiredGeneration() PublicationHandlerCurrentPublicationGroupedByDesiredGeneration {
	result := make(PublicationHandlerCurrentPublicationGroupedByDesiredGeneration)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByDesiredGenerationKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PublicationHandlerCurrentPublicationGroupedByDesiredGeneration) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func PublicationHandlerCurrentPublicationIndexByPublicationStatusKey(value *CurrentPublicationView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.PublicationStatus == nil {
		return zero, false
	}
	return *value.PublicationStatus, true
}

type PublicationHandlerCurrentPublicationIndexedByPublicationStatus map[string]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) IndexByPublicationStatus() (PublicationHandlerCurrentPublicationIndexedByPublicationStatus, error) {
	result := make(PublicationHandlerCurrentPublicationIndexedByPublicationStatus)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByPublicationStatusKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PublicationHandlerCurrentPublicationSlice.IndexByPublicationStatus")
		}
		result[key] = row
	}
	return result, nil
}
func (index PublicationHandlerCurrentPublicationIndexedByPublicationStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerCurrentPublicationGroupedByPublicationStatus map[string][]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) GroupByPublicationStatus() PublicationHandlerCurrentPublicationGroupedByPublicationStatus {
	result := make(PublicationHandlerCurrentPublicationGroupedByPublicationStatus)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByPublicationStatusKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PublicationHandlerCurrentPublicationGroupedByPublicationStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PublicationHandlerCurrentPublicationIndexBySpecHashKey(value *CurrentPublicationView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SpecHash == nil {
		return zero, false
	}
	return *value.SpecHash, true
}

type PublicationHandlerCurrentPublicationIndexedBySpecHash map[string]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) IndexBySpecHash() (PublicationHandlerCurrentPublicationIndexedBySpecHash, error) {
	result := make(PublicationHandlerCurrentPublicationIndexedBySpecHash)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexBySpecHashKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PublicationHandlerCurrentPublicationSlice.IndexBySpecHash")
		}
		result[key] = row
	}
	return result, nil
}
func (index PublicationHandlerCurrentPublicationIndexedBySpecHash) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerCurrentPublicationGroupedBySpecHash map[string][]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) GroupBySpecHash() PublicationHandlerCurrentPublicationGroupedBySpecHash {
	result := make(PublicationHandlerCurrentPublicationGroupedBySpecHash)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexBySpecHashKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PublicationHandlerCurrentPublicationGroupedBySpecHash) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PublicationHandlerCurrentPublicationIndexByPublishedByKey(value *CurrentPublicationView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.PublishedBy == nil {
		return zero, false
	}
	return *value.PublishedBy, true
}

type PublicationHandlerCurrentPublicationIndexedByPublishedBy map[string]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) IndexByPublishedBy() (PublicationHandlerCurrentPublicationIndexedByPublishedBy, error) {
	result := make(PublicationHandlerCurrentPublicationIndexedByPublishedBy)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByPublishedByKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PublicationHandlerCurrentPublicationSlice.IndexByPublishedBy")
		}
		result[key] = row
	}
	return result, nil
}
func (index PublicationHandlerCurrentPublicationIndexedByPublishedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerCurrentPublicationGroupedByPublishedBy map[string][]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) GroupByPublishedBy() PublicationHandlerCurrentPublicationGroupedByPublishedBy {
	result := make(PublicationHandlerCurrentPublicationGroupedByPublishedBy)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByPublishedByKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PublicationHandlerCurrentPublicationGroupedByPublishedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PublicationHandlerCurrentPublicationIndexByPublishedAtKey(value *CurrentPublicationView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.PublishedAt == nil {
		return zero, false
	}
	return *value.PublishedAt, true
}

type PublicationHandlerCurrentPublicationIndexedByPublishedAt map[time.Time]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) IndexByPublishedAt() (PublicationHandlerCurrentPublicationIndexedByPublishedAt, error) {
	result := make(PublicationHandlerCurrentPublicationIndexedByPublishedAt)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByPublishedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PublicationHandlerCurrentPublicationSlice.IndexByPublishedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index PublicationHandlerCurrentPublicationIndexedByPublishedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerCurrentPublicationGroupedByPublishedAt map[time.Time][]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) GroupByPublishedAt() PublicationHandlerCurrentPublicationGroupedByPublishedAt {
	result := make(PublicationHandlerCurrentPublicationGroupedByPublishedAt)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByPublishedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PublicationHandlerCurrentPublicationGroupedByPublishedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func PublicationHandlerCurrentPublicationIndexByActiveGenerationKey(value *CurrentPublicationView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.ActiveGeneration == nil {
		return zero, false
	}
	return *value.ActiveGeneration, true
}

type PublicationHandlerCurrentPublicationIndexedByActiveGeneration map[int]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) IndexByActiveGeneration() (PublicationHandlerCurrentPublicationIndexedByActiveGeneration, error) {
	result := make(PublicationHandlerCurrentPublicationIndexedByActiveGeneration)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByActiveGenerationKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PublicationHandlerCurrentPublicationSlice.IndexByActiveGeneration")
		}
		result[key] = row
	}
	return result, nil
}
func (index PublicationHandlerCurrentPublicationIndexedByActiveGeneration) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerCurrentPublicationGroupedByActiveGeneration map[int][]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) GroupByActiveGeneration() PublicationHandlerCurrentPublicationGroupedByActiveGeneration {
	result := make(PublicationHandlerCurrentPublicationGroupedByActiveGeneration)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByActiveGenerationKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PublicationHandlerCurrentPublicationGroupedByActiveGeneration) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func PublicationHandlerCurrentPublicationIndexByRuntimeRevisionKey(value *CurrentPublicationView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.RuntimeRevision == nil {
		return zero, false
	}
	return *value.RuntimeRevision, true
}

type PublicationHandlerCurrentPublicationIndexedByRuntimeRevision map[string]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) IndexByRuntimeRevision() (PublicationHandlerCurrentPublicationIndexedByRuntimeRevision, error) {
	result := make(PublicationHandlerCurrentPublicationIndexedByRuntimeRevision)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByRuntimeRevisionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PublicationHandlerCurrentPublicationSlice.IndexByRuntimeRevision")
		}
		result[key] = row
	}
	return result, nil
}
func (index PublicationHandlerCurrentPublicationIndexedByRuntimeRevision) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerCurrentPublicationGroupedByRuntimeRevision map[string][]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) GroupByRuntimeRevision() PublicationHandlerCurrentPublicationGroupedByRuntimeRevision {
	result := make(PublicationHandlerCurrentPublicationGroupedByRuntimeRevision)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByRuntimeRevisionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PublicationHandlerCurrentPublicationGroupedByRuntimeRevision) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PublicationHandlerCurrentPublicationIndexByActivatedAtKey(value *CurrentPublicationView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.ActivatedAt == nil {
		return zero, false
	}
	return *value.ActivatedAt, true
}

type PublicationHandlerCurrentPublicationIndexedByActivatedAt map[time.Time]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) IndexByActivatedAt() (PublicationHandlerCurrentPublicationIndexedByActivatedAt, error) {
	result := make(PublicationHandlerCurrentPublicationIndexedByActivatedAt)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByActivatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PublicationHandlerCurrentPublicationSlice.IndexByActivatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index PublicationHandlerCurrentPublicationIndexedByActivatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerCurrentPublicationGroupedByActivatedAt map[time.Time][]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) GroupByActivatedAt() PublicationHandlerCurrentPublicationGroupedByActivatedAt {
	result := make(PublicationHandlerCurrentPublicationGroupedByActivatedAt)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByActivatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PublicationHandlerCurrentPublicationGroupedByActivatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func PublicationHandlerCurrentPublicationIndexByDesiredVersionNoKey(value *CurrentPublicationView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.DesiredVersionNo == nil {
		return zero, false
	}
	return *value.DesiredVersionNo, true
}

type PublicationHandlerCurrentPublicationIndexedByDesiredVersionNo map[int]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) IndexByDesiredVersionNo() (PublicationHandlerCurrentPublicationIndexedByDesiredVersionNo, error) {
	result := make(PublicationHandlerCurrentPublicationIndexedByDesiredVersionNo)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByDesiredVersionNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PublicationHandlerCurrentPublicationSlice.IndexByDesiredVersionNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index PublicationHandlerCurrentPublicationIndexedByDesiredVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerCurrentPublicationGroupedByDesiredVersionNo map[int][]*CurrentPublicationView

func (rows PublicationHandlerCurrentPublicationSlice) GroupByDesiredVersionNo() PublicationHandlerCurrentPublicationGroupedByDesiredVersionNo {
	result := make(PublicationHandlerCurrentPublicationGroupedByDesiredVersionNo)
	for _, row := range rows {
		key, ok := PublicationHandlerCurrentPublicationIndexByDesiredVersionNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PublicationHandlerCurrentPublicationGroupedByDesiredVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type PublicationHandlerReadIndexes struct {
	CurrentPublication           PublicationHandlerCurrentPublicationSlice
	CurrentPublicationByReportId PublicationHandlerCurrentPublicationIndexedByReportId
}

func BuildPublicationHandlerReadIndexes(ctx context.Context, input *Input) (*PublicationHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &PublicationHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentPublication")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentPublication")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentPublication")
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
			if _, err := (xshape.Collection[CurrentPublicationView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentPublicationView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentPublicationView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("FailureJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.FailureJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ReportId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ActiveVersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.ActiveVersionNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DesiredGeneration") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.DesiredGeneration")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("PublicationStatus") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.PublicationStatus")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SpecHash") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.SpecHash")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("PublishedBy") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.PublishedBy")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("PublishedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.PublishedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ActiveGeneration") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.ActiveGeneration")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RuntimeRevision") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.RuntimeRevision")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ActivatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.ActivatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DesiredVersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPublication.DesiredVersionNo")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentPublication = PublicationHandlerCurrentPublicationSlice(cloned.([]*CurrentPublicationView))
		result.CurrentPublicationByReportId, err = result.CurrentPublication.IndexByReportId()
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
	input._publicationHandlerReadIndexes = nil
	indexes, err := BuildPublicationHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._publicationHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*PublicationHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._publicationHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._publicationHandlerReadIndexes, nil
}
