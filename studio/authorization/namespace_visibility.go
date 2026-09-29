package authorization

import (
	"context"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/namespacevisibility"
	"github.com/viant/xdatly/connector"
	xpredicate "github.com/viant/xdatly/predicate"
	"strings"
)

func namespaceVisibilityCriteria(ctx context.Context, connectors connector.Provider, subject, bearer string) (*xpredicate.Criteria, error) {
	return namespacevisibility.Criteria(oauth.WithBearer(ctx, strings.TrimPrefix(bearer, "Bearer ")), connectors, subject, "namespaces")
}
