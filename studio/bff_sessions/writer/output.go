package writer

import (
	xhandler "github.com/viant/xdatly/handler"
	response "github.com/viant/xdatly/response"
)

// Output is the generated output scaffold for session.
type Output struct {
	Status     response.Status       `parameter:"Status,kind=output,in=status,dataType=response.Status" anonymous:"true"`
	Violations []*xhandler.Violation `parameter:"Violations,kind=output,in=status"`
	Data       []*SessionRevocation  `parameter:"Data,kind=output,in=body,dataType=[]*SessionRevocation"`
}
