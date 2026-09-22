package writer

import (
	"context"

	"github.com/viant/datly-studio/studio/authorization"
)

func (i *Input) Init(context.Context) error {
	if i == nil {
		return authorization.AuthorizeOwner(nil, "")
	}
	for _, report := range i.Reports {
		if report != nil && report.OwnerId != nil {
			if err := authorization.AuthorizeOwner(i.Jwt, *report.OwnerId); err != nil {
				return err
			}
		}
	}
	return nil
}
