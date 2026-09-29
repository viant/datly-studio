package catalogpredicate

import (
	"context"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/namespacevisibility"
	"github.com/viant/xdatly/connector"
	xpredicate "github.com/viant/xdatly/predicate"
	"strings"
)

// RunNamespaceRead adds namespace visibility to the private run-access reader.
// The existing run permission remains independent of metadata view permission.
type RunNamespaceRead struct {
	Input         any                `bind:"kind=input,required"`
	NamespaceID   *string            `bind:"kind=header,in=X-Studio-Namespace"`
	Authorization string             `bind:"kind=header,in=Authorization"`
	Connectors    connector.Provider `bind:"kind=connector"`
}

func (p *RunNamespaceRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.NamespaceID == nil {
		return &xpredicate.Criteria{Expression: "1=1"}, nil
	}
	input, ok := p.Input.(interface{ NamespaceRunSubject() string })
	if !ok {
		return nil, forbidden("namespace run scope is unavailable")
	}
	return namespacevisibility.ComponentCriteria(oauth.WithBearer(ctx, strings.TrimPrefix(p.Authorization, "Bearer ")), p.Connectors, input.NamespaceRunSubject(), p.NamespaceID, "r.id")
}
