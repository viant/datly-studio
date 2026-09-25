package store_config

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
)

type ConnectorHandlerCurrentConnectorSlice []*CurrentConnectorView

func ConnectorHandlerCurrentConnectorIndexByNameKey(value *CurrentConnectorView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Name, true
}

type ConnectorHandlerCurrentConnectorIndexedByName map[string]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexByName() (ConnectorHandlerCurrentConnectorIndexedByName, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedByName)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexByName")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedByName map[string][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupByName() ConnectorHandlerCurrentConnectorGroupedByName {
	result := make(ConnectorHandlerCurrentConnectorGroupedByName)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ConnectorHandlerCurrentConnectorIndexByDriverKey(value *CurrentConnectorView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Driver, true
}

type ConnectorHandlerCurrentConnectorIndexedByDriver map[string]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexByDriver() (ConnectorHandlerCurrentConnectorIndexedByDriver, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedByDriver)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByDriverKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexByDriver")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedByDriver) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedByDriver map[string][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupByDriver() ConnectorHandlerCurrentConnectorGroupedByDriver {
	result := make(ConnectorHandlerCurrentConnectorGroupedByDriver)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByDriverKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedByDriver) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ConnectorHandlerCurrentConnectorIndexByDsnTemplateKey(value *CurrentConnectorView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.DsnTemplate == nil {
		return zero, false
	}
	return *value.DsnTemplate, true
}

type ConnectorHandlerCurrentConnectorIndexedByDsnTemplate map[string]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexByDsnTemplate() (ConnectorHandlerCurrentConnectorIndexedByDsnTemplate, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedByDsnTemplate)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByDsnTemplateKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexByDsnTemplate")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedByDsnTemplate) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedByDsnTemplate map[string][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupByDsnTemplate() ConnectorHandlerCurrentConnectorGroupedByDsnTemplate {
	result := make(ConnectorHandlerCurrentConnectorGroupedByDsnTemplate)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByDsnTemplateKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedByDsnTemplate) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ConnectorHandlerCurrentConnectorIndexBySecretRefKey(value *CurrentConnectorView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SecretRef == nil {
		return zero, false
	}
	return *value.SecretRef, true
}

type ConnectorHandlerCurrentConnectorIndexedBySecretRef map[string]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexBySecretRef() (ConnectorHandlerCurrentConnectorIndexedBySecretRef, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedBySecretRef)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexBySecretRefKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexBySecretRef")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedBySecretRef) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedBySecretRef map[string][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupBySecretRef() ConnectorHandlerCurrentConnectorGroupedBySecretRef {
	result := make(ConnectorHandlerCurrentConnectorGroupedBySecretRef)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexBySecretRefKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedBySecretRef) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ConnectorHandlerCurrentConnectorIndexByDescriptionKey(value *CurrentConnectorView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Description == nil {
		return zero, false
	}
	return *value.Description, true
}

type ConnectorHandlerCurrentConnectorIndexedByDescription map[string]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexByDescription() (ConnectorHandlerCurrentConnectorIndexedByDescription, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedByDescription)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByDescriptionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexByDescription")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedByDescription) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedByDescription map[string][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupByDescription() ConnectorHandlerCurrentConnectorGroupedByDescription {
	result := make(ConnectorHandlerCurrentConnectorGroupedByDescription)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByDescriptionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedByDescription) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ConnectorHandlerCurrentConnectorIndexByStatusKey(value *CurrentConnectorView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Status, true
}

type ConnectorHandlerCurrentConnectorIndexedByStatus map[string]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexByStatus() (ConnectorHandlerCurrentConnectorIndexedByStatus, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedByStatus)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByStatusKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexByStatus")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedByStatus map[string][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupByStatus() ConnectorHandlerCurrentConnectorGroupedByStatus {
	result := make(ConnectorHandlerCurrentConnectorGroupedByStatus)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByStatusKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ConnectorHandlerCurrentConnectorIndexByLastTestStatusKey(value *CurrentConnectorView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.LastTestStatus == nil {
		return zero, false
	}
	return *value.LastTestStatus, true
}

type ConnectorHandlerCurrentConnectorIndexedByLastTestStatus map[string]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexByLastTestStatus() (ConnectorHandlerCurrentConnectorIndexedByLastTestStatus, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedByLastTestStatus)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByLastTestStatusKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexByLastTestStatus")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedByLastTestStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedByLastTestStatus map[string][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupByLastTestStatus() ConnectorHandlerCurrentConnectorGroupedByLastTestStatus {
	result := make(ConnectorHandlerCurrentConnectorGroupedByLastTestStatus)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByLastTestStatusKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedByLastTestStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ConnectorHandlerCurrentConnectorIndexByLastTestErrorCodeKey(value *CurrentConnectorView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.LastTestErrorCode == nil {
		return zero, false
	}
	return *value.LastTestErrorCode, true
}

type ConnectorHandlerCurrentConnectorIndexedByLastTestErrorCode map[string]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexByLastTestErrorCode() (ConnectorHandlerCurrentConnectorIndexedByLastTestErrorCode, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedByLastTestErrorCode)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByLastTestErrorCodeKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexByLastTestErrorCode")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedByLastTestErrorCode) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedByLastTestErrorCode map[string][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupByLastTestErrorCode() ConnectorHandlerCurrentConnectorGroupedByLastTestErrorCode {
	result := make(ConnectorHandlerCurrentConnectorGroupedByLastTestErrorCode)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByLastTestErrorCodeKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedByLastTestErrorCode) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ConnectorHandlerCurrentConnectorIndexByLastTestedAtKey(value *CurrentConnectorView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.LastTestedAt == nil {
		return zero, false
	}
	return *value.LastTestedAt, true
}

type ConnectorHandlerCurrentConnectorIndexedByLastTestedAt map[time.Time]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexByLastTestedAt() (ConnectorHandlerCurrentConnectorIndexedByLastTestedAt, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedByLastTestedAt)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByLastTestedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexByLastTestedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedByLastTestedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedByLastTestedAt map[time.Time][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupByLastTestedAt() ConnectorHandlerCurrentConnectorGroupedByLastTestedAt {
	result := make(ConnectorHandlerCurrentConnectorGroupedByLastTestedAt)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByLastTestedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedByLastTestedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func ConnectorHandlerCurrentConnectorIndexByEtagKey(value *CurrentConnectorView) (int64, bool) {
	var zero int64
	if value == nil {
		return zero, false
	}
	if value.Etag == nil {
		return zero, false
	}
	return *value.Etag, true
}

type ConnectorHandlerCurrentConnectorIndexedByEtag map[int64]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexByEtag() (ConnectorHandlerCurrentConnectorIndexedByEtag, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedByEtag)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByEtagKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexByEtag")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedByEtag) Has(key int64) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedByEtag map[int64][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupByEtag() ConnectorHandlerCurrentConnectorGroupedByEtag {
	result := make(ConnectorHandlerCurrentConnectorGroupedByEtag)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByEtagKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedByEtag) Has(key int64) bool {
	_, ok := index[key]
	return ok
}
func ConnectorHandlerCurrentConnectorIndexByUpdatedAtKey(value *CurrentConnectorView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.UpdatedAt == nil {
		return zero, false
	}
	return *value.UpdatedAt, true
}

type ConnectorHandlerCurrentConnectorIndexedByUpdatedAt map[time.Time]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexByUpdatedAt() (ConnectorHandlerCurrentConnectorIndexedByUpdatedAt, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedByUpdatedAt)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexByUpdatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedByUpdatedAt map[time.Time][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupByUpdatedAt() ConnectorHandlerCurrentConnectorGroupedByUpdatedAt {
	result := make(ConnectorHandlerCurrentConnectorGroupedByUpdatedAt)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func ConnectorHandlerCurrentConnectorIndexByDeletedAtKey(value *CurrentConnectorView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.DeletedAt == nil {
		return zero, false
	}
	return *value.DeletedAt, true
}

type ConnectorHandlerCurrentConnectorIndexedByDeletedAt map[time.Time]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) IndexByDeletedAt() (ConnectorHandlerCurrentConnectorIndexedByDeletedAt, error) {
	result := make(ConnectorHandlerCurrentConnectorIndexedByDeletedAt)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByDeletedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConnectorHandlerCurrentConnectorSlice.IndexByDeletedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConnectorHandlerCurrentConnectorIndexedByDeletedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerCurrentConnectorGroupedByDeletedAt map[time.Time][]*CurrentConnectorView

func (rows ConnectorHandlerCurrentConnectorSlice) GroupByDeletedAt() ConnectorHandlerCurrentConnectorGroupedByDeletedAt {
	result := make(ConnectorHandlerCurrentConnectorGroupedByDeletedAt)
	for _, row := range rows {
		key, ok := ConnectorHandlerCurrentConnectorIndexByDeletedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConnectorHandlerCurrentConnectorGroupedByDeletedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type ConnectorHandlerReadIndexes struct {
	CurrentConnector       ConnectorHandlerCurrentConnectorSlice
	CurrentConnectorByName ConnectorHandlerCurrentConnectorIndexedByName
}

func BuildConnectorHandlerReadIndexes(ctx context.Context, input *Input) (*ConnectorHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &ConnectorHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentConnector")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentConnector")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentConnector")
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
			if _, err := (xshape.Collection[CurrentConnectorView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentConnectorView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentConnectorView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Name") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.Name")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Driver") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.Driver")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DsnTemplate") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.DsnTemplate")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SecretRef") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.SecretRef")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Description") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.Description")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("OptionsJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.OptionsJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Status") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.Status")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("LastTestStatus") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.LastTestStatus")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("LastTestErrorCode") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.LastTestErrorCode")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("LastTestedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.LastTestedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Etag") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.Etag")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("UpdatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.UpdatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DeletedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConnector.DeletedAt")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentConnector = ConnectorHandlerCurrentConnectorSlice(cloned.([]*CurrentConnectorView))
		result.CurrentConnectorByName, err = result.CurrentConnector.IndexByName()
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
	input._connectorHandlerReadIndexes = nil
	indexes, err := BuildConnectorHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._connectorHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*ConnectorHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._connectorHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._connectorHandlerReadIndexes, nil
}
