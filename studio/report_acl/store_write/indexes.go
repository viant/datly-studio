package store_write

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
)

type AclHandlerCurrentAclSlice []*CurrentAclView

func AclHandlerCurrentAclIndexByReportIdKey(value *CurrentAclView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	return *value.ReportId, true
}

type AclHandlerCurrentAclIndexedByReportId map[string]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) IndexByReportId() (AclHandlerCurrentAclIndexedByReportId, error) {
	result := make(AclHandlerCurrentAclIndexedByReportId)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AclHandlerCurrentAclSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index AclHandlerCurrentAclIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerCurrentAclGroupedByReportId map[string][]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) GroupByReportId() AclHandlerCurrentAclGroupedByReportId {
	result := make(AclHandlerCurrentAclGroupedByReportId)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AclHandlerCurrentAclGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func AclHandlerCurrentAclIndexBySubjectTypeKey(value *CurrentAclView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SubjectType == nil {
		return zero, false
	}
	return *value.SubjectType, true
}

type AclHandlerCurrentAclIndexedBySubjectType map[string]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) IndexBySubjectType() (AclHandlerCurrentAclIndexedBySubjectType, error) {
	result := make(AclHandlerCurrentAclIndexedBySubjectType)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexBySubjectTypeKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AclHandlerCurrentAclSlice.IndexBySubjectType")
		}
		result[key] = row
	}
	return result, nil
}
func (index AclHandlerCurrentAclIndexedBySubjectType) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerCurrentAclGroupedBySubjectType map[string][]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) GroupBySubjectType() AclHandlerCurrentAclGroupedBySubjectType {
	result := make(AclHandlerCurrentAclGroupedBySubjectType)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexBySubjectTypeKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AclHandlerCurrentAclGroupedBySubjectType) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func AclHandlerCurrentAclIndexBySubjectIdKey(value *CurrentAclView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	if value.SubjectId == nil {
		return zero, false
	}
	return *value.SubjectId, true
}

type AclHandlerCurrentAclIndexedBySubjectId map[string]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) IndexBySubjectId() (AclHandlerCurrentAclIndexedBySubjectId, error) {
	result := make(AclHandlerCurrentAclIndexedBySubjectId)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexBySubjectIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AclHandlerCurrentAclSlice.IndexBySubjectId")
		}
		result[key] = row
	}
	return result, nil
}
func (index AclHandlerCurrentAclIndexedBySubjectId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerCurrentAclGroupedBySubjectId map[string][]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) GroupBySubjectId() AclHandlerCurrentAclGroupedBySubjectId {
	result := make(AclHandlerCurrentAclGroupedBySubjectId)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexBySubjectIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AclHandlerCurrentAclGroupedBySubjectId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func AclHandlerCurrentAclIndexByEtagKey(value *CurrentAclView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.Etag == nil {
		return zero, false
	}
	return *value.Etag, true
}

type AclHandlerCurrentAclIndexedByEtag map[int]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) IndexByEtag() (AclHandlerCurrentAclIndexedByEtag, error) {
	result := make(AclHandlerCurrentAclIndexedByEtag)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByEtagKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AclHandlerCurrentAclSlice.IndexByEtag")
		}
		result[key] = row
	}
	return result, nil
}
func (index AclHandlerCurrentAclIndexedByEtag) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerCurrentAclGroupedByEtag map[int][]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) GroupByEtag() AclHandlerCurrentAclGroupedByEtag {
	result := make(AclHandlerCurrentAclGroupedByEtag)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByEtagKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AclHandlerCurrentAclGroupedByEtag) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func AclHandlerCurrentAclIndexByCanViewKey(value *CurrentAclView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.CanView == nil {
		return zero, false
	}
	return *value.CanView, true
}

type AclHandlerCurrentAclIndexedByCanView map[int]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) IndexByCanView() (AclHandlerCurrentAclIndexedByCanView, error) {
	result := make(AclHandlerCurrentAclIndexedByCanView)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByCanViewKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AclHandlerCurrentAclSlice.IndexByCanView")
		}
		result[key] = row
	}
	return result, nil
}
func (index AclHandlerCurrentAclIndexedByCanView) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerCurrentAclGroupedByCanView map[int][]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) GroupByCanView() AclHandlerCurrentAclGroupedByCanView {
	result := make(AclHandlerCurrentAclGroupedByCanView)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByCanViewKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AclHandlerCurrentAclGroupedByCanView) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func AclHandlerCurrentAclIndexByCanRunKey(value *CurrentAclView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.CanRun == nil {
		return zero, false
	}
	return *value.CanRun, true
}

type AclHandlerCurrentAclIndexedByCanRun map[int]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) IndexByCanRun() (AclHandlerCurrentAclIndexedByCanRun, error) {
	result := make(AclHandlerCurrentAclIndexedByCanRun)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByCanRunKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AclHandlerCurrentAclSlice.IndexByCanRun")
		}
		result[key] = row
	}
	return result, nil
}
func (index AclHandlerCurrentAclIndexedByCanRun) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerCurrentAclGroupedByCanRun map[int][]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) GroupByCanRun() AclHandlerCurrentAclGroupedByCanRun {
	result := make(AclHandlerCurrentAclGroupedByCanRun)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByCanRunKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AclHandlerCurrentAclGroupedByCanRun) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func AclHandlerCurrentAclIndexByCanEditKey(value *CurrentAclView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.CanEdit == nil {
		return zero, false
	}
	return *value.CanEdit, true
}

type AclHandlerCurrentAclIndexedByCanEdit map[int]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) IndexByCanEdit() (AclHandlerCurrentAclIndexedByCanEdit, error) {
	result := make(AclHandlerCurrentAclIndexedByCanEdit)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByCanEditKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AclHandlerCurrentAclSlice.IndexByCanEdit")
		}
		result[key] = row
	}
	return result, nil
}
func (index AclHandlerCurrentAclIndexedByCanEdit) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerCurrentAclGroupedByCanEdit map[int][]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) GroupByCanEdit() AclHandlerCurrentAclGroupedByCanEdit {
	result := make(AclHandlerCurrentAclGroupedByCanEdit)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByCanEditKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AclHandlerCurrentAclGroupedByCanEdit) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func AclHandlerCurrentAclIndexByCanPublishKey(value *CurrentAclView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.CanPublish == nil {
		return zero, false
	}
	return *value.CanPublish, true
}

type AclHandlerCurrentAclIndexedByCanPublish map[int]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) IndexByCanPublish() (AclHandlerCurrentAclIndexedByCanPublish, error) {
	result := make(AclHandlerCurrentAclIndexedByCanPublish)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByCanPublishKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AclHandlerCurrentAclSlice.IndexByCanPublish")
		}
		result[key] = row
	}
	return result, nil
}
func (index AclHandlerCurrentAclIndexedByCanPublish) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerCurrentAclGroupedByCanPublish map[int][]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) GroupByCanPublish() AclHandlerCurrentAclGroupedByCanPublish {
	result := make(AclHandlerCurrentAclGroupedByCanPublish)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByCanPublishKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AclHandlerCurrentAclGroupedByCanPublish) Has(key int) bool {
	_, ok := index[key]
	return ok
}
func AclHandlerCurrentAclIndexByCanUseDqlKey(value *CurrentAclView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.CanUseDql == nil {
		return zero, false
	}
	return *value.CanUseDql, true
}

type AclHandlerCurrentAclIndexedByCanUseDql map[int]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) IndexByCanUseDql() (AclHandlerCurrentAclIndexedByCanUseDql, error) {
	result := make(AclHandlerCurrentAclIndexedByCanUseDql)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByCanUseDqlKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AclHandlerCurrentAclSlice.IndexByCanUseDql")
		}
		result[key] = row
	}
	return result, nil
}
func (index AclHandlerCurrentAclIndexedByCanUseDql) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerCurrentAclGroupedByCanUseDql map[int][]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) GroupByCanUseDql() AclHandlerCurrentAclGroupedByCanUseDql {
	result := make(AclHandlerCurrentAclGroupedByCanUseDql)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByCanUseDqlKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AclHandlerCurrentAclGroupedByCanUseDql) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerCurrentAclKey struct {
	ReportId    string
	SubjectType string
	SubjectId   string
}

func AclHandlerCurrentAclIndexByKeyKey(value *CurrentAclView) (AclHandlerCurrentAclKey, bool) {
	var zero AclHandlerCurrentAclKey
	if value == nil {
		return zero, false
	}
	if value.ReportId == nil {
		return zero, false
	}
	if value.SubjectType == nil {
		return zero, false
	}
	if value.SubjectId == nil {
		return zero, false
	}
	return AclHandlerCurrentAclKey{ReportId: *value.ReportId, SubjectType: *value.SubjectType, SubjectId: *value.SubjectId}, true
}

type AclHandlerCurrentAclIndexedByKey map[AclHandlerCurrentAclKey]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) IndexByKey() (AclHandlerCurrentAclIndexedByKey, error) {
	result := make(AclHandlerCurrentAclIndexedByKey)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByKeyKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index AclHandlerCurrentAclSlice.IndexByKey")
		}
		result[key] = row
	}
	return result, nil
}
func (index AclHandlerCurrentAclIndexedByKey) Has(key AclHandlerCurrentAclKey) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerCurrentAclGroupedByKey map[AclHandlerCurrentAclKey][]*CurrentAclView

func (rows AclHandlerCurrentAclSlice) GroupByKey() AclHandlerCurrentAclGroupedByKey {
	result := make(AclHandlerCurrentAclGroupedByKey)
	for _, row := range rows {
		key, ok := AclHandlerCurrentAclIndexByKeyKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index AclHandlerCurrentAclGroupedByKey) Has(key AclHandlerCurrentAclKey) bool {
	_, ok := index[key]
	return ok
}

type AclHandlerReadIndexes struct {
	CurrentAcl      AclHandlerCurrentAclSlice
	CurrentAclByKey AclHandlerCurrentAclIndexedByKey
}

func BuildAclHandlerReadIndexes(ctx context.Context, input *Input) (*AclHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &AclHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentAcl")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentAcl")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentAcl")
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
			if _, err := (xshape.Collection[CurrentAclView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentAclView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentAclView]{}).Pointers(value)
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
				return nil, fmt.Errorf("application index field was not loaded: CurrentAcl.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SubjectType") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAcl.SubjectType")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SubjectId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAcl.SubjectId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Etag") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAcl.Etag")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CanView") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAcl.CanView")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CanRun") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAcl.CanRun")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CanEdit") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAcl.CanEdit")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CanPublish") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAcl.CanPublish")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CanUseDql") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentAcl.CanUseDql")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentAcl = AclHandlerCurrentAclSlice(cloned.([]*CurrentAclView))
		result.CurrentAclByKey, err = result.CurrentAcl.IndexByKey()
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
	input._aclHandlerReadIndexes = nil
	indexes, err := BuildAclHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._aclHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*AclHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._aclHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._aclHandlerReadIndexes, nil
}
