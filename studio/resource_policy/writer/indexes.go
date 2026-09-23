package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
)

type PolicyHandlerCurrentPolicySlice []*CurrentPolicyView

func PolicyHandlerCurrentPolicyIndexByTenantIdKey(value *CurrentPolicyView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.TenantId, true
}

type PolicyHandlerCurrentPolicyIndexedByTenantId map[string]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) IndexByTenantId() (PolicyHandlerCurrentPolicyIndexedByTenantId, error) {
	result := make(PolicyHandlerCurrentPolicyIndexedByTenantId)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByTenantIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentPolicySlice.IndexByTenantId")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentPolicyIndexedByTenantId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentPolicyGroupedByTenantId map[string][]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) GroupByTenantId() PolicyHandlerCurrentPolicyGroupedByTenantId {
	result := make(PolicyHandlerCurrentPolicyGroupedByTenantId)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByTenantIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentPolicyGroupedByTenantId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PolicyHandlerCurrentPolicyIndexByResourceKindKey(value *CurrentPolicyView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ResourceKind, true
}

type PolicyHandlerCurrentPolicyIndexedByResourceKind map[string]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) IndexByResourceKind() (PolicyHandlerCurrentPolicyIndexedByResourceKind, error) {
	result := make(PolicyHandlerCurrentPolicyIndexedByResourceKind)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByResourceKindKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentPolicySlice.IndexByResourceKind")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentPolicyIndexedByResourceKind) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentPolicyGroupedByResourceKind map[string][]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) GroupByResourceKind() PolicyHandlerCurrentPolicyGroupedByResourceKind {
	result := make(PolicyHandlerCurrentPolicyGroupedByResourceKind)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByResourceKindKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentPolicyGroupedByResourceKind) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PolicyHandlerCurrentPolicyIndexByResourceIdKey(value *CurrentPolicyView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ResourceId, true
}

type PolicyHandlerCurrentPolicyIndexedByResourceId map[string]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) IndexByResourceId() (PolicyHandlerCurrentPolicyIndexedByResourceId, error) {
	result := make(PolicyHandlerCurrentPolicyIndexedByResourceId)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByResourceIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentPolicySlice.IndexByResourceId")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentPolicyIndexedByResourceId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentPolicyGroupedByResourceId map[string][]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) GroupByResourceId() PolicyHandlerCurrentPolicyGroupedByResourceId {
	result := make(PolicyHandlerCurrentPolicyGroupedByResourceId)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByResourceIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentPolicyGroupedByResourceId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PolicyHandlerCurrentPolicyIndexByResourceVersionKey(value *CurrentPolicyView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ResourceVersion, true
}

type PolicyHandlerCurrentPolicyIndexedByResourceVersion map[string]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) IndexByResourceVersion() (PolicyHandlerCurrentPolicyIndexedByResourceVersion, error) {
	result := make(PolicyHandlerCurrentPolicyIndexedByResourceVersion)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByResourceVersionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentPolicySlice.IndexByResourceVersion")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentPolicyIndexedByResourceVersion) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentPolicyGroupedByResourceVersion map[string][]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) GroupByResourceVersion() PolicyHandlerCurrentPolicyGroupedByResourceVersion {
	result := make(PolicyHandlerCurrentPolicyGroupedByResourceVersion)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByResourceVersionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentPolicyGroupedByResourceVersion) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PolicyHandlerCurrentPolicyIndexByRevisionKey(value *CurrentPolicyView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.Revision == nil {
		return zero, false
	}
	return *value.Revision, true
}

type PolicyHandlerCurrentPolicyIndexedByRevision map[int]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) IndexByRevision() (PolicyHandlerCurrentPolicyIndexedByRevision, error) {
	result := make(PolicyHandlerCurrentPolicyIndexedByRevision)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByRevisionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentPolicySlice.IndexByRevision")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentPolicyIndexedByRevision) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentPolicyGroupedByRevision map[int][]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) GroupByRevision() PolicyHandlerCurrentPolicyGroupedByRevision {
	result := make(PolicyHandlerCurrentPolicyGroupedByRevision)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByRevisionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentPolicyGroupedByRevision) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentPolicyKey struct {
	TenantId        string
	ResourceKind    string
	ResourceId      string
	ResourceVersion string
}

func PolicyHandlerCurrentPolicyIndexByKeyKey(value *CurrentPolicyView) (PolicyHandlerCurrentPolicyKey, bool) {
	var zero PolicyHandlerCurrentPolicyKey
	if value == nil {
		return zero, false
	}
	return PolicyHandlerCurrentPolicyKey{TenantId: value.TenantId, ResourceKind: value.ResourceKind, ResourceId: value.ResourceId, ResourceVersion: value.ResourceVersion}, true
}

type PolicyHandlerCurrentPolicyIndexedByKey map[PolicyHandlerCurrentPolicyKey]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) IndexByKey() (PolicyHandlerCurrentPolicyIndexedByKey, error) {
	result := make(PolicyHandlerCurrentPolicyIndexedByKey)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentPolicySlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentPolicyIndexedByKey) Has(key PolicyHandlerCurrentPolicyKey) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentPolicyGroupedByKey map[PolicyHandlerCurrentPolicyKey][]*CurrentPolicyView

func (rows PolicyHandlerCurrentPolicySlice) GroupByKey() PolicyHandlerCurrentPolicyGroupedByKey {
	result := make(PolicyHandlerCurrentPolicyGroupedByKey)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentPolicyIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentPolicyGroupedByKey) Has(key PolicyHandlerCurrentPolicyKey) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistorySlice []*CurrentHistoryView

func PolicyHandlerCurrentHistoryIndexByTenantIdKey(value *CurrentHistoryView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.TenantId, true
}

type PolicyHandlerCurrentHistoryIndexedByTenantId map[string]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) IndexByTenantId() (PolicyHandlerCurrentHistoryIndexedByTenantId, error) {
	result := make(PolicyHandlerCurrentHistoryIndexedByTenantId)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByTenantIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentHistorySlice.IndexByTenantId")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentHistoryIndexedByTenantId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistoryGroupedByTenantId map[string][]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) GroupByTenantId() PolicyHandlerCurrentHistoryGroupedByTenantId {
	result := make(PolicyHandlerCurrentHistoryGroupedByTenantId)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByTenantIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentHistoryGroupedByTenantId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PolicyHandlerCurrentHistoryIndexByResourceKindKey(value *CurrentHistoryView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ResourceKind, true
}

type PolicyHandlerCurrentHistoryIndexedByResourceKind map[string]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) IndexByResourceKind() (PolicyHandlerCurrentHistoryIndexedByResourceKind, error) {
	result := make(PolicyHandlerCurrentHistoryIndexedByResourceKind)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByResourceKindKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentHistorySlice.IndexByResourceKind")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentHistoryIndexedByResourceKind) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistoryGroupedByResourceKind map[string][]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) GroupByResourceKind() PolicyHandlerCurrentHistoryGroupedByResourceKind {
	result := make(PolicyHandlerCurrentHistoryGroupedByResourceKind)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByResourceKindKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentHistoryGroupedByResourceKind) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PolicyHandlerCurrentHistoryIndexByResourceIdKey(value *CurrentHistoryView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ResourceId, true
}

type PolicyHandlerCurrentHistoryIndexedByResourceId map[string]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) IndexByResourceId() (PolicyHandlerCurrentHistoryIndexedByResourceId, error) {
	result := make(PolicyHandlerCurrentHistoryIndexedByResourceId)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByResourceIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentHistorySlice.IndexByResourceId")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentHistoryIndexedByResourceId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistoryGroupedByResourceId map[string][]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) GroupByResourceId() PolicyHandlerCurrentHistoryGroupedByResourceId {
	result := make(PolicyHandlerCurrentHistoryGroupedByResourceId)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByResourceIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentHistoryGroupedByResourceId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PolicyHandlerCurrentHistoryIndexByResourceVersionKey(value *CurrentHistoryView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ResourceVersion, true
}

type PolicyHandlerCurrentHistoryIndexedByResourceVersion map[string]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) IndexByResourceVersion() (PolicyHandlerCurrentHistoryIndexedByResourceVersion, error) {
	result := make(PolicyHandlerCurrentHistoryIndexedByResourceVersion)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByResourceVersionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentHistorySlice.IndexByResourceVersion")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentHistoryIndexedByResourceVersion) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistoryGroupedByResourceVersion map[string][]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) GroupByResourceVersion() PolicyHandlerCurrentHistoryGroupedByResourceVersion {
	result := make(PolicyHandlerCurrentHistoryGroupedByResourceVersion)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByResourceVersionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentHistoryGroupedByResourceVersion) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PolicyHandlerCurrentHistoryIndexByActorIdKey(value *CurrentHistoryView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ActorId, true
}

type PolicyHandlerCurrentHistoryIndexedByActorId map[string]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) IndexByActorId() (PolicyHandlerCurrentHistoryIndexedByActorId, error) {
	result := make(PolicyHandlerCurrentHistoryIndexedByActorId)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByActorIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentHistorySlice.IndexByActorId")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentHistoryIndexedByActorId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistoryGroupedByActorId map[string][]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) GroupByActorId() PolicyHandlerCurrentHistoryGroupedByActorId {
	result := make(PolicyHandlerCurrentHistoryGroupedByActorId)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByActorIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentHistoryGroupedByActorId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func PolicyHandlerCurrentHistoryIndexByRevisionKey(value *CurrentHistoryView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.Revision == nil {
		return zero, false
	}
	return *value.Revision, true
}

type PolicyHandlerCurrentHistoryIndexedByRevision map[int]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) IndexByRevision() (PolicyHandlerCurrentHistoryIndexedByRevision, error) {
	result := make(PolicyHandlerCurrentHistoryIndexedByRevision)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByRevisionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentHistorySlice.IndexByRevision")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentHistoryIndexedByRevision) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistoryGroupedByRevision map[int][]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) GroupByRevision() PolicyHandlerCurrentHistoryGroupedByRevision {
	result := make(PolicyHandlerCurrentHistoryGroupedByRevision)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByRevisionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentHistoryGroupedByRevision) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func PolicyHandlerCurrentHistoryIndexByOccurredAtKey(value *CurrentHistoryView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.OccurredAt == nil {
		return zero, false
	}
	return *value.OccurredAt, true
}

type PolicyHandlerCurrentHistoryIndexedByOccurredAt map[time.Time]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) IndexByOccurredAt() (PolicyHandlerCurrentHistoryIndexedByOccurredAt, error) {
	result := make(PolicyHandlerCurrentHistoryIndexedByOccurredAt)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByOccurredAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentHistorySlice.IndexByOccurredAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentHistoryIndexedByOccurredAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistoryGroupedByOccurredAt map[time.Time][]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) GroupByOccurredAt() PolicyHandlerCurrentHistoryGroupedByOccurredAt {
	result := make(PolicyHandlerCurrentHistoryGroupedByOccurredAt)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByOccurredAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentHistoryGroupedByOccurredAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistoryKey struct {
	TenantId        string
	ResourceKind    string
	ResourceId      string
	ResourceVersion string
	Revision        int
}

func PolicyHandlerCurrentHistoryIndexByKeyKey(value *CurrentHistoryView) (PolicyHandlerCurrentHistoryKey, bool) {
	var zero PolicyHandlerCurrentHistoryKey
	if value == nil {
		return zero, false
	}
	if value.Revision == nil {
		return zero, false
	}
	return PolicyHandlerCurrentHistoryKey{TenantId: value.TenantId, ResourceKind: value.ResourceKind, ResourceId: value.ResourceId, ResourceVersion: value.ResourceVersion, Revision: *value.Revision}, true
}

type PolicyHandlerCurrentHistoryIndexedByKey map[PolicyHandlerCurrentHistoryKey]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) IndexByKey() (PolicyHandlerCurrentHistoryIndexedByKey, error) {
	result := make(PolicyHandlerCurrentHistoryIndexedByKey)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentHistorySlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentHistoryIndexedByKey) Has(key PolicyHandlerCurrentHistoryKey) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistoryGroupedByKey map[PolicyHandlerCurrentHistoryKey][]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) GroupByKey() PolicyHandlerCurrentHistoryGroupedByKey {
	result := make(PolicyHandlerCurrentHistoryGroupedByKey)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentHistoryGroupedByKey) Has(key PolicyHandlerCurrentHistoryKey) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistoryTenantIdAndResourceKindAndResourceIdAndResourceVersionKey struct {
	TenantId        string
	ResourceKind    string
	ResourceId      string
	ResourceVersion string
}

func PolicyHandlerCurrentHistoryIndexByTenantIdAndResourceKindAndResourceIdAndResourceVersionKey(value *CurrentHistoryView) (PolicyHandlerCurrentHistoryTenantIdAndResourceKindAndResourceIdAndResourceVersionKey, bool) {
	var zero PolicyHandlerCurrentHistoryTenantIdAndResourceKindAndResourceIdAndResourceVersionKey
	if value == nil {
		return zero, false
	}
	return PolicyHandlerCurrentHistoryTenantIdAndResourceKindAndResourceIdAndResourceVersionKey{TenantId: value.TenantId, ResourceKind: value.ResourceKind, ResourceId: value.ResourceId, ResourceVersion: value.ResourceVersion}, true
}

type PolicyHandlerCurrentHistoryIndexedByTenantIdAndResourceKindAndResourceIdAndResourceVersion map[PolicyHandlerCurrentHistoryTenantIdAndResourceKindAndResourceIdAndResourceVersionKey]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) IndexByTenantIdAndResourceKindAndResourceIdAndResourceVersion() (PolicyHandlerCurrentHistoryIndexedByTenantIdAndResourceKindAndResourceIdAndResourceVersion, error) {
	result := make(PolicyHandlerCurrentHistoryIndexedByTenantIdAndResourceKindAndResourceIdAndResourceVersion)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByTenantIdAndResourceKindAndResourceIdAndResourceVersionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index PolicyHandlerCurrentHistorySlice.IndexByTenantIdAndResourceKindAndResourceIdAndResourceVersion")
		}
		result[key] = row
	}
	return result, nil
}
func (index PolicyHandlerCurrentHistoryIndexedByTenantIdAndResourceKindAndResourceIdAndResourceVersion) Has(key PolicyHandlerCurrentHistoryTenantIdAndResourceKindAndResourceIdAndResourceVersionKey) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerCurrentHistoryGroupedByTenantIdAndResourceKindAndResourceIdAndResourceVersion map[PolicyHandlerCurrentHistoryTenantIdAndResourceKindAndResourceIdAndResourceVersionKey][]*CurrentHistoryView

func (rows PolicyHandlerCurrentHistorySlice) GroupByTenantIdAndResourceKindAndResourceIdAndResourceVersion() PolicyHandlerCurrentHistoryGroupedByTenantIdAndResourceKindAndResourceIdAndResourceVersion {
	result := make(PolicyHandlerCurrentHistoryGroupedByTenantIdAndResourceKindAndResourceIdAndResourceVersion)
	for _, row := range rows {
		key, ok := PolicyHandlerCurrentHistoryIndexByTenantIdAndResourceKindAndResourceIdAndResourceVersionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index PolicyHandlerCurrentHistoryGroupedByTenantIdAndResourceKindAndResourceIdAndResourceVersion) Has(key PolicyHandlerCurrentHistoryTenantIdAndResourceKindAndResourceIdAndResourceVersionKey) bool {
	_, ok := index[key]
	return ok
}

type PolicyHandlerReadIndexes struct {
	CurrentPolicy                                                                 PolicyHandlerCurrentPolicySlice
	CurrentPolicyByKey                                                            PolicyHandlerCurrentPolicyIndexedByKey
	CurrentPolicyGroupedByKey                                                     PolicyHandlerCurrentPolicyGroupedByKey
	CurrentHistory                                                                PolicyHandlerCurrentHistorySlice
	CurrentHistoryByKey                                                           PolicyHandlerCurrentHistoryIndexedByKey
	CurrentHistoryGroupedByTenantIdAndResourceKindAndResourceIdAndResourceVersion PolicyHandlerCurrentHistoryGroupedByTenantIdAndResourceKindAndResourceIdAndResourceVersion
}

func BuildPolicyHandlerReadIndexes(ctx context.Context, input *Input) (*PolicyHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &PolicyHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentPolicy")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentPolicy")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentPolicy")
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
			if _, err := (xshape.Collection[CurrentPolicyView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentPolicyView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentPolicyView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("TenantId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPolicy.TenantId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ResourceKind") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPolicy.ResourceKind")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ResourceId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPolicy.ResourceId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ResourceVersion") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPolicy.ResourceVersion")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Revision") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentPolicy.Revision")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentPolicy = PolicyHandlerCurrentPolicySlice(cloned.([]*CurrentPolicyView))
		result.CurrentPolicyByKey, err = result.CurrentPolicy.IndexByKey()
		if err != nil {
			return nil, err
		}
		result.CurrentPolicyGroupedByKey = result.CurrentPolicy.GroupByKey()
	}
	{
		projection, err := metadata.Projection("CurrentHistory")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentHistory")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentHistory")
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
			if _, err := (xshape.Collection[CurrentHistoryView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentHistoryView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentHistoryView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("TenantId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentHistory.TenantId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ResourceKind") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentHistory.ResourceKind")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ResourceId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentHistory.ResourceId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ResourceVersion") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentHistory.ResourceVersion")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ActorId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentHistory.ActorId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("PoliciesJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentHistory.PoliciesJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Revision") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentHistory.Revision")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("OccurredAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentHistory.OccurredAt")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentHistory = PolicyHandlerCurrentHistorySlice(cloned.([]*CurrentHistoryView))
		result.CurrentHistoryByKey, err = result.CurrentHistory.IndexByKey()
		if err != nil {
			return nil, err
		}
		result.CurrentHistoryGroupedByTenantIdAndResourceKindAndResourceIdAndResourceVersion = result.CurrentHistory.GroupByTenantIdAndResourceKindAndResourceIdAndResourceVersion()
	}
	return result, nil
}
func (input *Input) PrepareReadIndexes(ctx context.Context) error {
	if input == nil {
		return fmt.Errorf("read indexes require input")
	}
	input._policyHandlerReadIndexes = nil
	indexes, err := BuildPolicyHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._policyHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*PolicyHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._policyHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._policyHandlerReadIndexes, nil
}
