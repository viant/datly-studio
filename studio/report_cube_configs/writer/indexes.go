package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
)

type ConfigHandlerCurrentConfigSlice []*CurrentConfigView

func ConfigHandlerCurrentConfigIndexByReportIdKey(value *CurrentConfigView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	return *value.ReportId, true
}

type ConfigHandlerCurrentConfigIndexedByReportId map[string]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) IndexByReportId() (ConfigHandlerCurrentConfigIndexedByReportId, error) {
	result := make(ConfigHandlerCurrentConfigIndexedByReportId)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConfigHandlerCurrentConfigSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConfigHandlerCurrentConfigIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigGroupedByReportId map[string][]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) GroupByReportId() ConfigHandlerCurrentConfigGroupedByReportId {
	result := make(ConfigHandlerCurrentConfigGroupedByReportId)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConfigHandlerCurrentConfigGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ConfigHandlerCurrentConfigIndexByVersionNoKey(value *CurrentConfigView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	return *value.VersionNo, true
}

type ConfigHandlerCurrentConfigIndexedByVersionNo map[int]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) IndexByVersionNo() (ConfigHandlerCurrentConfigIndexedByVersionNo, error) {
	result := make(ConfigHandlerCurrentConfigIndexedByVersionNo)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConfigHandlerCurrentConfigSlice.IndexByVersionNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConfigHandlerCurrentConfigIndexedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigGroupedByVersionNo map[int][]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) GroupByVersionNo() ConfigHandlerCurrentConfigGroupedByVersionNo {
	result := make(ConfigHandlerCurrentConfigGroupedByVersionNo)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConfigHandlerCurrentConfigGroupedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ConfigHandlerCurrentConfigIndexByComposeMaxCubesKey(value *CurrentConfigView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.ComposeMaxCubes == nil {
		return zero, false
	}
	return *value.ComposeMaxCubes, true
}

type ConfigHandlerCurrentConfigIndexedByComposeMaxCubes map[int]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) IndexByComposeMaxCubes() (ConfigHandlerCurrentConfigIndexedByComposeMaxCubes, error) {
	result := make(ConfigHandlerCurrentConfigIndexedByComposeMaxCubes)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByComposeMaxCubesKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConfigHandlerCurrentConfigSlice.IndexByComposeMaxCubes")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConfigHandlerCurrentConfigIndexedByComposeMaxCubes) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigGroupedByComposeMaxCubes map[int][]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) GroupByComposeMaxCubes() ConfigHandlerCurrentConfigGroupedByComposeMaxCubes {
	result := make(ConfigHandlerCurrentConfigGroupedByComposeMaxCubes)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByComposeMaxCubesKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConfigHandlerCurrentConfigGroupedByComposeMaxCubes) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ConfigHandlerCurrentConfigIndexByComposeMaxLimitKey(value *CurrentConfigView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.ComposeMaxLimit == nil {
		return zero, false
	}
	return *value.ComposeMaxLimit, true
}

type ConfigHandlerCurrentConfigIndexedByComposeMaxLimit map[int]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) IndexByComposeMaxLimit() (ConfigHandlerCurrentConfigIndexedByComposeMaxLimit, error) {
	result := make(ConfigHandlerCurrentConfigIndexedByComposeMaxLimit)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByComposeMaxLimitKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConfigHandlerCurrentConfigSlice.IndexByComposeMaxLimit")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConfigHandlerCurrentConfigIndexedByComposeMaxLimit) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigGroupedByComposeMaxLimit map[int][]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) GroupByComposeMaxLimit() ConfigHandlerCurrentConfigGroupedByComposeMaxLimit {
	result := make(ConfigHandlerCurrentConfigGroupedByComposeMaxLimit)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByComposeMaxLimitKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConfigHandlerCurrentConfigGroupedByComposeMaxLimit) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ConfigHandlerCurrentConfigIndexByComposeTimeoutMsKey(value *CurrentConfigView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.ComposeTimeoutMs == nil {
		return zero, false
	}
	return *value.ComposeTimeoutMs, true
}

type ConfigHandlerCurrentConfigIndexedByComposeTimeoutMs map[int]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) IndexByComposeTimeoutMs() (ConfigHandlerCurrentConfigIndexedByComposeTimeoutMs, error) {
	result := make(ConfigHandlerCurrentConfigIndexedByComposeTimeoutMs)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByComposeTimeoutMsKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConfigHandlerCurrentConfigSlice.IndexByComposeTimeoutMs")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConfigHandlerCurrentConfigIndexedByComposeTimeoutMs) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigGroupedByComposeTimeoutMs map[int][]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) GroupByComposeTimeoutMs() ConfigHandlerCurrentConfigGroupedByComposeTimeoutMs {
	result := make(ConfigHandlerCurrentConfigGroupedByComposeTimeoutMs)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByComposeTimeoutMsKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConfigHandlerCurrentConfigGroupedByComposeTimeoutMs) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ConfigHandlerCurrentConfigIndexByCubeEnabledKey(value *CurrentConfigView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.CubeEnabled == nil {
		return zero, false
	}
	return *value.CubeEnabled, true
}

type ConfigHandlerCurrentConfigIndexedByCubeEnabled map[int]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) IndexByCubeEnabled() (ConfigHandlerCurrentConfigIndexedByCubeEnabled, error) {
	result := make(ConfigHandlerCurrentConfigIndexedByCubeEnabled)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByCubeEnabledKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConfigHandlerCurrentConfigSlice.IndexByCubeEnabled")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConfigHandlerCurrentConfigIndexedByCubeEnabled) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigGroupedByCubeEnabled map[int][]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) GroupByCubeEnabled() ConfigHandlerCurrentConfigGroupedByCubeEnabled {
	result := make(ConfigHandlerCurrentConfigGroupedByCubeEnabled)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByCubeEnabledKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConfigHandlerCurrentConfigGroupedByCubeEnabled) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ConfigHandlerCurrentConfigIndexByCubeMcpToolEnabledKey(value *CurrentConfigView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.CubeMcpToolEnabled == nil {
		return zero, false
	}
	return *value.CubeMcpToolEnabled, true
}

type ConfigHandlerCurrentConfigIndexedByCubeMcpToolEnabled map[int]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) IndexByCubeMcpToolEnabled() (ConfigHandlerCurrentConfigIndexedByCubeMcpToolEnabled, error) {
	result := make(ConfigHandlerCurrentConfigIndexedByCubeMcpToolEnabled)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByCubeMcpToolEnabledKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConfigHandlerCurrentConfigSlice.IndexByCubeMcpToolEnabled")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConfigHandlerCurrentConfigIndexedByCubeMcpToolEnabled) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigGroupedByCubeMcpToolEnabled map[int][]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) GroupByCubeMcpToolEnabled() ConfigHandlerCurrentConfigGroupedByCubeMcpToolEnabled {
	result := make(ConfigHandlerCurrentConfigGroupedByCubeMcpToolEnabled)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByCubeMcpToolEnabledKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConfigHandlerCurrentConfigGroupedByCubeMcpToolEnabled) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ConfigHandlerCurrentConfigIndexByLinkedInputTypeKey(value *CurrentConfigView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.LinkedInputType == nil {
		return zero, false
	}
	return *value.LinkedInputType, true
}

type ConfigHandlerCurrentConfigIndexedByLinkedInputType map[string]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) IndexByLinkedInputType() (ConfigHandlerCurrentConfigIndexedByLinkedInputType, error) {
	result := make(ConfigHandlerCurrentConfigIndexedByLinkedInputType)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByLinkedInputTypeKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConfigHandlerCurrentConfigSlice.IndexByLinkedInputType")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConfigHandlerCurrentConfigIndexedByLinkedInputType) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigGroupedByLinkedInputType map[string][]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) GroupByLinkedInputType() ConfigHandlerCurrentConfigGroupedByLinkedInputType {
	result := make(ConfigHandlerCurrentConfigGroupedByLinkedInputType)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByLinkedInputTypeKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConfigHandlerCurrentConfigGroupedByLinkedInputType) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ConfigHandlerCurrentConfigIndexByComposeEnabledKey(value *CurrentConfigView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.ComposeEnabled == nil {
		return zero, false
	}
	return *value.ComposeEnabled, true
}

type ConfigHandlerCurrentConfigIndexedByComposeEnabled map[int]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) IndexByComposeEnabled() (ConfigHandlerCurrentConfigIndexedByComposeEnabled, error) {
	result := make(ConfigHandlerCurrentConfigIndexedByComposeEnabled)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByComposeEnabledKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConfigHandlerCurrentConfigSlice.IndexByComposeEnabled")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConfigHandlerCurrentConfigIndexedByComposeEnabled) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigGroupedByComposeEnabled map[int][]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) GroupByComposeEnabled() ConfigHandlerCurrentConfigGroupedByComposeEnabled {
	result := make(ConfigHandlerCurrentConfigGroupedByComposeEnabled)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByComposeEnabledKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConfigHandlerCurrentConfigGroupedByComposeEnabled) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func ConfigHandlerCurrentConfigIndexByComposeMcpToolEnabledKey(value *CurrentConfigView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.ComposeMcpToolEnabled == nil {
		return zero, false
	}
	return *value.ComposeMcpToolEnabled, true
}

type ConfigHandlerCurrentConfigIndexedByComposeMcpToolEnabled map[int]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) IndexByComposeMcpToolEnabled() (ConfigHandlerCurrentConfigIndexedByComposeMcpToolEnabled, error) {
	result := make(ConfigHandlerCurrentConfigIndexedByComposeMcpToolEnabled)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByComposeMcpToolEnabledKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConfigHandlerCurrentConfigSlice.IndexByComposeMcpToolEnabled")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConfigHandlerCurrentConfigIndexedByComposeMcpToolEnabled) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigGroupedByComposeMcpToolEnabled map[int][]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) GroupByComposeMcpToolEnabled() ConfigHandlerCurrentConfigGroupedByComposeMcpToolEnabled {
	result := make(ConfigHandlerCurrentConfigGroupedByComposeMcpToolEnabled)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByComposeMcpToolEnabledKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConfigHandlerCurrentConfigGroupedByComposeMcpToolEnabled) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigKey struct {
	ReportId  string
	VersionNo int
}

func ConfigHandlerCurrentConfigIndexByKeyKey(value *CurrentConfigView) (ConfigHandlerCurrentConfigKey, bool) {
	var zero ConfigHandlerCurrentConfigKey
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	return ConfigHandlerCurrentConfigKey{ReportId: *value.ReportId, VersionNo: *value.VersionNo}, true
}

type ConfigHandlerCurrentConfigIndexedByKey map[ConfigHandlerCurrentConfigKey]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) IndexByKey() (ConfigHandlerCurrentConfigIndexedByKey, error) {
	result := make(ConfigHandlerCurrentConfigIndexedByKey)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ConfigHandlerCurrentConfigSlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index ConfigHandlerCurrentConfigIndexedByKey) Has(key ConfigHandlerCurrentConfigKey) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerCurrentConfigGroupedByKey map[ConfigHandlerCurrentConfigKey][]*CurrentConfigView

func (rows ConfigHandlerCurrentConfigSlice) GroupByKey() ConfigHandlerCurrentConfigGroupedByKey {
	result := make(ConfigHandlerCurrentConfigGroupedByKey)
	for _, row := range rows {
		key, ok := ConfigHandlerCurrentConfigIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ConfigHandlerCurrentConfigGroupedByKey) Has(key ConfigHandlerCurrentConfigKey) bool {
	_, ok := index[key]
	return ok
}

type ConfigHandlerReadIndexes struct {
	CurrentConfig      ConfigHandlerCurrentConfigSlice
	CurrentConfigByKey ConfigHandlerCurrentConfigIndexedByKey
}

func BuildConfigHandlerReadIndexes(ctx context.Context, input *Input) (*ConfigHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &ConfigHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentConfig")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentConfig")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentConfig")
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
			if _, err := (xshape.Collection[CurrentConfigView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentConfigView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentConfigView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("DimensionsJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.DimensionsJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("MeasuresJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.MeasuresJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("FiltersJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.FiltersJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("OrderByJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.OrderByJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("InputLayoutJson") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.InputLayoutJson")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ReportId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("VersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.VersionNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ComposeMaxCubes") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.ComposeMaxCubes")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ComposeMaxLimit") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.ComposeMaxLimit")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ComposeTimeoutMs") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.ComposeTimeoutMs")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CubeEnabled") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.CubeEnabled")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CubeMcpToolEnabled") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.CubeMcpToolEnabled")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("LinkedInputType") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.LinkedInputType")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ComposeEnabled") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.ComposeEnabled")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ComposeMcpToolEnabled") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentConfig.ComposeMcpToolEnabled")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentConfig = ConfigHandlerCurrentConfigSlice(cloned.([]*CurrentConfigView))
		result.CurrentConfigByKey, err = result.CurrentConfig.IndexByKey()
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
	input._configHandlerReadIndexes = nil
	indexes, err := BuildConfigHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._configHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*ConfigHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._configHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._configHandlerReadIndexes, nil
}
