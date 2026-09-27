package extension

import (
	"context"
	"reflect"

	"github.com/viant/xdatly/predicate"
)

type ReaderScope struct{}

func (*ReaderScope) Compute(context.Context, any) (*predicate.Criteria, error) {
	return &predicate.Criteria{Expression: "t.project_id = ?", Placeholders: []any{101}}, nil
}

type NotAPredicate struct{}

var LinkedType reflect.Type

func init() { LinkedType = reflect.TypeOf(ReaderScope{}) }
