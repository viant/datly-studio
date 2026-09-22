package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
)

type FolderHandlerCurrentFolderSlice []*CurrentFolderView

func FolderHandlerCurrentFolderIndexByReportIdKey(value *CurrentFolderView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	return *value.ReportId, true
}

type FolderHandlerCurrentFolderIndexedByReportId map[string]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) IndexByReportId() (FolderHandlerCurrentFolderIndexedByReportId, error) {
	result := make(FolderHandlerCurrentFolderIndexedByReportId)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FolderHandlerCurrentFolderSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index FolderHandlerCurrentFolderIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type FolderHandlerCurrentFolderGroupedByReportId map[string][]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) GroupByReportId() FolderHandlerCurrentFolderGroupedByReportId {
	result := make(FolderHandlerCurrentFolderGroupedByReportId)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FolderHandlerCurrentFolderGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func FolderHandlerCurrentFolderIndexByVersionNoKey(value *CurrentFolderView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	return *value.VersionNo, true
}

type FolderHandlerCurrentFolderIndexedByVersionNo map[int]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) IndexByVersionNo() (FolderHandlerCurrentFolderIndexedByVersionNo, error) {
	result := make(FolderHandlerCurrentFolderIndexedByVersionNo)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FolderHandlerCurrentFolderSlice.IndexByVersionNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index FolderHandlerCurrentFolderIndexedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type FolderHandlerCurrentFolderGroupedByVersionNo map[int][]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) GroupByVersionNo() FolderHandlerCurrentFolderGroupedByVersionNo {
	result := make(FolderHandlerCurrentFolderGroupedByVersionNo)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FolderHandlerCurrentFolderGroupedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func FolderHandlerCurrentFolderIndexByFolderIdKey(value *CurrentFolderView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.FolderId == nil {
		return zero, false
	}
	return *value.FolderId, true
}

type FolderHandlerCurrentFolderIndexedByFolderId map[string]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) IndexByFolderId() (FolderHandlerCurrentFolderIndexedByFolderId, error) {
	result := make(FolderHandlerCurrentFolderIndexedByFolderId)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByFolderIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FolderHandlerCurrentFolderSlice.IndexByFolderId")
		}
		result[key] = row
	}
	return result, nil
}
func (index FolderHandlerCurrentFolderIndexedByFolderId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type FolderHandlerCurrentFolderGroupedByFolderId map[string][]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) GroupByFolderId() FolderHandlerCurrentFolderGroupedByFolderId {
	result := make(FolderHandlerCurrentFolderGroupedByFolderId)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByFolderIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FolderHandlerCurrentFolderGroupedByFolderId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func FolderHandlerCurrentFolderIndexByNamespaceKey(value *CurrentFolderView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.Namespace == nil {
		return zero, false
	}
	return *value.Namespace, true
}

type FolderHandlerCurrentFolderIndexedByNamespace map[string]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) IndexByNamespace() (FolderHandlerCurrentFolderIndexedByNamespace, error) {
	result := make(FolderHandlerCurrentFolderIndexedByNamespace)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByNamespaceKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FolderHandlerCurrentFolderSlice.IndexByNamespace")
		}
		result[key] = row
	}
	return result, nil
}
func (index FolderHandlerCurrentFolderIndexedByNamespace) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type FolderHandlerCurrentFolderGroupedByNamespace map[string][]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) GroupByNamespace() FolderHandlerCurrentFolderGroupedByNamespace {
	result := make(FolderHandlerCurrentFolderGroupedByNamespace)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByNamespaceKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FolderHandlerCurrentFolderGroupedByNamespace) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func FolderHandlerCurrentFolderIndexByRootPathKey(value *CurrentFolderView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.RootPath == nil {
		return zero, false
	}
	return *value.RootPath, true
}

type FolderHandlerCurrentFolderIndexedByRootPath map[string]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) IndexByRootPath() (FolderHandlerCurrentFolderIndexedByRootPath, error) {
	result := make(FolderHandlerCurrentFolderIndexedByRootPath)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByRootPathKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FolderHandlerCurrentFolderSlice.IndexByRootPath")
		}
		result[key] = row
	}
	return result, nil
}
func (index FolderHandlerCurrentFolderIndexedByRootPath) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type FolderHandlerCurrentFolderGroupedByRootPath map[string][]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) GroupByRootPath() FolderHandlerCurrentFolderGroupedByRootPath {
	result := make(FolderHandlerCurrentFolderGroupedByRootPath)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByRootPathKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FolderHandlerCurrentFolderGroupedByRootPath) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func FolderHandlerCurrentFolderIndexByUriPrefixKey(value *CurrentFolderView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.UriPrefix == nil {
		return zero, false
	}
	return *value.UriPrefix, true
}

type FolderHandlerCurrentFolderIndexedByUriPrefix map[string]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) IndexByUriPrefix() (FolderHandlerCurrentFolderIndexedByUriPrefix, error) {
	result := make(FolderHandlerCurrentFolderIndexedByUriPrefix)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByUriPrefixKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FolderHandlerCurrentFolderSlice.IndexByUriPrefix")
		}
		result[key] = row
	}
	return result, nil
}
func (index FolderHandlerCurrentFolderIndexedByUriPrefix) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type FolderHandlerCurrentFolderGroupedByUriPrefix map[string][]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) GroupByUriPrefix() FolderHandlerCurrentFolderGroupedByUriPrefix {
	result := make(FolderHandlerCurrentFolderGroupedByUriPrefix)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByUriPrefixKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FolderHandlerCurrentFolderGroupedByUriPrefix) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func FolderHandlerCurrentFolderIndexByOrdinalKey(value *CurrentFolderView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.Ordinal == nil {
		return zero, false
	}
	return *value.Ordinal, true
}

type FolderHandlerCurrentFolderIndexedByOrdinal map[int]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) IndexByOrdinal() (FolderHandlerCurrentFolderIndexedByOrdinal, error) {
	result := make(FolderHandlerCurrentFolderIndexedByOrdinal)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByOrdinalKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FolderHandlerCurrentFolderSlice.IndexByOrdinal")
		}
		result[key] = row
	}
	return result, nil
}
func (index FolderHandlerCurrentFolderIndexedByOrdinal) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type FolderHandlerCurrentFolderGroupedByOrdinal map[int][]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) GroupByOrdinal() FolderHandlerCurrentFolderGroupedByOrdinal {
	result := make(FolderHandlerCurrentFolderGroupedByOrdinal)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByOrdinalKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FolderHandlerCurrentFolderGroupedByOrdinal) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type FolderHandlerCurrentFolderKey struct {
	ReportId  string
	VersionNo int
	FolderId  string
}

func FolderHandlerCurrentFolderIndexByKeyKey(value *CurrentFolderView) (FolderHandlerCurrentFolderKey, bool) {
	var zero FolderHandlerCurrentFolderKey
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	if value.VersionNo == nil {
		return zero, false
	}
	if value.FolderId == nil {
		return zero, false
	}
	return FolderHandlerCurrentFolderKey{ReportId: *value.ReportId, VersionNo: *value.VersionNo, FolderId: *value.FolderId}, true
}

type FolderHandlerCurrentFolderIndexedByKey map[FolderHandlerCurrentFolderKey]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) IndexByKey() (FolderHandlerCurrentFolderIndexedByKey, error) {
	result := make(FolderHandlerCurrentFolderIndexedByKey)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FolderHandlerCurrentFolderSlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index FolderHandlerCurrentFolderIndexedByKey) Has(key FolderHandlerCurrentFolderKey) bool {
	_, ok := index[key]
	return ok
}

type FolderHandlerCurrentFolderGroupedByKey map[FolderHandlerCurrentFolderKey][]*CurrentFolderView

func (rows FolderHandlerCurrentFolderSlice) GroupByKey() FolderHandlerCurrentFolderGroupedByKey {
	result := make(FolderHandlerCurrentFolderGroupedByKey)
	for _, row := range rows {
		key, ok := FolderHandlerCurrentFolderIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FolderHandlerCurrentFolderGroupedByKey) Has(key FolderHandlerCurrentFolderKey) bool {
	_, ok := index[key]
	return ok
}

type FolderHandlerReadIndexes struct {
	CurrentFolder      FolderHandlerCurrentFolderSlice
	CurrentFolderByKey FolderHandlerCurrentFolderIndexedByKey
}

func BuildFolderHandlerReadIndexes(ctx context.Context, input *Input) (*FolderHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &FolderHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentFolder")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentFolder")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentFolder")
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
			if _, err := (xshape.Collection[CurrentFolderView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentFolderView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentFolderView]{}).Pointers(value)
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
				return nil, fmt.Errorf("application index field was not loaded: CurrentFolder.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("VersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFolder.VersionNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("FolderId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFolder.FolderId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Namespace") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFolder.Namespace")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("RootPath") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFolder.RootPath")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("UriPrefix") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFolder.UriPrefix")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Ordinal") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFolder.Ordinal")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentFolder = FolderHandlerCurrentFolderSlice(cloned.([]*CurrentFolderView))
		result.CurrentFolderByKey, err = result.CurrentFolder.IndexByKey()
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
	input._folderHandlerReadIndexes = nil
	indexes, err := BuildFolderHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._folderHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*FolderHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._folderHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._folderHandlerReadIndexes, nil
}
