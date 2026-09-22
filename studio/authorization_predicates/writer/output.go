package writer

import (
	xhandler "github.com/viant/xdatly/handler"
	response "github.com/viant/xdatly/response"
)

// AuthorizationPredicateMutationOutput is the generated output scaffold for authorization_predicate.
type AuthorizationPredicateMutationOutput struct {
	Status     response.Status                         `parameter:"Status,kind=output,in=status,dataType=response.Status" anonymous:"true"`
	Violations []*xhandler.Violation                   `parameter:"Violations,kind=output,in=status"`
	Data       []*AuthorizationPredicateMutationRecord `parameter:"Data,kind=output,in=body,dataType=[]*AuthorizationPredicateMutationRecord"`
}
