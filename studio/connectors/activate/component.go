package activate

import (
	"reflect"

	status "github.com/viant/datly-studio/studio/connectors/disable"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/xdatly"
)

// Activation and disabling share the same verified identity, edit guard,
// redacted DTO, and optimistic writer. The handler chooses the server-owned
// operation; callers cannot set the store writer's operation parameter.
type Input = status.Input
type Output = status.Output

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.activate,method=POST,connector=studio,handler=NewConnectorActivate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.activate\",\"description\":\"Activate an authorized tested Datly Studio connector at an expected revision\"}]" caseFormat:"lc"`
}

var ConnectorDatly = new(Component)
var ConnectorDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewConnectorActivate" {
		return custom.Factory(status.NewConnectorActivate)
	}
	return nil
}
