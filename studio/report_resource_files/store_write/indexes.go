package store_write

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
)

type FileHandlerCurrentFileSlice []*CurrentFileView

func FileHandlerCurrentFileIndexByReportIdKey(value *CurrentFileView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ReportId, true
}

type FileHandlerCurrentFileIndexedByReportId map[string]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) IndexByReportId() (FileHandlerCurrentFileIndexedByReportId, error) {
	result := make(FileHandlerCurrentFileIndexedByReportId)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FileHandlerCurrentFileSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index FileHandlerCurrentFileIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileGroupedByReportId map[string][]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) GroupByReportId() FileHandlerCurrentFileGroupedByReportId {
	result := make(FileHandlerCurrentFileGroupedByReportId)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FileHandlerCurrentFileGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func FileHandlerCurrentFileIndexByVersionNoKey(value *CurrentFileView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	return value.VersionNo, true
}

type FileHandlerCurrentFileIndexedByVersionNo map[int]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) IndexByVersionNo() (FileHandlerCurrentFileIndexedByVersionNo, error) {
	result := make(FileHandlerCurrentFileIndexedByVersionNo)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FileHandlerCurrentFileSlice.IndexByVersionNo")
		}
		result[key] = row
	}
	return result, nil
}
func (index FileHandlerCurrentFileIndexedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileGroupedByVersionNo map[int][]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) GroupByVersionNo() FileHandlerCurrentFileGroupedByVersionNo {
	result := make(FileHandlerCurrentFileGroupedByVersionNo)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByVersionNoKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FileHandlerCurrentFileGroupedByVersionNo) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func FileHandlerCurrentFileIndexByResourceIdKey(value *CurrentFileView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ResourceId, true
}

type FileHandlerCurrentFileIndexedByResourceId map[string]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) IndexByResourceId() (FileHandlerCurrentFileIndexedByResourceId, error) {
	result := make(FileHandlerCurrentFileIndexedByResourceId)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByResourceIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FileHandlerCurrentFileSlice.IndexByResourceId")
		}
		result[key] = row
	}
	return result, nil
}
func (index FileHandlerCurrentFileIndexedByResourceId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileGroupedByResourceId map[string][]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) GroupByResourceId() FileHandlerCurrentFileGroupedByResourceId {
	result := make(FileHandlerCurrentFileGroupedByResourceId)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByResourceIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FileHandlerCurrentFileGroupedByResourceId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func FileHandlerCurrentFileIndexByNamespaceKey(value *CurrentFileView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Namespace, true
}

type FileHandlerCurrentFileIndexedByNamespace map[string]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) IndexByNamespace() (FileHandlerCurrentFileIndexedByNamespace, error) {
	result := make(FileHandlerCurrentFileIndexedByNamespace)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByNamespaceKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FileHandlerCurrentFileSlice.IndexByNamespace")
		}
		result[key] = row
	}
	return result, nil
}
func (index FileHandlerCurrentFileIndexedByNamespace) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileGroupedByNamespace map[string][]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) GroupByNamespace() FileHandlerCurrentFileGroupedByNamespace {
	result := make(FileHandlerCurrentFileGroupedByNamespace)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByNamespaceKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FileHandlerCurrentFileGroupedByNamespace) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func FileHandlerCurrentFileIndexByResourcePathKey(value *CurrentFileView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ResourcePath, true
}

type FileHandlerCurrentFileIndexedByResourcePath map[string]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) IndexByResourcePath() (FileHandlerCurrentFileIndexedByResourcePath, error) {
	result := make(FileHandlerCurrentFileIndexedByResourcePath)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByResourcePathKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FileHandlerCurrentFileSlice.IndexByResourcePath")
		}
		result[key] = row
	}
	return result, nil
}
func (index FileHandlerCurrentFileIndexedByResourcePath) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileGroupedByResourcePath map[string][]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) GroupByResourcePath() FileHandlerCurrentFileGroupedByResourcePath {
	result := make(FileHandlerCurrentFileGroupedByResourcePath)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByResourcePathKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FileHandlerCurrentFileGroupedByResourcePath) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func FileHandlerCurrentFileIndexByMediaTypeKey(value *CurrentFileView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.MediaType == nil {
		return zero, false
	}
	return *value.MediaType, true
}

type FileHandlerCurrentFileIndexedByMediaType map[string]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) IndexByMediaType() (FileHandlerCurrentFileIndexedByMediaType, error) {
	result := make(FileHandlerCurrentFileIndexedByMediaType)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByMediaTypeKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FileHandlerCurrentFileSlice.IndexByMediaType")
		}
		result[key] = row
	}
	return result, nil
}
func (index FileHandlerCurrentFileIndexedByMediaType) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileGroupedByMediaType map[string][]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) GroupByMediaType() FileHandlerCurrentFileGroupedByMediaType {
	result := make(FileHandlerCurrentFileGroupedByMediaType)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByMediaTypeKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FileHandlerCurrentFileGroupedByMediaType) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func FileHandlerCurrentFileIndexByContentSizeKey(value *CurrentFileView) (int64, bool) {
	var zero int64
	if value == nil {
		return zero, false
	}
	return value.ContentSize, true
}

type FileHandlerCurrentFileIndexedByContentSize map[int64]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) IndexByContentSize() (FileHandlerCurrentFileIndexedByContentSize, error) {
	result := make(FileHandlerCurrentFileIndexedByContentSize)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByContentSizeKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FileHandlerCurrentFileSlice.IndexByContentSize")
		}
		result[key] = row
	}
	return result, nil
}
func (index FileHandlerCurrentFileIndexedByContentSize) Has(key int64) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileGroupedByContentSize map[int64][]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) GroupByContentSize() FileHandlerCurrentFileGroupedByContentSize {
	result := make(FileHandlerCurrentFileGroupedByContentSize)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByContentSizeKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FileHandlerCurrentFileGroupedByContentSize) Has(key int64) bool {
	_, ok := index[key]
	return ok
}
func FileHandlerCurrentFileIndexByContentSha256Key(value *CurrentFileView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ContentSha256, true
}

type FileHandlerCurrentFileIndexedByContentSha256 map[string]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) IndexByContentSha256() (FileHandlerCurrentFileIndexedByContentSha256, error) {
	result := make(FileHandlerCurrentFileIndexedByContentSha256)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByContentSha256Key(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FileHandlerCurrentFileSlice.IndexByContentSha256")
		}
		result[key] = row
	}
	return result, nil
}
func (index FileHandlerCurrentFileIndexedByContentSha256) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileGroupedByContentSha256 map[string][]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) GroupByContentSha256() FileHandlerCurrentFileGroupedByContentSha256 {
	result := make(FileHandlerCurrentFileGroupedByContentSha256)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByContentSha256Key(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FileHandlerCurrentFileGroupedByContentSha256) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func FileHandlerCurrentFileIndexByIsBinaryKey(value *CurrentFileView) (bool, bool) {
	var zero bool
	if value == nil {
		return zero, false
	}
	return value.IsBinary, true
}

type FileHandlerCurrentFileIndexedByIsBinary map[bool]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) IndexByIsBinary() (FileHandlerCurrentFileIndexedByIsBinary, error) {
	result := make(FileHandlerCurrentFileIndexedByIsBinary)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByIsBinaryKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FileHandlerCurrentFileSlice.IndexByIsBinary")
		}
		result[key] = row
	}
	return result, nil
}
func (index FileHandlerCurrentFileIndexedByIsBinary) Has(key bool) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileGroupedByIsBinary map[bool][]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) GroupByIsBinary() FileHandlerCurrentFileGroupedByIsBinary {
	result := make(FileHandlerCurrentFileGroupedByIsBinary)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByIsBinaryKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FileHandlerCurrentFileGroupedByIsBinary) Has(key bool) bool {
	_, ok := index[key]
	return ok
}
func FileHandlerCurrentFileIndexByCreatedAtKey(value *CurrentFileView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	return value.CreatedAt, true
}

type FileHandlerCurrentFileIndexedByCreatedAt map[time.Time]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) IndexByCreatedAt() (FileHandlerCurrentFileIndexedByCreatedAt, error) {
	result := make(FileHandlerCurrentFileIndexedByCreatedAt)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FileHandlerCurrentFileSlice.IndexByCreatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index FileHandlerCurrentFileIndexedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileGroupedByCreatedAt map[time.Time][]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) GroupByCreatedAt() FileHandlerCurrentFileGroupedByCreatedAt {
	result := make(FileHandlerCurrentFileGroupedByCreatedAt)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FileHandlerCurrentFileGroupedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileKey struct {
	ReportId   string
	VersionNo  int
	ResourceId string
}

func FileHandlerCurrentFileIndexByKeyKey(value *CurrentFileView) (FileHandlerCurrentFileKey, bool) {
	var zero FileHandlerCurrentFileKey
	if value == nil {
		return zero, false
	}
	return FileHandlerCurrentFileKey{ReportId: value.ReportId, VersionNo: value.VersionNo, ResourceId: value.ResourceId}, true
}

type FileHandlerCurrentFileIndexedByKey map[FileHandlerCurrentFileKey]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) IndexByKey() (FileHandlerCurrentFileIndexedByKey, error) {
	result := make(FileHandlerCurrentFileIndexedByKey)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index FileHandlerCurrentFileSlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index FileHandlerCurrentFileIndexedByKey) Has(key FileHandlerCurrentFileKey) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerCurrentFileGroupedByKey map[FileHandlerCurrentFileKey][]*CurrentFileView

func (rows FileHandlerCurrentFileSlice) GroupByKey() FileHandlerCurrentFileGroupedByKey {
	result := make(FileHandlerCurrentFileGroupedByKey)
	for _, row := range rows {
		key, ok := FileHandlerCurrentFileIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index FileHandlerCurrentFileGroupedByKey) Has(key FileHandlerCurrentFileKey) bool {
	_, ok := index[key]
	return ok
}

type FileHandlerReadIndexes struct {
	CurrentFile      FileHandlerCurrentFileSlice
	CurrentFileByKey FileHandlerCurrentFileIndexedByKey
}

func BuildFileHandlerReadIndexes(ctx context.Context, input *Input) (*FileHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &FileHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentFile")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentFile")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentFile")
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
			if _, err := (xshape.Collection[CurrentFileView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentFileView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentFileView]{}).Pointers(value)
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
				return nil, fmt.Errorf("application index field was not loaded: CurrentFile.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("VersionNo") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFile.VersionNo")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ResourceId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFile.ResourceId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Namespace") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFile.Namespace")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ResourcePath") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFile.ResourcePath")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("MediaType") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFile.MediaType")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Content") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFile.Content")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ContentSize") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFile.ContentSize")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ContentSha256") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFile.ContentSha256")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("IsBinary") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFile.IsBinary")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CreatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentFile.CreatedAt")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentFile = FileHandlerCurrentFileSlice(cloned.([]*CurrentFileView))
		result.CurrentFileByKey, err = result.CurrentFile.IndexByKey()
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
	input._fileHandlerReadIndexes = nil
	indexes, err := BuildFileHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._fileHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*FileHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._fileHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._fileHandlerReadIndexes, nil
}
