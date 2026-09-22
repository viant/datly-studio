package writer

import (
	response "github.com/viant/xdatly/response"
	xhandler "github.com/viant/xdatly/handler"
)

// Output is the generated output scaffold for version.
type Output struct {
	Status response.Status `parameter:"Status,kind=output,in=status,dataType=response.Status" anonymous:"true"`
	Violations []*xhandler.Violation `parameter:"Violations,kind=output,in=status"`
	Data []*ReportVersion `parameter:"Data,kind=output,in=body,dataType=[]*ReportVersion"`
}
