package predicateprojection

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	storedreader "github.com/viant/datly-studio/studio/authorization_predicates/store_read"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
)

func Read(ctx context.Context, invoker exec.ComponentInvoker, name, query, status string, limit, offset int) ([]*storedreader.StoredAuthorizationPredicate, error) {
	if invoker == nil || limit < 1 || limit > 500 || offset < 0 {
		return nil, fmt.Errorf("authorization predicate reader arguments are invalid")
	}
	input := &storedreader.Input{}
	if name != "" {
		input.SetName(name)
	}
	pattern := ""
	if value := strings.TrimSpace(query); value != "" {
		pattern = "%" + strings.ToLower(value) + "%"
	}
	input.SetSearchPattern(pattern)
	input.SetStatus(status)
	input.SetOrderBy("updated_at DESC, name ASC")
	input.SetLimit(limit)
	input.SetOffset(offset)
	value, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[storedreader.AuthorizationPredicateComponent]().PkgPath(), Name: "authorization_predicate"},
		Route:     spec.RouteRef{Method: "GET", Path: "/_studio/authorization-predicate-store"},
	}, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*storedreader.Output)
	if !ok || output == nil || len(output.AuthorizationPredicates) > limit {
		return nil, fmt.Errorf("authorization predicate reader returned %T with invalid page", value)
	}
	return output.AuthorizationPredicates, nil
}
