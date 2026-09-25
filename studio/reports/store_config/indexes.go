package store_config

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
func ReportHandlerCurrentReportIndexByNamespaceKey(value *CurrentReportView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Namespace, true
}

type ReportHandlerCurrentReportIndexedByNamespace map[string]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByNamespace() (ReportHandlerCurrentReportIndexedByNamespace, error) {
	result := make(ReportHandlerCurrentReportIndexedByNamespace)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByNamespaceKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByNamespace")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByNamespace) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByNamespace map[string][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByNamespace() ReportHandlerCurrentReportGroupedByNamespace {
	result := make(ReportHandlerCurrentReportGroupedByNamespace)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByNamespaceKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByNamespace) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ReportHandlerCurrentReportIndexBySlugKey(value *CurrentReportView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Slug, true
}

type ReportHandlerCurrentReportIndexedBySlug map[string]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexBySlug() (ReportHandlerCurrentReportIndexedBySlug, error) {
	result := make(ReportHandlerCurrentReportIndexedBySlug)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexBySlugKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexBySlug")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedBySlug) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedBySlug map[string][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupBySlug() ReportHandlerCurrentReportGroupedBySlug {
	result := make(ReportHandlerCurrentReportGroupedBySlug)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexBySlugKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedBySlug) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ReportHandlerCurrentReportIndexByTitleKey(value *CurrentReportView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Title, true
}

type ReportHandlerCurrentReportIndexedByTitle map[string]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByTitle() (ReportHandlerCurrentReportIndexedByTitle, error) {
	result := make(ReportHandlerCurrentReportIndexedByTitle)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByTitleKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByTitle")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByTitle) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByTitle map[string][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByTitle() ReportHandlerCurrentReportGroupedByTitle {
	result := make(ReportHandlerCurrentReportGroupedByTitle)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByTitleKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByTitle) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ReportHandlerCurrentReportIndexByDescriptionKey(value *CurrentReportView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Description == nil {
		return zero, false
	}
	return *value.Description, true
}

type ReportHandlerCurrentReportIndexedByDescription map[string]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByDescription() (ReportHandlerCurrentReportIndexedByDescription, error) {
	result := make(ReportHandlerCurrentReportIndexedByDescription)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByDescriptionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByDescription")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByDescription) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByDescription map[string][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByDescription() ReportHandlerCurrentReportGroupedByDescription {
	result := make(ReportHandlerCurrentReportGroupedByDescription)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByDescriptionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByDescription) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ReportHandlerCurrentReportIndexByOwnerIdKey(value *CurrentReportView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.OwnerId, true
}

type ReportHandlerCurrentReportIndexedByOwnerId map[string]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByOwnerId() (ReportHandlerCurrentReportIndexedByOwnerId, error) {
	result := make(ReportHandlerCurrentReportIndexedByOwnerId)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByOwnerIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByOwnerId")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByOwnerId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByOwnerId map[string][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByOwnerId() ReportHandlerCurrentReportGroupedByOwnerId {
	result := make(ReportHandlerCurrentReportGroupedByOwnerId)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByOwnerIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByOwnerId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ReportHandlerCurrentReportIndexByStatusKey(value *CurrentReportView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Status, true
}

type ReportHandlerCurrentReportIndexedByStatus map[string]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByStatus() (ReportHandlerCurrentReportIndexedByStatus, error) {
	result := make(ReportHandlerCurrentReportIndexedByStatus)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByStatusKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByStatus")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByStatus map[string][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByStatus() ReportHandlerCurrentReportGroupedByStatus {
	result := make(ReportHandlerCurrentReportGroupedByStatus)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByStatusKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ReportHandlerCurrentReportIndexByDefaultConnectorNameKey(value *CurrentReportView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.DefaultConnectorName, true
}

type ReportHandlerCurrentReportIndexedByDefaultConnectorName map[string]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByDefaultConnectorName() (ReportHandlerCurrentReportIndexedByDefaultConnectorName, error) {
	result := make(ReportHandlerCurrentReportIndexedByDefaultConnectorName)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByDefaultConnectorNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByDefaultConnectorName")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByDefaultConnectorName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByDefaultConnectorName map[string][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByDefaultConnectorName() ReportHandlerCurrentReportGroupedByDefaultConnectorName {
	result := make(ReportHandlerCurrentReportGroupedByDefaultConnectorName)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByDefaultConnectorNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByDefaultConnectorName) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ReportHandlerCurrentReportIndexByComponentScopeKey(value *CurrentReportView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ComponentScope, true
}

type ReportHandlerCurrentReportIndexedByComponentScope map[string]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByComponentScope() (ReportHandlerCurrentReportIndexedByComponentScope, error) {
	result := make(ReportHandlerCurrentReportIndexedByComponentScope)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByComponentScopeKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByComponentScope")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByComponentScope) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByComponentScope map[string][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByComponentScope() ReportHandlerCurrentReportGroupedByComponentScope {
	result := make(ReportHandlerCurrentReportGroupedByComponentScope)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByComponentScopeKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByComponentScope) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ReportHandlerCurrentReportIndexByComponentNameKey(value *CurrentReportView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ComponentName, true
}

type ReportHandlerCurrentReportIndexedByComponentName map[string]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByComponentName() (ReportHandlerCurrentReportIndexedByComponentName, error) {
	result := make(ReportHandlerCurrentReportIndexedByComponentName)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByComponentNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByComponentName")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByComponentName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByComponentName map[string][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByComponentName() ReportHandlerCurrentReportGroupedByComponentName {
	result := make(ReportHandlerCurrentReportGroupedByComponentName)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByComponentNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByComponentName) Has(key string) bool {
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
func ReportHandlerCurrentReportIndexByDeletedAtKey(value *CurrentReportView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.DeletedAt == nil {
		return zero, false
	}
	return *value.DeletedAt, true
}

type ReportHandlerCurrentReportIndexedByDeletedAt map[time.Time]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) IndexByDeletedAt() (ReportHandlerCurrentReportIndexedByDeletedAt, error) {
	result := make(ReportHandlerCurrentReportIndexedByDeletedAt)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByDeletedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ReportHandlerCurrentReportSlice.IndexByDeletedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index ReportHandlerCurrentReportIndexedByDeletedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type ReportHandlerCurrentReportGroupedByDeletedAt map[time.Time][]*CurrentReportView

func (rows ReportHandlerCurrentReportSlice) GroupByDeletedAt() ReportHandlerCurrentReportGroupedByDeletedAt {
	result := make(ReportHandlerCurrentReportGroupedByDeletedAt)
	for _, row := range rows {
		key, ok := ReportHandlerCurrentReportIndexByDeletedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ReportHandlerCurrentReportGroupedByDeletedAt) Has(key time.Time) bool {
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Namespace") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.Namespace")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Slug") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.Slug")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Title") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.Title")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Description") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.Description")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("OwnerId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.OwnerId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Status") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.Status")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DefaultConnectorName") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.DefaultConnectorName")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ComponentScope") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.ComponentScope")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ComponentName") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.ComponentName")
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DeletedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentReport.DeletedAt")
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
