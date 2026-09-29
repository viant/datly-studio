package catalog

import (
	"embed"
	"github.com/viant/xdatly"
	"reflect"
)

type Input struct {
	Limit  int `parameter:"Limit,kind=query,in=limit,dataType=int"`
	Offset int `parameter:"Offset,kind=query,in=offset,dataType=int"`
}
type Entry struct {
	NamespaceID string `sqlx:"namespace_id"`
	Published   bool   `sqlx:"is_published"`
	OwnerID     string `sqlx:"owner_id"`
	ComponentID string `sqlx:"component_id"`
	HasPolicy   bool   `sqlx:"has_policy"`
	PolicyJSON  string `sqlx:"policy_json"`
	SourceDQL   string `sqlx:"source_dql"`
	Kind        string `sqlx:"kind"`
	ID          string `sqlx:"id"`
	Name        string `sqlx:"name"`
	Tenant      string `sqlx:"tenant"`
	Version     string `sqlx:"version"`
	PolicyKind  string `sqlx:"policy_kind"`
	PolicyID    string `sqlx:"policy_id"`
	OwnerName   string `sqlx:"owner_name"`
}
type Output struct {
	Items []*Entry `parameter:"Items,kind=output,in=view,dataType=[]*Entry" view:"catalog,type=Entry,selectorNoLimit=true" sql:"uri=studio_policy_catalog:sql/catalog.sql"`
}
type Component struct {
	Contract xdatly.Component[Input, Output] `component:"catalog,path=/_studio/resource-policy-store/catalog,method=GET,connector=studio,view=catalog,internal=true" caseFormat:"lc"`
}

var CatalogDatly = new(Component)
var CatalogDatlyLinkedType = reflect.TypeFor[Component]()

const Namespace = "studio_policy_catalog"

//go:embed sql/catalog.sql
var Resources embed.FS

func (Component) EmbedFS() *embed.FS     { return &Resources }
func (Component) EmbedNamespace() string { return Namespace }
