package writer

import (
	"context"

	"github.com/viant/datly-studio/studio/authorization"
)

// Init validates explicitly supplied ownership before writer matching. Existing
// rows are separately constrained by the authored authorization context query.
func (i *Input) Init(context.Context) error {
	if i == nil {
		return authorization.AuthorizeOwner(nil, "")
	}
	for _, connector := range i.Connectors {
		if connector != nil && connector.OwnerId != nil {
			if err := authorization.AuthorizeOwner(i.Jwt, *connector.OwnerId); err != nil {
				return err
			}
		}
	}
	return nil
}
