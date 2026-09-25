package store_write

import (
	xhandler "github.com/viant/xdatly/handler"
	response "github.com/viant/xdatly/response"
)

// Output is the generated output scaffold for acl.
type Output struct {
	response.Status `parameter:",kind=output,in=status"`
	Violations      []*xhandler.Violation `parameter:"Violations,kind=output,in=status"`
	Data            []*StoredACL          `parameter:"Data,kind=output,in=body,dataType=[]*StoredACL"`
}
