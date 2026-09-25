package store_snapshot

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for skill.
type SkillComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"skill,path=/_studio/resource-snapshot/skills,method=GET,connector=studio,view=skill\" routeName:\"skill\" caseFormat:\"lc\""
}

// SkillDatlyType keeps the public component type linked for blank-import discovery.
func SkillDatlyType() reflect.Type { return reflect.TypeOf((*SkillComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var SkillDatly = new(SkillComponent)
var SkillDatlyLinkedType = SkillDatlyType()

func (SkillComponent) EmbedFS() *embed.FS {
	return &SkillDatlyResources
}

func (SkillComponent) EmbedNamespace() string {
	return SkillDatlyResourceNamespace
}
