package store_write

import (
	xhandler "github.com/viant/xdatly/handler"
	response "github.com/viant/xdatly/response"
)

// Output is the generated output scaffold for authorization_predicate.
type Output struct {
	response.Status `parameter:",kind=output,in=status"`
	Violations      []*xhandler.Violation           `parameter:"Violations,kind=output,in=status"`
	Data            []*StoredAuthorizationPredicate `parameter:"Data,kind=output,in=body,dataType=[]*StoredAuthorizationPredicate"`
}
