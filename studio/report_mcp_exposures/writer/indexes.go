package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
)

type ExposureHandlerCurrentExposureSlice []*CurrentExposureView

func ExposureHandlerCurrentExposureIndexByReportIdKey(value *CurrentExposureView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	return *value.ReportId, true
}

type ExposureHandlerCurrentExposureIndexedByReportId map[string]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByReportId() (ExposureHandlerCurrentExposureIndexedByReportId, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByReportId)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByReportId map[string][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByReportId() ExposureHandlerCurrentExposureGroupedByReportId {
	result := make(ExposureHandlerCurrentExposureGroupedByReportId)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByVersionNoKey(value *CurrentExposureView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	return *value.VersionNo, true
}

type ExposureHandlerCurrentExposureIndexedByVersionNo map[int]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByVersionNo() (ExposureHandlerCurrentExposureIndexedByVersionNo, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByVersionNo)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByVersionNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByVersionNo map[int][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByVersionNo() ExposureHandlerCurrentExposureGroupedByVersionNo {
	result := make(ExposureHandlerCurrentExposureGroupedByVersionNo)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByExposureIdKey(value *CurrentExposureView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ExposureId == nil {
		return zero, false
	}
	return *value.ExposureId, true
}

type ExposureHandlerCurrentExposureIndexedByExposureId map[string]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByExposureId() (ExposureHandlerCurrentExposureIndexedByExposureId, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByExposureId)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByExposureIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByExposureId")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByExposureId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByExposureId map[string][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByExposureId() ExposureHandlerCurrentExposureGroupedByExposureId {
	result := make(ExposureHandlerCurrentExposureGroupedByExposureId)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByExposureIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByExposureId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByRouteIdKey(value *CurrentExposureView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.RouteId == nil {
		return zero, false
	}
	return *value.RouteId, true
}

type ExposureHandlerCurrentExposureIndexedByRouteId map[string]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByRouteId() (ExposureHandlerCurrentExposureIndexedByRouteId, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByRouteId)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByRouteIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByRouteId")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByRouteId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByRouteId map[string][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByRouteId() ExposureHandlerCurrentExposureGroupedByRouteId {
	result := make(ExposureHandlerCurrentExposureGroupedByRouteId)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByRouteIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByRouteId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByRouteMethodKey(value *CurrentExposureView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.RouteMethod == nil {
		return zero, false
	}
	return *value.RouteMethod, true
}

type ExposureHandlerCurrentExposureIndexedByRouteMethod map[string]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByRouteMethod() (ExposureHandlerCurrentExposureIndexedByRouteMethod, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByRouteMethod)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByRouteMethodKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByRouteMethod")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByRouteMethod) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByRouteMethod map[string][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByRouteMethod() ExposureHandlerCurrentExposureGroupedByRouteMethod {
	result := make(ExposureHandlerCurrentExposureGroupedByRouteMethod)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByRouteMethodKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByRouteMethod) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByRoutePathKey(value *CurrentExposureView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.RoutePath == nil {
		return zero, false
	}
	return *value.RoutePath, true
}

type ExposureHandlerCurrentExposureIndexedByRoutePath map[string]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByRoutePath() (ExposureHandlerCurrentExposureIndexedByRoutePath, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByRoutePath)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByRoutePathKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByRoutePath")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByRoutePath) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByRoutePath map[string][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByRoutePath() ExposureHandlerCurrentExposureGroupedByRoutePath {
	result := make(ExposureHandlerCurrentExposureGroupedByRoutePath)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByRoutePathKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByRoutePath) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByKindKey(value *CurrentExposureView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Kind == nil {
		return zero, false
	}
	return *value.Kind, true
}

type ExposureHandlerCurrentExposureIndexedByKind map[string]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByKind() (ExposureHandlerCurrentExposureIndexedByKind, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByKind)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByKindKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByKind")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByKind) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByKind map[string][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByKind() ExposureHandlerCurrentExposureGroupedByKind {
	result := make(ExposureHandlerCurrentExposureGroupedByKind)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByKindKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByKind) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByNameKey(value *CurrentExposureView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Name == nil {
		return zero, false
	}
	return *value.Name, true
}

type ExposureHandlerCurrentExposureIndexedByName map[string]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByName() (ExposureHandlerCurrentExposureIndexedByName, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByName)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByNameKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByName")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByName map[string][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByName() ExposureHandlerCurrentExposureGroupedByName {
	result := make(ExposureHandlerCurrentExposureGroupedByName)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByNameKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByName) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByDescriptionKey(value *CurrentExposureView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Description == nil {
		return zero, false
	}
	return *value.Description, true
}

type ExposureHandlerCurrentExposureIndexedByDescription map[string]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByDescription() (ExposureHandlerCurrentExposureIndexedByDescription, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByDescription)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByDescriptionKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByDescription")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByDescription) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByDescription map[string][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByDescription() ExposureHandlerCurrentExposureGroupedByDescription {
	result := make(ExposureHandlerCurrentExposureGroupedByDescription)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByDescriptionKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByDescription) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByDescriptionPathKey(value *CurrentExposureView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.DescriptionPath == nil {
		return zero, false
	}
	return *value.DescriptionPath, true
}

type ExposureHandlerCurrentExposureIndexedByDescriptionPath map[string]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByDescriptionPath() (ExposureHandlerCurrentExposureIndexedByDescriptionPath, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByDescriptionPath)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByDescriptionPathKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByDescriptionPath")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByDescriptionPath) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByDescriptionPath map[string][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByDescriptionPath() ExposureHandlerCurrentExposureGroupedByDescriptionPath {
	result := make(ExposureHandlerCurrentExposureGroupedByDescriptionPath)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByDescriptionPathKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByDescriptionPath) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByMimeTypeKey(value *CurrentExposureView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.MimeType == nil {
		return zero, false
	}
	return *value.MimeType, true
}

type ExposureHandlerCurrentExposureIndexedByMimeType map[string]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByMimeType() (ExposureHandlerCurrentExposureIndexedByMimeType, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByMimeType)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByMimeTypeKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByMimeType")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByMimeType) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByMimeType map[string][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByMimeType() ExposureHandlerCurrentExposureGroupedByMimeType {
	result := make(ExposureHandlerCurrentExposureGroupedByMimeType)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByMimeTypeKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByMimeType) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByEnabledKey(value *CurrentExposureView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.Enabled == nil {
		return zero, false
	}
	return *value.Enabled, true
}

type ExposureHandlerCurrentExposureIndexedByEnabled map[int]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByEnabled() (ExposureHandlerCurrentExposureIndexedByEnabled, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByEnabled)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByEnabledKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByEnabled")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByEnabled) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByEnabled map[int][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByEnabled() ExposureHandlerCurrentExposureGroupedByEnabled {
	result := make(ExposureHandlerCurrentExposureGroupedByEnabled)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByEnabledKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByEnabled) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ExposureHandlerCurrentExposureIndexByOrdinalKey(value *CurrentExposureView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.Ordinal == nil {
		return zero, false
	}
	return *value.Ordinal, true
}

type ExposureHandlerCurrentExposureIndexedByOrdinal map[int]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByOrdinal() (ExposureHandlerCurrentExposureIndexedByOrdinal, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByOrdinal)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByOrdinalKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByOrdinal")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByOrdinal) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByOrdinal map[int][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByOrdinal() ExposureHandlerCurrentExposureGroupedByOrdinal {
	result := make(ExposureHandlerCurrentExposureGroupedByOrdinal)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByOrdinalKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByOrdinal) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureKey struct {
	ReportId   string
	VersionNo  int
	ExposureId string
}

func ExposureHandlerCurrentExposureIndexByKeyKey(value *CurrentExposureView) (ExposureHandlerCurrentExposureKey, bool) {
	var zero ExposureHandlerCurrentExposureKey
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	if value.ExposureId == nil {
		return zero, false
	}
	return ExposureHandlerCurrentExposureKey{ReportId: *value.ReportId, VersionNo: *value.VersionNo, ExposureId: *value.ExposureId}, true
}

type ExposureHandlerCurrentExposureIndexedByKey map[ExposureHandlerCurrentExposureKey]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) IndexByKey() (ExposureHandlerCurrentExposureIndexedByKey, error) {
	result := make(ExposureHandlerCurrentExposureIndexedByKey)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ExposureHandlerCurrentExposureSlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index ExposureHandlerCurrentExposureIndexedByKey) Has(key ExposureHandlerCurrentExposureKey) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerCurrentExposureGroupedByKey map[ExposureHandlerCurrentExposureKey][]*CurrentExposureView

func (rows ExposureHandlerCurrentExposureSlice) GroupByKey() ExposureHandlerCurrentExposureGroupedByKey {
	result := make(ExposureHandlerCurrentExposureGroupedByKey)
	for _, row := range rows {
		key, ok := ExposureHandlerCurrentExposureIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ExposureHandlerCurrentExposureGroupedByKey) Has(key ExposureHandlerCurrentExposureKey) bool {
	_, ok := index[key]
	return ok
}

type ExposureHandlerReadIndexes struct {
	CurrentExposure      ExposureHandlerCurrentExposureSlice
	CurrentExposureByKey ExposureHandlerCurrentExposureIndexedByKey
}

func BuildExposureHandlerReadIndexes(ctx context.Context, input *Input) (*ExposureHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &ExposureHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentExposure")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentExposure")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentExposure")
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
			if _, err := (xshape.Collection[CurrentExposureView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentExposureView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentExposureView]{}).Pointers(value)
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
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("VersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.VersionNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ExposureId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.ExposureId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RouteId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.RouteId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RouteMethod") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.RouteMethod")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RoutePath") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.RoutePath")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Kind") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.Kind")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Name") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.Name")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Description") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.Description")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DescriptionPath") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.DescriptionPath")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("MimeType") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.MimeType")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Enabled") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.Enabled")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Ordinal") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentExposure.Ordinal")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentExposure = ExposureHandlerCurrentExposureSlice(cloned.([]*CurrentExposureView))
		result.CurrentExposureByKey, err = result.CurrentExposure.IndexByKey()
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
	input._exposureHandlerReadIndexes = nil
	indexes, err := BuildExposureHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._exposureHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*ExposureHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._exposureHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._exposureHandlerReadIndexes, nil
}
