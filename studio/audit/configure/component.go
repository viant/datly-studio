package configure

import (
	"github.com/viant/datly-studio/internal/auditcontrol"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	"reflect"
)

type Component struct {
	Contract xdatly.Component[auditcontrol.Input, auditcontrol.Output] `component:"audit,path=/v1/studio/sdk/audit.configure,method=POST,connector=studio,handler=NewHandler" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.audit.configure\",\"description\":\"Audit configure with independent viewing and resource management permissions\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewHandler" {
		return custom.Factory(NewHandler)
	}
	return nil
}
func NewHandler() xhandler.Contract[auditcontrol.Input, auditcontrol.Output] {
	return &auditcontrol.Handler{Operation: "configure"}
}
