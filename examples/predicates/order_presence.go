// Package predicates supplies opt-in examples for the standard Studio binaries.
// Embedding applications can omit this package entirely.
package predicates

import (
	"context"
	"fmt"
	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/xdatly/predicate"
	"reflect"
)

const PackagePath = "github.com/viant/datly-studio/examples/predicates"

// OrderPresence supports a combined eligibility condition: an order exists and
// no active exclusion exists. The individual modes remain available.
// The parent reader must use alias c and the documented example tables.
// This is a query filter, not an authorization decision or a source of roles.
type OrderPresence struct{}

// Description documents the handler in Studio's discovered predicate catalog.
func (*OrderPresence) Description() string {
	return "Filters customers by paid-order presence and active exclusions. Modes: eligible, exists, not_exists. Requires parent alias c."
}

var _ predicatecatalog.Describer = (*OrderPresence)(nil)

func (*OrderPresence) Compute(ctx context.Context, value any) (*predicate.Criteria, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mode, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("order presence must be eligible, exists or not_exists")
	}
	operator := ""
	switch mode {
	case "eligible":
		return &predicate.Criteria{
			Expression:   "EXISTS (SELECT 1 FROM predicate_presence_orders o WHERE o.customer_id = c.id AND o.status = ?) AND NOT EXISTS (SELECT 1 FROM predicate_presence_exclusions x WHERE x.customer_id = c.id AND x.active = ?)",
			Placeholders: []any{"paid", 1},
		}, nil
	case "exists":
		operator = "EXISTS"
	case "not_exists":
		operator = "NOT EXISTS"
	default:
		return nil, fmt.Errorf("order presence must be eligible, exists or not_exists")
	}
	return &predicate.Criteria{Expression: operator + " (SELECT 1 FROM predicate_presence_orders o WHERE o.customer_id = c.id AND o.status = ?)", Placeholders: []any{"paid"}}, nil
}

var LinkedType = reflect.TypeFor[OrderPresence]()

func init() {
	_ = LinkedType
	if err := host.RegisterPredicatePackage(predicatecatalog.Package{Alias: "examplepredicates", Path: PackagePath}); err != nil {
		panic(err)
	}
}
