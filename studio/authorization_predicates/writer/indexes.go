package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
)

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice []*CurrentAuthorizationPredicateView

func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByNameKey(value *CurrentAuthorizationPredicateView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Name, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByName map[string]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexByName() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByName, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByName)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexByName")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByName map[string][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupByName() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByName {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByName)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByTitleKey(value *CurrentAuthorizationPredicateView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Title, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByTitle map[string]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexByTitle() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByTitle, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByTitle)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByTitleKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexByTitle")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByTitle) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByTitle map[string][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupByTitle() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByTitle {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByTitle)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByTitleKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByTitle) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByPackagePathKey(value *CurrentAuthorizationPredicateView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.PackagePath, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByPackagePath map[string]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexByPackagePath() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByPackagePath, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByPackagePath)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByPackagePathKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexByPackagePath")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByPackagePath) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByPackagePath map[string][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupByPackagePath() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByPackagePath {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByPackagePath)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByPackagePathKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByPackagePath) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByTypeNameKey(value *CurrentAuthorizationPredicateView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.TypeName, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByTypeName map[string]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexByTypeName() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByTypeName, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByTypeName)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByTypeNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexByTypeName")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByTypeName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByTypeName map[string][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupByTypeName() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByTypeName {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByTypeName)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByTypeNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByTypeName) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexBySqlScopeJsonKey(value *CurrentAuthorizationPredicateView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SqlScopeJson == nil {
		return zero, false
	}
	return *value.SqlScopeJson, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedBySqlScopeJson map[string]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexBySqlScopeJson() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedBySqlScopeJson, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedBySqlScopeJson)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexBySqlScopeJsonKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexBySqlScopeJson")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedBySqlScopeJson) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedBySqlScopeJson map[string][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupBySqlScopeJson() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedBySqlScopeJson {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedBySqlScopeJson)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexBySqlScopeJsonKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedBySqlScopeJson) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByOwnerIdKey(value *CurrentAuthorizationPredicateView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.OwnerId, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByOwnerId map[string]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexByOwnerId() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByOwnerId, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByOwnerId)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByOwnerIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexByOwnerId")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByOwnerId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByOwnerId map[string][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupByOwnerId() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByOwnerId {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByOwnerId)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByOwnerIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByOwnerId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByStatusKey(value *CurrentAuthorizationPredicateView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Status, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByStatus map[string]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexByStatus() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByStatus, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByStatus)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByStatusKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexByStatus")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByStatus map[string][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupByStatus() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByStatus {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByStatus)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByStatusKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByEtagKey(value *CurrentAuthorizationPredicateView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.Etag == nil {
		return zero, false
	}
	return *value.Etag, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByEtag map[int]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexByEtag() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByEtag, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByEtag)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByEtagKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexByEtag")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByEtag) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByEtag map[int][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupByEtag() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByEtag {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByEtag)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByEtagKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByEtag) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByDescriptionKey(value *CurrentAuthorizationPredicateView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Description == nil {
		return zero, false
	}
	return *value.Description, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByDescription map[string]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexByDescription() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByDescription, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByDescription)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByDescriptionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexByDescription")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByDescription) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByDescription map[string][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupByDescription() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByDescription {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByDescription)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByDescriptionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByDescription) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByCreatedAtKey(value *CurrentAuthorizationPredicateView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.CreatedAt == nil {
		return zero, false
	}
	return *value.CreatedAt, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByCreatedAt map[time.Time]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexByCreatedAt() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByCreatedAt, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByCreatedAt)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexByCreatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByCreatedAt map[time.Time][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupByCreatedAt() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByCreatedAt {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByCreatedAt)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByUpdatedAtKey(value *CurrentAuthorizationPredicateView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.UpdatedAt == nil {
		return zero, false
	}
	return *value.UpdatedAt, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByUpdatedAt map[time.Time]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexByUpdatedAt() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByUpdatedAt, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByUpdatedAt)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexByUpdatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByUpdatedAt map[time.Time][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupByUpdatedAt() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByUpdatedAt {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByUpdatedAt)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByDeletedAtKey(value *CurrentAuthorizationPredicateView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.DeletedAt == nil {
		return zero, false
	}
	return *value.DeletedAt, true
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByDeletedAt map[time.Time]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) IndexByDeletedAt() (AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByDeletedAt, error) {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByDeletedAt)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByDeletedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice.IndexByDeletedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByDeletedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByDeletedAt map[time.Time][]*CurrentAuthorizationPredicateView

func (rows AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice) GroupByDeletedAt() AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByDeletedAt {
	result := make(AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByDeletedAt)
	for _, row := range rows {
		key, ok := AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexByDeletedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AuthorizationPredicateHandlerCurrentAuthorizationPredicateGroupedByDeletedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type AuthorizationPredicateHandlerReadIndexes struct {
	CurrentAuthorizationPredicate       AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice
	CurrentAuthorizationPredicateByName AuthorizationPredicateHandlerCurrentAuthorizationPredicateIndexedByName
}

func BuildAuthorizationPredicateHandlerReadIndexes(ctx context.Context, input *AuthorizationPredicateMutationInput) (*AuthorizationPredicateHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &AuthorizationPredicateHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentAuthorizationPredicate")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentAuthorizationPredicate")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*AuthorizationPredicateMutationInput)(nil)).Elem()).Accessor("CurrentAuthorizationPredicate")
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
			if _, err := (xshape.Collection[CurrentAuthorizationPredicateView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentAuthorizationPredicateView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentAuthorizationPredicateView]{}).Pointers(value)
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
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.Name")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Title") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.Title")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("PackagePath") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.PackagePath")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("TypeName") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.TypeName")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SqlScopeJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.SqlScopeJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("OwnerId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.OwnerId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Status") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.Status")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Etag") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.Etag")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Description") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.Description")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CreatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.CreatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("UpdatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.UpdatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DeletedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAuthorizationPredicate.DeletedAt")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentAuthorizationPredicate = AuthorizationPredicateHandlerCurrentAuthorizationPredicateSlice(cloned.([]*CurrentAuthorizationPredicateView))
		result.CurrentAuthorizationPredicateByName, err = result.CurrentAuthorizationPredicate.IndexByName()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
func (input *AuthorizationPredicateMutationInput) PrepareReadIndexes(ctx context.Context) error {
	if input == nil {
		return fmt.Errorf("read indexes require input")
	}
	input._authorizationPredicateHandlerReadIndexes = nil
	indexes, err := BuildAuthorizationPredicateHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._authorizationPredicateHandlerReadIndexes = indexes
	return nil
}
func (input *AuthorizationPredicateMutationInput) ReadIndexes(ctx context.Context) (*AuthorizationPredicateHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._authorizationPredicateHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._authorizationPredicateHandlerReadIndexes, nil
}
