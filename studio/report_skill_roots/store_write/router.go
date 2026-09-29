package store_write

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for skill.
type SkillComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"skill,path=/_studio/skill-root-store/write,method=PATCH,connector=studio,view=skill,internal=true\" routeName:\"skill\" mutation:\"patch\" caseFormat:\"lc\""
}

// SkillDatlyType returns the public component type.
func SkillDatlyType() reflect.Type { return reflect.TypeOf((*SkillComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var SkillDatly = new(SkillComponent)
var _datlyReachableSkillComponent = reflect.TypeFor[SkillComponent]()

func (SkillComponent) EmbedFS() *embed.FS {
	return &SkillDatlyResources
}

func (SkillComponent) EmbedNamespace() string {
	return SkillDatlyResourceNamespace
}
