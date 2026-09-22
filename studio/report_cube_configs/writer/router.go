package writer

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for config.
type ConfigComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"config,path=/v1/studio/report-cube-configs,method=PATCH,connector=studio,view=config\" routeName:\"config\" mutation:\"patch\" caseFormat:\"lc\""
}

// ConfigDatlyType keeps the public component type linked for blank-import discovery.
func ConfigDatlyType() reflect.Type { return reflect.TypeOf((*ConfigComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var ConfigDatly = new(ConfigComponent)
var ConfigDatlyLinkedType = ConfigDatlyType()

func (ConfigComponent) EmbedFS() *embed.FS {
	return &ConfigDatlyResources
}

func (ConfigComponent) EmbedNamespace() string {
	return ConfigDatlyResourceNamespace
}
