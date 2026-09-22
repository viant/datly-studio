package writer

import (
	response "github.com/viant/xdatly/response"
	xhandler "github.com/viant/xdatly/handler"
)

// Output is the generated output scaffold for publication.
type Output struct {
	Status response.Status `parameter:"Status,kind=output,in=status,dataType=response.Status" anonymous:"true"`
	Violations []*xhandler.Violation `parameter:"Violations,kind=output,in=status"`
	Data []*ReportPublication `parameter:"Data,kind=output,in=body,dataType=[]*ReportPublication"`
}
