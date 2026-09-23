package writer

import (
	xhandler "github.com/viant/xdatly/handler"
	response "github.com/viant/xdatly/response"
)

// Output is the generated output scaffold for policy.
type Output struct {
	response.Status `parameter:",kind=output,in=status"`
	Violations      []*xhandler.Violation `parameter:"Violations,kind=output,in=status"`
	Data            []*ResourcePolicyHead `parameter:"Data,kind=output,in=body,dataType=[]*ResourcePolicyHead"`
}
