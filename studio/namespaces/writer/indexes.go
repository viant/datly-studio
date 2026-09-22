package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
)

type NamespaceHandlerCurrentNamespaceSlice []*CurrentNamespaceView

func NamespaceHandlerCurrentNamespaceIndexByOwnerIdKey(value *CurrentNamespaceView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.OwnerId, true
}

type NamespaceHandlerCurrentNamespaceIndexedByOwnerId map[string]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) IndexByOwnerId() (NamespaceHandlerCurrentNamespaceIndexedByOwnerId, error) {
	result := make(NamespaceHandlerCurrentNamespaceIndexedByOwnerId)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByOwnerIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index NamespaceHandlerCurrentNamespaceSlice.IndexByOwnerId")
		}
		result[key] = row
	}
	return result, nil
}
func (index NamespaceHandlerCurrentNamespaceIndexedByOwnerId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerCurrentNamespaceGroupedByOwnerId map[string][]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) GroupByOwnerId() NamespaceHandlerCurrentNamespaceGroupedByOwnerId {
	result := make(NamespaceHandlerCurrentNamespaceGroupedByOwnerId)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByOwnerIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index NamespaceHandlerCurrentNamespaceGroupedByOwnerId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func NamespaceHandlerCurrentNamespaceIndexByNameKey(value *CurrentNamespaceView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Name, true
}

type NamespaceHandlerCurrentNamespaceIndexedByName map[string]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) IndexByName() (NamespaceHandlerCurrentNamespaceIndexedByName, error) {
	result := make(NamespaceHandlerCurrentNamespaceIndexedByName)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index NamespaceHandlerCurrentNamespaceSlice.IndexByName")
		}
		result[key] = row
	}
	return result, nil
}
func (index NamespaceHandlerCurrentNamespaceIndexedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerCurrentNamespaceGroupedByName map[string][]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) GroupByName() NamespaceHandlerCurrentNamespaceGroupedByName {
	result := make(NamespaceHandlerCurrentNamespaceGroupedByName)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index NamespaceHandlerCurrentNamespaceGroupedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func NamespaceHandlerCurrentNamespaceIndexByTitleKey(value *CurrentNamespaceView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Title, true
}

type NamespaceHandlerCurrentNamespaceIndexedByTitle map[string]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) IndexByTitle() (NamespaceHandlerCurrentNamespaceIndexedByTitle, error) {
	result := make(NamespaceHandlerCurrentNamespaceIndexedByTitle)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByTitleKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index NamespaceHandlerCurrentNamespaceSlice.IndexByTitle")
		}
		result[key] = row
	}
	return result, nil
}
func (index NamespaceHandlerCurrentNamespaceIndexedByTitle) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerCurrentNamespaceGroupedByTitle map[string][]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) GroupByTitle() NamespaceHandlerCurrentNamespaceGroupedByTitle {
	result := make(NamespaceHandlerCurrentNamespaceGroupedByTitle)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByTitleKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index NamespaceHandlerCurrentNamespaceGroupedByTitle) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func NamespaceHandlerCurrentNamespaceIndexByStatusKey(value *CurrentNamespaceView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Status, true
}

type NamespaceHandlerCurrentNamespaceIndexedByStatus map[string]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) IndexByStatus() (NamespaceHandlerCurrentNamespaceIndexedByStatus, error) {
	result := make(NamespaceHandlerCurrentNamespaceIndexedByStatus)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByStatusKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index NamespaceHandlerCurrentNamespaceSlice.IndexByStatus")
		}
		result[key] = row
	}
	return result, nil
}
func (index NamespaceHandlerCurrentNamespaceIndexedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerCurrentNamespaceGroupedByStatus map[string][]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) GroupByStatus() NamespaceHandlerCurrentNamespaceGroupedByStatus {
	result := make(NamespaceHandlerCurrentNamespaceGroupedByStatus)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByStatusKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index NamespaceHandlerCurrentNamespaceGroupedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func NamespaceHandlerCurrentNamespaceIndexByEtagKey(value *CurrentNamespaceView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.Etag == nil {
		return zero, false
	}
	return *value.Etag, true
}

type NamespaceHandlerCurrentNamespaceIndexedByEtag map[int]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) IndexByEtag() (NamespaceHandlerCurrentNamespaceIndexedByEtag, error) {
	result := make(NamespaceHandlerCurrentNamespaceIndexedByEtag)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByEtagKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index NamespaceHandlerCurrentNamespaceSlice.IndexByEtag")
		}
		result[key] = row
	}
	return result, nil
}
func (index NamespaceHandlerCurrentNamespaceIndexedByEtag) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerCurrentNamespaceGroupedByEtag map[int][]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) GroupByEtag() NamespaceHandlerCurrentNamespaceGroupedByEtag {
	result := make(NamespaceHandlerCurrentNamespaceGroupedByEtag)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByEtagKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index NamespaceHandlerCurrentNamespaceGroupedByEtag) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func NamespaceHandlerCurrentNamespaceIndexByDescriptionKey(value *CurrentNamespaceView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Description == nil {
		return zero, false
	}
	return *value.Description, true
}

type NamespaceHandlerCurrentNamespaceIndexedByDescription map[string]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) IndexByDescription() (NamespaceHandlerCurrentNamespaceIndexedByDescription, error) {
	result := make(NamespaceHandlerCurrentNamespaceIndexedByDescription)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByDescriptionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index NamespaceHandlerCurrentNamespaceSlice.IndexByDescription")
		}
		result[key] = row
	}
	return result, nil
}
func (index NamespaceHandlerCurrentNamespaceIndexedByDescription) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerCurrentNamespaceGroupedByDescription map[string][]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) GroupByDescription() NamespaceHandlerCurrentNamespaceGroupedByDescription {
	result := make(NamespaceHandlerCurrentNamespaceGroupedByDescription)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByDescriptionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index NamespaceHandlerCurrentNamespaceGroupedByDescription) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func NamespaceHandlerCurrentNamespaceIndexByCreatedAtKey(value *CurrentNamespaceView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.CreatedAt == nil {
		return zero, false
	}
	return *value.CreatedAt, true
}

type NamespaceHandlerCurrentNamespaceIndexedByCreatedAt map[time.Time]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) IndexByCreatedAt() (NamespaceHandlerCurrentNamespaceIndexedByCreatedAt, error) {
	result := make(NamespaceHandlerCurrentNamespaceIndexedByCreatedAt)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index NamespaceHandlerCurrentNamespaceSlice.IndexByCreatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index NamespaceHandlerCurrentNamespaceIndexedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerCurrentNamespaceGroupedByCreatedAt map[time.Time][]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) GroupByCreatedAt() NamespaceHandlerCurrentNamespaceGroupedByCreatedAt {
	result := make(NamespaceHandlerCurrentNamespaceGroupedByCreatedAt)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index NamespaceHandlerCurrentNamespaceGroupedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func NamespaceHandlerCurrentNamespaceIndexByUpdatedAtKey(value *CurrentNamespaceView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.UpdatedAt == nil {
		return zero, false
	}
	return *value.UpdatedAt, true
}

type NamespaceHandlerCurrentNamespaceIndexedByUpdatedAt map[time.Time]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) IndexByUpdatedAt() (NamespaceHandlerCurrentNamespaceIndexedByUpdatedAt, error) {
	result := make(NamespaceHandlerCurrentNamespaceIndexedByUpdatedAt)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index NamespaceHandlerCurrentNamespaceSlice.IndexByUpdatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index NamespaceHandlerCurrentNamespaceIndexedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerCurrentNamespaceGroupedByUpdatedAt map[time.Time][]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) GroupByUpdatedAt() NamespaceHandlerCurrentNamespaceGroupedByUpdatedAt {
	result := make(NamespaceHandlerCurrentNamespaceGroupedByUpdatedAt)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index NamespaceHandlerCurrentNamespaceGroupedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func NamespaceHandlerCurrentNamespaceIndexByDeletedAtKey(value *CurrentNamespaceView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.DeletedAt == nil {
		return zero, false
	}
	return *value.DeletedAt, true
}

type NamespaceHandlerCurrentNamespaceIndexedByDeletedAt map[time.Time]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) IndexByDeletedAt() (NamespaceHandlerCurrentNamespaceIndexedByDeletedAt, error) {
	result := make(NamespaceHandlerCurrentNamespaceIndexedByDeletedAt)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByDeletedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index NamespaceHandlerCurrentNamespaceSlice.IndexByDeletedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index NamespaceHandlerCurrentNamespaceIndexedByDeletedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerCurrentNamespaceGroupedByDeletedAt map[time.Time][]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) GroupByDeletedAt() NamespaceHandlerCurrentNamespaceGroupedByDeletedAt {
	result := make(NamespaceHandlerCurrentNamespaceGroupedByDeletedAt)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByDeletedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index NamespaceHandlerCurrentNamespaceGroupedByDeletedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerCurrentNamespaceKey struct {
	OwnerId string
	Name    string
}

func NamespaceHandlerCurrentNamespaceIndexByKeyKey(value *CurrentNamespaceView) (NamespaceHandlerCurrentNamespaceKey, bool) {
	var zero NamespaceHandlerCurrentNamespaceKey
	if value == nil {
		return zero, false
	}
	return NamespaceHandlerCurrentNamespaceKey{OwnerId: value.OwnerId, Name: value.Name}, true
}

type NamespaceHandlerCurrentNamespaceIndexedByKey map[NamespaceHandlerCurrentNamespaceKey]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) IndexByKey() (NamespaceHandlerCurrentNamespaceIndexedByKey, error) {
	result := make(NamespaceHandlerCurrentNamespaceIndexedByKey)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index NamespaceHandlerCurrentNamespaceSlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index NamespaceHandlerCurrentNamespaceIndexedByKey) Has(key NamespaceHandlerCurrentNamespaceKey) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerCurrentNamespaceGroupedByKey map[NamespaceHandlerCurrentNamespaceKey][]*CurrentNamespaceView

func (rows NamespaceHandlerCurrentNamespaceSlice) GroupByKey() NamespaceHandlerCurrentNamespaceGroupedByKey {
	result := make(NamespaceHandlerCurrentNamespaceGroupedByKey)
	for _, row := range rows {
		key, ok := NamespaceHandlerCurrentNamespaceIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index NamespaceHandlerCurrentNamespaceGroupedByKey) Has(key NamespaceHandlerCurrentNamespaceKey) bool {
	_, ok := index[key]
	return ok
}

type NamespaceHandlerReadIndexes struct {
	CurrentNamespace      NamespaceHandlerCurrentNamespaceSlice
	CurrentNamespaceByKey NamespaceHandlerCurrentNamespaceIndexedByKey
}

func BuildNamespaceHandlerReadIndexes(ctx context.Context, input *NamespaceMutationInput) (*NamespaceHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &NamespaceHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentNamespace")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentNamespace")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*NamespaceMutationInput)(nil)).Elem()).Accessor("CurrentNamespace")
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
			if _, err := (xshape.Collection[CurrentNamespaceView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentNamespaceView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentNamespaceView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("OwnerId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentNamespace.OwnerId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Name") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentNamespace.Name")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Title") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentNamespace.Title")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Status") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentNamespace.Status")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Etag") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentNamespace.Etag")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Description") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentNamespace.Description")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CreatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentNamespace.CreatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("UpdatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentNamespace.UpdatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DeletedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentNamespace.DeletedAt")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentNamespace = NamespaceHandlerCurrentNamespaceSlice(cloned.([]*CurrentNamespaceView))
		result.CurrentNamespaceByKey, err = result.CurrentNamespace.IndexByKey()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
func (input *NamespaceMutationInput) PrepareReadIndexes(ctx context.Context) error {
	if input == nil {
		return fmt.Errorf("read indexes require input")
	}
	input._namespaceHandlerReadIndexes = nil
	indexes, err := BuildNamespaceHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._namespaceHandlerReadIndexes = indexes
	return nil
}
func (input *NamespaceMutationInput) ReadIndexes(ctx context.Context) (*NamespaceHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._namespaceHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._namespaceHandlerReadIndexes, nil
}
