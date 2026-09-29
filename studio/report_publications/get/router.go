package get

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for publication.
type PublicationComponent struct {
	Contract xdatly.Component[PublicationGetInput, PublicationGetOutput] "component:\"publication,path=/v1/studio/sdk/publications.get,method=POST,connector=studio,view=publication\" routeName:\"publication\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.sdk.publications.get\\\",\\\"description\\\":\\\"Read one authorized Datly Studio publication\\\"}]\" caseFormat:\"lc\""
}

// PublicationDatlyType returns the public component type.
func PublicationDatlyType() reflect.Type { return reflect.TypeOf((*PublicationComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var PublicationDatly = new(PublicationComponent)
var _datlyReachablePublicationComponent = reflect.TypeFor[PublicationComponent]()

func (PublicationComponent) EmbedFS() *embed.FS {
	return &PublicationDatlyResources
}

func (PublicationComponent) EmbedNamespace() string {
	return PublicationDatlyResourceNamespace
}
