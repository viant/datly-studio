package writer

import (
	xhandler "github.com/viant/xdatly/handler"
	response "github.com/viant/xdatly/response"
)

// NamespaceMutationOutput is the generated output scaffold for namespace.
type NamespaceMutationOutput struct {
	Status     response.Status            `parameter:"Status,kind=output,in=status,dataType=response.Status" anonymous:"true"`
	Violations []*xhandler.Violation      `parameter:"Violations,kind=output,in=status"`
	Data       []*NamespaceMutationRecord `parameter:"Data,kind=output,in=body,dataType=[]*NamespaceMutationRecord"`
}
