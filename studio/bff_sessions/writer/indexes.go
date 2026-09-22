package writer

import (
	context "context"
	fmt "fmt"
	xshape "github.com/viant/x/shape"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
)

type SessionHandlerCurrentSessionSlice []*CurrentSessionView

func SessionHandlerCurrentSessionIndexBySessionIdHashKey(value *CurrentSessionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.SessionIdHash, true
}

type SessionHandlerCurrentSessionIndexedBySessionIdHash map[string]*CurrentSessionView

func (rows SessionHandlerCurrentSessionSlice) IndexBySessionIdHash() (SessionHandlerCurrentSessionIndexedBySessionIdHash, error) {
	result := make(SessionHandlerCurrentSessionIndexedBySessionIdHash)
	for _, row := range rows {
		key, ok := SessionHandlerCurrentSessionIndexBySessionIdHashKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index SessionHandlerCurrentSessionSlice.IndexBySessionIdHash")
		}
		result[key] = row
	}
	return result, nil
}
func (index SessionHandlerCurrentSessionIndexedBySessionIdHash) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type SessionHandlerCurrentSessionGroupedBySessionIdHash map[string][]*CurrentSessionView

func (rows SessionHandlerCurrentSessionSlice) GroupBySessionIdHash() SessionHandlerCurrentSessionGroupedBySessionIdHash {
	result := make(SessionHandlerCurrentSessionGroupedBySessionIdHash)
	for _, row := range rows {
		key, ok := SessionHandlerCurrentSessionIndexBySessionIdHashKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index SessionHandlerCurrentSessionGroupedBySessionIdHash) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func SessionHandlerCurrentSessionIndexBySubjectIdKey(value *CurrentSessionView) (string, bool) {
	var zero string
	if value == nil {
		return zero, false
	}
	return value.SubjectId, true
}

type SessionHandlerCurrentSessionIndexedBySubjectId map[string]*CurrentSessionView

func (rows SessionHandlerCurrentSessionSlice) IndexBySubjectId() (SessionHandlerCurrentSessionIndexedBySubjectId, error) {
	result := make(SessionHandlerCurrentSessionIndexedBySubjectId)
	for _, row := range rows {
		key, ok := SessionHandlerCurrentSessionIndexBySubjectIdKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index SessionHandlerCurrentSessionSlice.IndexBySubjectId")
		}
		result[key] = row
	}
	return result, nil
}
func (index SessionHandlerCurrentSessionIndexedBySubjectId) Has(key string) bool {
	_, ok := index[key]
	return ok
}

type SessionHandlerCurrentSessionGroupedBySubjectId map[string][]*CurrentSessionView

func (rows SessionHandlerCurrentSessionSlice) GroupBySubjectId() SessionHandlerCurrentSessionGroupedBySubjectId {
	result := make(SessionHandlerCurrentSessionGroupedBySubjectId)
	for _, row := range rows {
		key, ok := SessionHandlerCurrentSessionIndexBySubjectIdKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index SessionHandlerCurrentSessionGroupedBySubjectId) Has(key string) bool {
	_, ok := index[key]
	return ok
}
func SessionHandlerCurrentSessionIndexByExpiresAtUnixKey(value *CurrentSessionView) (int, bool) {
	var zero int
	if value == nil {
		return zero, false
	}
	if value.ExpiresAtUnix == nil {
		return zero, false
	}
	return *value.ExpiresAtUnix, true
}

type SessionHandlerCurrentSessionIndexedByExpiresAtUnix map[int]*CurrentSessionView

func (rows SessionHandlerCurrentSessionSlice) IndexByExpiresAtUnix() (SessionHandlerCurrentSessionIndexedByExpiresAtUnix, error) {
	result := make(SessionHandlerCurrentSessionIndexedByExpiresAtUnix)
	for _, row := range rows {
		key, ok := SessionHandlerCurrentSessionIndexByExpiresAtUnixKey(row)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("ambiguous application index SessionHandlerCurrentSessionSlice.IndexByExpiresAtUnix")
		}
		result[key] = row
	}
	return result, nil
}
func (index SessionHandlerCurrentSessionIndexedByExpiresAtUnix) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type SessionHandlerCurrentSessionGroupedByExpiresAtUnix map[int][]*CurrentSessionView

func (rows SessionHandlerCurrentSessionSlice) GroupByExpiresAtUnix() SessionHandlerCurrentSessionGroupedByExpiresAtUnix {
	result := make(SessionHandlerCurrentSessionGroupedByExpiresAtUnix)
	for _, row := range rows {
		key, ok := SessionHandlerCurrentSessionIndexByExpiresAtUnixKey(row)
		if !ok {
			continue
		}
		result[key] = append(result[key], row)
	}
	return result
}
func (index SessionHandlerCurrentSessionGroupedByExpiresAtUnix) Has(key int) bool {
	_, ok := index[key]
	return ok
}

type SessionHandlerReadIndexes struct {
	CurrentSession                SessionHandlerCurrentSessionSlice
	CurrentSessionBySessionIdHash SessionHandlerCurrentSessionIndexedBySessionIdHash
}

func BuildSessionHandlerReadIndexes(ctx context.Context, input *Input) (*SessionHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	metadata, ok := xhandler2.ReadMetadataFromContext(ctx)
	if !ok || (xshape.Runtime{}).IsNil(metadata) {
		return nil, fmt.Errorf("read indexes require bound read metadata")
	}
	result := &SessionHandlerReadIndexes{}
	{
		projection, err := metadata.Projection("CurrentSession")
		if err != nil {
			return nil, err
		}
		if (xshape.Runtime{}).IsNil(projection) {
			return nil, fmt.Errorf("application read projection is unavailable: CurrentSession")
		}
		inputAccess, err := xshape.Linked(reflect.TypeOf((*Input)(nil)).Elem()).Accessor("CurrentSession")
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
			if _, err := (xshape.Collection[CurrentSessionView]{}).Pointers(reflect.Zero(accessor.Type()).Interface()); err != nil {
				return nil, err
			}
			root, present, err := accessor.GetOptional(value)
			if err != nil {
				return nil, err
			}
			if present {
				value = root.Interface()
			} else {
				value = []*CurrentSessionView(nil)
			}
		}
		rows, err := (xshape.Collection[CurrentSessionView]{}).Pointers(value)
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
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SessionIdHash") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentSession.SessionIdHash")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("SubjectId") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentSession.SubjectId")
			}
			if (xshape.Runtime{}).IsNil(loaded) || !loaded.Has("ExpiresAtUnix") {
				return nil, fmt.Errorf("application index field was not loaded: CurrentSession.ExpiresAtUnix")
			}
		}
		cloned, err := (xshape.Runtime{}).CloneValue(rows, xshape.CloneOptions{})
		if err != nil {
			return nil, err
		}
		result.CurrentSession = SessionHandlerCurrentSessionSlice(cloned.([]*CurrentSessionView))
		result.CurrentSessionBySessionIdHash, err = result.CurrentSession.IndexBySessionIdHash()
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
	input._sessionHandlerReadIndexes = nil
	indexes, err := BuildSessionHandlerReadIndexes(ctx, input)
	if err := err; err != nil {
		return err
	}
	input._sessionHandlerReadIndexes = indexes
	return nil
}
func (input *Input) ReadIndexes(ctx context.Context) (*SessionHandlerReadIndexes, error) {
	if input == nil {
		return nil, fmt.Errorf("read indexes require input")
	}
	if input._sessionHandlerReadIndexes == nil {
		if err := input.PrepareReadIndexes(ctx); err != nil {
			return nil, err
		}
	}
	return input._sessionHandlerReadIndexes, nil
}
