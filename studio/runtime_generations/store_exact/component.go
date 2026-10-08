// Package store_exact is a server-only native Datly reader for one validated
// component version. It never follows the current publication alias.
package store_exact

import (
	"embed"
	"github.com/viant/xdatly"
)

const ResourceNamespace = "studio_runtime_generations_store_exact"

//go:embed sql/read.sql
var Resources embed.FS

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"definition,path=/_studio/runtime-definitions/exact,method=GET,connector=studio,view=definition" caseFormat:"lc"`
}

func (Component) EmbedFS() *embed.FS     { return &Resources }
func (Component) EmbedNamespace() string { return ResourceNamespace }

type Input struct {
	ReportID    string    `parameter:"ReportID,kind=query,in=reportId,dataType=string,required=true"`
	VersionNo   int       `parameter:"VersionNo,kind=query,in=versionNo,dataType=int,required=true"`
	NamespaceID string    `parameter:"NamespaceID,kind=query,in=namespaceId,dataType=string,required=false"`
	Has         *InputHas `setMarker:"true" json:"-" sqlx:"-"`
}
type InputHas struct{ ReportID, VersionNo, NamespaceID bool }
type Output struct {
	Definitions []*Definition `parameter:"Definitions,kind=output,in=view,dataType=[]*Definition" view:"definition,type=Definition,table=components,limit=1" sql:"uri=studio_runtime_generations_store_exact:sql/read.sql"`
}
type Definition struct {
	ReportID             string  `sqlx:"report_id"`
	VersionNo            int     `sqlx:"version_no"`
	ComponentScope       string  `sqlx:"component_scope"`
	ComponentName        string  `sqlx:"component_name"`
	DefaultConnectorName string  `sqlx:"default_connector_name"`
	Driver               string  `sqlx:"driver"`
	DSNTemplate          *string `sqlx:"dsn_template"`
	SecretRef            string  `sqlx:"secret_ref"`
	GeneratedDQL         string  `sqlx:"generated_dql"`
	AuthoredDQL          string  `sqlx:"authored_dql"`
}
