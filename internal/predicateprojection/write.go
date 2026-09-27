package predicateprojection

import (
	"context"
	"fmt"
	"reflect"

	stored "github.com/viant/datly-studio/studio/authorization_predicates/store_write"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
)

// Write invokes the selected internal optimistic mutation component. Only
// authorized public workflows should call it; this reader/writer is not an
// HTTP or MCP endpoint.
func Write(ctx context.Context, invoker exec.ComponentInvoker, row *stored.StoredAuthorizationPredicate) error {
	if invoker == nil || row == nil {
		return fmt.Errorf("authorization predicate writer is unavailable")
	}
	input := &stored.Input{}
	input.SetAuthorizationPredicates([]*stored.StoredAuthorizationPredicate{row})
	value, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[stored.AuthorizationPredicateComponent]().PkgPath(), Name: "authorization_predicate"},
		Route:     spec.RouteRef{Method: "PATCH", Path: "/_studio/authorization-predicate-store"},
	}, Input: input})
	if err != nil {
		return err
	}
	output, ok := value.(*stored.Output)
	if !ok || output == nil || len(output.Data) != 1 || output.Data[0] == nil || output.Data[0].Name != row.Name {
		return fmt.Errorf("authorization predicate writer returned %T without one matching row", value)
	}
	return nil
}
