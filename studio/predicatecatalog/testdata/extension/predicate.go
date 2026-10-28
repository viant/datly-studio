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

type ParameterizedScope struct{}

func (*ParameterizedScope) Compute(context.Context, any, ...any) (*predicate.Criteria, error) {
	return &predicate.Criteria{Expression: "1 = 0"}, nil
}

var LinkedVariadicType = reflect.TypeFor[ParameterizedScope]()

type WrongVariadicScope struct{}

func (*WrongVariadicScope) Compute(context.Context, any, ...string) (*predicate.Criteria, error) {
	return nil, nil
}

var LinkedWrongVariadicType = reflect.TypeFor[WrongVariadicScope]()

var LinkedType reflect.Type

func init() { LinkedType = reflect.TypeOf(ReaderScope{}) }
