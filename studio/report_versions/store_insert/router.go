package store_insert

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for version.
type VersionComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"version,path=/_studio/report-version-store/insert,method=POST,connector=studio,view=version,internal=true\" routeName:\"version\" mutation:\"post\" caseFormat:\"lc\""
}

// VersionDatlyType returns the public component type.
func VersionDatlyType() reflect.Type { return reflect.TypeOf((*VersionComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var VersionDatly = new(VersionComponent)
var _datlyReachableVersionComponent = reflect.TypeFor[VersionComponent]()

func (VersionComponent) EmbedFS() *embed.FS {
	return &VersionDatlyResources
}

func (VersionComponent) EmbedNamespace() string {
	return VersionDatlyResourceNamespace
}
