// Package sourcevalidation defines the trusted, optional main-DQL policy hook.
package sourcevalidation

import (
	"context"
	"github.com/viant/datly-studio/sdk"
)

// Rejection contains deliberate, public diagnostic text. Arbitrary errors remain private.
type Rejection struct{ Diagnostic sdk.Diagnostic }

func (r *Rejection) Error() string { return "component source policy rejected" }

type Input struct {
	// When true, the namespace requires a security declaration. A trusted hook
	// can additionally inspect mandatory application in the effective query.
	RequireSecurityPredicate bool
	PredicatePackages        []string
	NamespaceID              string
	ComponentID              string
	VersionNo                int
	SourceRevision           int64
	Package                  string
	AuthoredDQL              string
	DQL                      string
}

// Validator inspects immutable source text. It must not change the component or
// execute its business query. Any returned error rejects validation/publication.
type Validator func(context.Context, Input) error
