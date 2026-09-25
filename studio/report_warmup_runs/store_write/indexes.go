package store_write

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
)

type WarmupRunHandlerCurrentWarmupRunSlice []*CurrentWarmupRunView

func WarmupRunHandlerCurrentWarmupRunIndexByRunIdKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.RunId, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByRunId map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByRunId() (WarmupRunHandlerCurrentWarmupRunIndexedByRunId, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByRunId)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByRunIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByRunId")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByRunId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByRunId map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByRunId() WarmupRunHandlerCurrentWarmupRunGroupedByRunId {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByRunId)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByRunIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByRunId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByReportIdKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ReportId, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByReportId map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByReportId() (WarmupRunHandlerCurrentWarmupRunIndexedByReportId, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByReportId)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByReportId map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByReportId() WarmupRunHandlerCurrentWarmupRunGroupedByReportId {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByReportId)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByVersionNoKey(value *CurrentWarmupRunView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	return value.VersionNo, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByVersionNo map[int]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByVersionNo() (WarmupRunHandlerCurrentWarmupRunIndexedByVersionNo, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByVersionNo)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByVersionNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByVersionNo map[int][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByVersionNo() WarmupRunHandlerCurrentWarmupRunGroupedByVersionNo {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByVersionNo)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexBySourceRevisionKey(value *CurrentWarmupRunView) (int64, bool) {
	var zero int64
	if value == nil {
		return zero, false
	}
	return value.SourceRevision, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedBySourceRevision map[int64]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexBySourceRevision() (WarmupRunHandlerCurrentWarmupRunIndexedBySourceRevision, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedBySourceRevision)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexBySourceRevisionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexBySourceRevision")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedBySourceRevision) Has(key int64) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedBySourceRevision map[int64][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupBySourceRevision() WarmupRunHandlerCurrentWarmupRunGroupedBySourceRevision {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedBySourceRevision)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexBySourceRevisionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedBySourceRevision) Has(key int64) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexBySpecHashKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.SpecHash, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedBySpecHash map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexBySpecHash() (WarmupRunHandlerCurrentWarmupRunIndexedBySpecHash, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedBySpecHash)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexBySpecHashKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexBySpecHash")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedBySpecHash) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedBySpecHash map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupBySpecHash() WarmupRunHandlerCurrentWarmupRunGroupedBySpecHash {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedBySpecHash)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexBySpecHashKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedBySpecHash) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByPlanKeyKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.PlanKey, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByPlanKey map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByPlanKey() (WarmupRunHandlerCurrentWarmupRunIndexedByPlanKey, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByPlanKey)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByPlanKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByPlanKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByPlanKey) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByPlanKey map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByPlanKey() WarmupRunHandlerCurrentWarmupRunGroupedByPlanKey {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByPlanKey)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByPlanKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByPlanKey) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByActiveKeyKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ActiveKey == nil {
		return zero, false
	}
	return *value.ActiveKey, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByActiveKey map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByActiveKey() (WarmupRunHandlerCurrentWarmupRunIndexedByActiveKey, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByActiveKey)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByActiveKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByActiveKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByActiveKey) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByActiveKey map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByActiveKey() WarmupRunHandlerCurrentWarmupRunGroupedByActiveKey {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByActiveKey)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByActiveKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByActiveKey) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByStatusKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Status, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByStatus map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByStatus() (WarmupRunHandlerCurrentWarmupRunIndexedByStatus, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByStatus)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByStatusKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByStatus")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByStatus map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByStatus() WarmupRunHandlerCurrentWarmupRunGroupedByStatus {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByStatus)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByStatusKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByStatus) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByRequestedByKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.RequestedBy, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByRequestedBy map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByRequestedBy() (WarmupRunHandlerCurrentWarmupRunIndexedByRequestedBy, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByRequestedBy)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByRequestedByKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByRequestedBy")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByRequestedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByRequestedBy map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByRequestedBy() WarmupRunHandlerCurrentWarmupRunGroupedByRequestedBy {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByRequestedBy)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByRequestedByKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByRequestedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByCacheNameKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.CacheName == nil {
		return zero, false
	}
	return *value.CacheName, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByCacheName map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByCacheName() (WarmupRunHandlerCurrentWarmupRunIndexedByCacheName, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByCacheName)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCacheNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByCacheName")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByCacheName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByCacheName map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByCacheName() WarmupRunHandlerCurrentWarmupRunGroupedByCacheName {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByCacheName)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCacheNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByCacheName) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByCacheProviderKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.CacheProvider == nil {
		return zero, false
	}
	return *value.CacheProvider, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByCacheProvider map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByCacheProvider() (WarmupRunHandlerCurrentWarmupRunIndexedByCacheProvider, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByCacheProvider)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCacheProviderKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByCacheProvider")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByCacheProvider) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByCacheProvider map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByCacheProvider() WarmupRunHandlerCurrentWarmupRunGroupedByCacheProvider {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByCacheProvider)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCacheProviderKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByCacheProvider) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByConnectorNameKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ConnectorName == nil {
		return zero, false
	}
	return *value.ConnectorName, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByConnectorName map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByConnectorName() (WarmupRunHandlerCurrentWarmupRunIndexedByConnectorName, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByConnectorName)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByConnectorNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByConnectorName")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByConnectorName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByConnectorName map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByConnectorName() WarmupRunHandlerCurrentWarmupRunGroupedByConnectorName {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByConnectorName)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByConnectorNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByConnectorName) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByIndexColumnKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.IndexColumn == nil {
		return zero, false
	}
	return *value.IndexColumn, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByIndexColumn map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByIndexColumn() (WarmupRunHandlerCurrentWarmupRunIndexedByIndexColumn, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByIndexColumn)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByIndexColumnKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByIndexColumn")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByIndexColumn) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByIndexColumn map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByIndexColumn() WarmupRunHandlerCurrentWarmupRunGroupedByIndexColumn {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByIndexColumn)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByIndexColumnKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByIndexColumn) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByPlannedCasesKey(value *CurrentWarmupRunView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	return value.PlannedCases, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByPlannedCases map[int]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByPlannedCases() (WarmupRunHandlerCurrentWarmupRunIndexedByPlannedCases, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByPlannedCases)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByPlannedCasesKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByPlannedCases")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByPlannedCases) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByPlannedCases map[int][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByPlannedCases() WarmupRunHandlerCurrentWarmupRunGroupedByPlannedCases {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByPlannedCases)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByPlannedCasesKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByPlannedCases) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByCompletedCasesKey(value *CurrentWarmupRunView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	return value.CompletedCases, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByCompletedCases map[int]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByCompletedCases() (WarmupRunHandlerCurrentWarmupRunIndexedByCompletedCases, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByCompletedCases)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCompletedCasesKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByCompletedCases")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByCompletedCases) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByCompletedCases map[int][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByCompletedCases() WarmupRunHandlerCurrentWarmupRunGroupedByCompletedCases {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByCompletedCases)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCompletedCasesKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByCompletedCases) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByMaxCasesKey(value *CurrentWarmupRunView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.MaxCases == nil {
		return zero, false
	}
	return *value.MaxCases, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByMaxCases map[int]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByMaxCases() (WarmupRunHandlerCurrentWarmupRunIndexedByMaxCases, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByMaxCases)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByMaxCasesKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByMaxCases")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByMaxCases) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByMaxCases map[int][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByMaxCases() WarmupRunHandlerCurrentWarmupRunGroupedByMaxCases {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByMaxCases)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByMaxCasesKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByMaxCases) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByRowLimitKey(value *CurrentWarmupRunView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.RowLimit == nil {
		return zero, false
	}
	return *value.RowLimit, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByRowLimit map[int]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByRowLimit() (WarmupRunHandlerCurrentWarmupRunIndexedByRowLimit, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByRowLimit)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByRowLimitKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByRowLimit")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByRowLimit) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByRowLimit map[int][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByRowLimit() WarmupRunHandlerCurrentWarmupRunGroupedByRowLimit {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByRowLimit)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByRowLimitKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByRowLimit) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByEntriesKey(value *CurrentWarmupRunView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	return value.Entries, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByEntries map[int]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByEntries() (WarmupRunHandlerCurrentWarmupRunIndexedByEntries, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByEntries)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByEntriesKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByEntries")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByEntries) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByEntries map[int][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByEntries() WarmupRunHandlerCurrentWarmupRunGroupedByEntries {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByEntries)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByEntriesKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByEntries) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByDurationNsKey(value *CurrentWarmupRunView) (int64, bool) {
	var zero int64
	if value == nil {
		return zero, false
	}
	return value.DurationNs, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByDurationNs map[int64]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByDurationNs() (WarmupRunHandlerCurrentWarmupRunIndexedByDurationNs, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByDurationNs)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByDurationNsKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByDurationNs")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByDurationNs) Has(key int64) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByDurationNs map[int64][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByDurationNs() WarmupRunHandlerCurrentWarmupRunGroupedByDurationNs {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByDurationNs)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByDurationNsKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByDurationNs) Has(key int64) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByRequestedAtKey(value *CurrentWarmupRunView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	return value.RequestedAt, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByRequestedAt map[time.Time]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByRequestedAt() (WarmupRunHandlerCurrentWarmupRunIndexedByRequestedAt, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByRequestedAt)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByRequestedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByRequestedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByRequestedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByRequestedAt map[time.Time][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByRequestedAt() WarmupRunHandlerCurrentWarmupRunGroupedByRequestedAt {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByRequestedAt)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByRequestedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByRequestedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByCreatedAtKey(value *CurrentWarmupRunView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.CreatedAt == nil {
		return zero, false
	}
	return *value.CreatedAt, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByCreatedAt map[time.Time]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByCreatedAt() (WarmupRunHandlerCurrentWarmupRunIndexedByCreatedAt, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByCreatedAt)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByCreatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByCreatedAt map[time.Time][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByCreatedAt() WarmupRunHandlerCurrentWarmupRunGroupedByCreatedAt {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByCreatedAt)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByCreatedByKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.CreatedBy == nil {
		return zero, false
	}
	return *value.CreatedBy, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByCreatedBy map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByCreatedBy() (WarmupRunHandlerCurrentWarmupRunIndexedByCreatedBy, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByCreatedBy)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCreatedByKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByCreatedBy")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByCreatedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByCreatedBy map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByCreatedBy() WarmupRunHandlerCurrentWarmupRunGroupedByCreatedBy {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByCreatedBy)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCreatedByKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByCreatedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByUpdatedAtKey(value *CurrentWarmupRunView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.UpdatedAt == nil {
		return zero, false
	}
	return *value.UpdatedAt, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByUpdatedAt map[time.Time]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByUpdatedAt() (WarmupRunHandlerCurrentWarmupRunIndexedByUpdatedAt, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByUpdatedAt)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByUpdatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByUpdatedAt map[time.Time][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByUpdatedAt() WarmupRunHandlerCurrentWarmupRunGroupedByUpdatedAt {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByUpdatedAt)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByUpdatedByKey(value *CurrentWarmupRunView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.UpdatedBy == nil {
		return zero, false
	}
	return *value.UpdatedBy, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByUpdatedBy map[string]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByUpdatedBy() (WarmupRunHandlerCurrentWarmupRunIndexedByUpdatedBy, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByUpdatedBy)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByUpdatedByKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByUpdatedBy")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByUpdatedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByUpdatedBy map[string][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByUpdatedBy() WarmupRunHandlerCurrentWarmupRunGroupedByUpdatedBy {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByUpdatedBy)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByUpdatedByKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByUpdatedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByStartedAtKey(value *CurrentWarmupRunView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.StartedAt == nil {
		return zero, false
	}
	return *value.StartedAt, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByStartedAt map[time.Time]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByStartedAt() (WarmupRunHandlerCurrentWarmupRunIndexedByStartedAt, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByStartedAt)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByStartedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByStartedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByStartedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByStartedAt map[time.Time][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByStartedAt() WarmupRunHandlerCurrentWarmupRunGroupedByStartedAt {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByStartedAt)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByStartedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByStartedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func WarmupRunHandlerCurrentWarmupRunIndexByCompletedAtKey(value *CurrentWarmupRunView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	if value.CompletedAt == nil {
		return zero, false
	}
	return *value.CompletedAt, true
}

type WarmupRunHandlerCurrentWarmupRunIndexedByCompletedAt map[time.Time]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) IndexByCompletedAt() (WarmupRunHandlerCurrentWarmupRunIndexedByCompletedAt, error) {
	result := make(WarmupRunHandlerCurrentWarmupRunIndexedByCompletedAt)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCompletedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index WarmupRunHandlerCurrentWarmupRunSlice.IndexByCompletedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index WarmupRunHandlerCurrentWarmupRunIndexedByCompletedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerCurrentWarmupRunGroupedByCompletedAt map[time.Time][]*CurrentWarmupRunView

func (rows WarmupRunHandlerCurrentWarmupRunSlice) GroupByCompletedAt() WarmupRunHandlerCurrentWarmupRunGroupedByCompletedAt {
	result := make(WarmupRunHandlerCurrentWarmupRunGroupedByCompletedAt)
	for _, row := range rows {
		key, ok := WarmupRunHandlerCurrentWarmupRunIndexByCompletedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index WarmupRunHandlerCurrentWarmupRunGroupedByCompletedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type WarmupRunHandlerReadIndexes struct {
	CurrentWarmupRun        WarmupRunHandlerCurrentWarmupRunSlice
	CurrentWarmupRunByRunId WarmupRunHandlerCurrentWarmupRunIndexedByRunId
}

func BuildWarmupRunHandlerReadIndexes(ctx context.Context, input *Input) (*WarmupRunHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &WarmupRunHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentWarmupRun")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentWarmupRun")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentWarmupRun")
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
			if _, err := (xshape.Collection[CurrentWarmupRunView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentWarmupRunView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentWarmupRunView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RunId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.RunId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ReportId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("VersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.VersionNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SourceRevision") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.SourceRevision")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SpecHash") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.SpecHash")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("PlanKey") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.PlanKey")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ActiveKey") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.ActiveKey")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Status") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.Status")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RequestedBy") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.RequestedBy")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CacheName") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.CacheName")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CacheProvider") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.CacheProvider")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ConnectorName") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.ConnectorName")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("IndexColumn") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.IndexColumn")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("PlannedCases") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.PlannedCases")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CompletedCases") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.CompletedCases")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("MaxCases") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.MaxCases")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RowLimit") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.RowLimit")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Entries") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.Entries")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DurationNs") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.DurationNs")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DiagnosticsJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.DiagnosticsJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("TargetJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.TargetJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RequestedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.RequestedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CreatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.CreatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CreatedBy") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.CreatedBy")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("UpdatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.UpdatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("UpdatedBy") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.UpdatedBy")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("StartedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.StartedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CompletedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentWarmupRun.CompletedAt")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentWarmupRun = WarmupRunHandlerCurrentWarmupRunSlice(cloned.([]*CurrentWarmupRunView))
		result.CurrentWarmupRunByRunId, err = result.CurrentWarmupRun.IndexByRunId()
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
	input._warmupRunHandlerReadIndexes = nil
	indexes, err := BuildWarmupRunHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._warmupRunHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*WarmupRunHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._warmupRunHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._warmupRunHandlerReadIndexes, nil
}
