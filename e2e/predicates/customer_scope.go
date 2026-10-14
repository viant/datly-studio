// Package predicates contains synthetic example handlers linked by the e2e host.
package predicates

import (
	"context"
	"fmt"
	"github.com/viant/xdatly/predicate"
	"reflect"
)

type CustomerScope struct{}

// Compute binds the declared DQL tenant input. It never interpolates it into SQL.
func (*CustomerScope) Compute(_ context.Context, value any) (*predicate.Criteria, error) {
	tenant, ok := value.(string)
	if !ok || tenant == "" {
		return nil, fmt.Errorf("tenant is required")
	}
	return &predicate.Criteria{Expression: "c.tenant = ?", Placeholders: []any{tenant}}, nil
}

// Keep the handler reachable so package-string runtime scanning can find it.
var LinkedType = reflect.TypeFor[CustomerScope]()
