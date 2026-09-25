package store_write

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	time "time"
)

type ClaimHandlerCurrentClaimSlice []*CurrentClaimView

func ClaimHandlerCurrentClaimIndexByNamespaceKey(value *CurrentClaimView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.Namespace, true
}

type ClaimHandlerCurrentClaimIndexedByNamespace map[string]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) IndexByNamespace() (ClaimHandlerCurrentClaimIndexedByNamespace, error) {
	result := make(ClaimHandlerCurrentClaimIndexedByNamespace)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByNamespaceKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ClaimHandlerCurrentClaimSlice.IndexByNamespace")
		}
		result[key] = row
	}
	return result, nil
}
func (index ClaimHandlerCurrentClaimIndexedByNamespace) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ClaimHandlerCurrentClaimGroupedByNamespace map[string][]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) GroupByNamespace() ClaimHandlerCurrentClaimGroupedByNamespace {
	result := make(ClaimHandlerCurrentClaimGroupedByNamespace)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByNamespaceKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ClaimHandlerCurrentClaimGroupedByNamespace) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ClaimHandlerCurrentClaimIndexByReportIdKey(value *CurrentClaimView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.ReportId, true
}

type ClaimHandlerCurrentClaimIndexedByReportId map[string]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) IndexByReportId() (ClaimHandlerCurrentClaimIndexedByReportId, error) {
	result := make(ClaimHandlerCurrentClaimIndexedByReportId)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByReportIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ClaimHandlerCurrentClaimSlice.IndexByReportId")
		}
		result[key] = row
	}
	return result, nil
}
func (index ClaimHandlerCurrentClaimIndexedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ClaimHandlerCurrentClaimGroupedByReportId map[string][]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) GroupByReportId() ClaimHandlerCurrentClaimGroupedByReportId {
	result := make(ClaimHandlerCurrentClaimGroupedByReportId)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByReportIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ClaimHandlerCurrentClaimGroupedByReportId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ClaimHandlerCurrentClaimIndexByCreatedAtKey(value *CurrentClaimView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	return value.CreatedAt, true
}

type ClaimHandlerCurrentClaimIndexedByCreatedAt map[time.Time]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) IndexByCreatedAt() (ClaimHandlerCurrentClaimIndexedByCreatedAt, error) {
	result := make(ClaimHandlerCurrentClaimIndexedByCreatedAt)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ClaimHandlerCurrentClaimSlice.IndexByCreatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index ClaimHandlerCurrentClaimIndexedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type ClaimHandlerCurrentClaimGroupedByCreatedAt map[time.Time][]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) GroupByCreatedAt() ClaimHandlerCurrentClaimGroupedByCreatedAt {
	result := make(ClaimHandlerCurrentClaimGroupedByCreatedAt)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByCreatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ClaimHandlerCurrentClaimGroupedByCreatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func ClaimHandlerCurrentClaimIndexByCreatedByKey(value *CurrentClaimView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.CreatedBy, true
}

type ClaimHandlerCurrentClaimIndexedByCreatedBy map[string]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) IndexByCreatedBy() (ClaimHandlerCurrentClaimIndexedByCreatedBy, error) {
	result := make(ClaimHandlerCurrentClaimIndexedByCreatedBy)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByCreatedByKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ClaimHandlerCurrentClaimSlice.IndexByCreatedBy")
		}
		result[key] = row
	}
	return result, nil
}
func (index ClaimHandlerCurrentClaimIndexedByCreatedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ClaimHandlerCurrentClaimGroupedByCreatedBy map[string][]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) GroupByCreatedBy() ClaimHandlerCurrentClaimGroupedByCreatedBy {
	result := make(ClaimHandlerCurrentClaimGroupedByCreatedBy)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByCreatedByKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ClaimHandlerCurrentClaimGroupedByCreatedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func ClaimHandlerCurrentClaimIndexByUpdatedAtKey(value *CurrentClaimView) (time.Time, bool) {
	var zero time.Time
	if value == nil {
		return zero, false
	}
	return value.UpdatedAt, true
}

type ClaimHandlerCurrentClaimIndexedByUpdatedAt map[time.Time]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) IndexByUpdatedAt() (ClaimHandlerCurrentClaimIndexedByUpdatedAt, error) {
	result := make(ClaimHandlerCurrentClaimIndexedByUpdatedAt)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ClaimHandlerCurrentClaimSlice.IndexByUpdatedAt")
		}
		result[key] = row
	}
	return result, nil
}
func (index ClaimHandlerCurrentClaimIndexedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}

type ClaimHandlerCurrentClaimGroupedByUpdatedAt map[time.Time][]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) GroupByUpdatedAt() ClaimHandlerCurrentClaimGroupedByUpdatedAt {
	result := make(ClaimHandlerCurrentClaimGroupedByUpdatedAt)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByUpdatedAtKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ClaimHandlerCurrentClaimGroupedByUpdatedAt) Has(key time.Time) bool {
	_, ok := index[key]
	return ok
}
func ClaimHandlerCurrentClaimIndexByUpdatedByKey(value *CurrentClaimView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.UpdatedBy, true
}

type ClaimHandlerCurrentClaimIndexedByUpdatedBy map[string]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) IndexByUpdatedBy() (ClaimHandlerCurrentClaimIndexedByUpdatedBy, error) {
	result := make(ClaimHandlerCurrentClaimIndexedByUpdatedBy)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByUpdatedByKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index ClaimHandlerCurrentClaimSlice.IndexByUpdatedBy")
		}
		result[key] = row
	}
	return result, nil
}
func (index ClaimHandlerCurrentClaimIndexedByUpdatedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ClaimHandlerCurrentClaimGroupedByUpdatedBy map[string][]*CurrentClaimView

func (rows ClaimHandlerCurrentClaimSlice) GroupByUpdatedBy() ClaimHandlerCurrentClaimGroupedByUpdatedBy {
	result := make(ClaimHandlerCurrentClaimGroupedByUpdatedBy)
	for _, row := range rows {
		key, ok := ClaimHandlerCurrentClaimIndexByUpdatedByKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index ClaimHandlerCurrentClaimGroupedByUpdatedBy) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type ClaimHandlerReadIndexes struct {
	CurrentClaim            ClaimHandlerCurrentClaimSlice
	CurrentClaimByNamespace ClaimHandlerCurrentClaimIndexedByNamespace
}

func BuildClaimHandlerReadIndexes(ctx context.Context, input *Input) (*ClaimHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &ClaimHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentClaim")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentClaim")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentClaim")
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
			if _, err := (xshape.Collection[CurrentClaimView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentClaimView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentClaimView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("Namespace") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentClaim.Namespace")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ReportId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentClaim.ReportId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CreatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentClaim.CreatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("CreatedBy") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentClaim.CreatedBy")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("UpdatedAt") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentClaim.UpdatedAt")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("UpdatedBy") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentClaim.UpdatedBy")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentClaim = ClaimHandlerCurrentClaimSlice(cloned.([]*CurrentClaimView))
		result.CurrentClaimByNamespace, err = result.CurrentClaim.IndexByNamespace()
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
	input._claimHandlerReadIndexes = nil
	indexes, err := BuildClaimHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._claimHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*ClaimHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._claimHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._claimHandlerReadIndexes, nil
}
