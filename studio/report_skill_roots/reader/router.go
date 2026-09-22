package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for skill.
type SkillComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"skill,path=/v1/studio/reports/{reportId}/versions/{versionNo}/skills,method=GET,connector=studio,view=skill\" routeName:\"skill\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.report_skill_roots.read\\\",\\\"description\\\":\\\"Read report skill roots\\\"}]\" caseFormat:\"lc\""
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
