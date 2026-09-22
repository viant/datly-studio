package writer

import (
	"context"

	"github.com/viant/datly-studio/studio/authorization"
)

// Init prevents clients from creating namespaces for, or transferring a
// namespace to, another verified principal. Existing rows are additionally
// constrained by the authorization-scoped CurrentNamespace view.
func (i *NamespaceMutationInput) Init(context.Context) error {
	if i == nil {
		return authorization.AuthorizeOwner(nil, "")
	}
	for _, namespace := range i.Namespaces {
		if namespace != nil && namespace.OwnerId != "" {
			if err := authorization.AuthorizeOwner(i.Jwt, namespace.OwnerId); err != nil {
				return err
			}
		}
	}
	return nil
}
