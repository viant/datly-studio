package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for skill.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Skills []*ReportSkillRoot `parameter:"Skills,kind=body,in=data,dataType=[]*ReportSkillRoot" view:"skill,type=ReportSkillRoot,entityHooks=SkillRules,table=report_skill_roots" sql:"uri=studio_report_skill_roots_writer_skill:sql/skill.sql"`
	SkillKeys []SkillKeysRow `parameter:"SkillKeys,kind=param,in=Skills,cardinality=Many" codec:"structql,'uri=studio_report_skill_roots_writer_skill:sql/skill_keys.sql'"`
	CurrentSkill []*CurrentSkillView `parameter:"CurrentSkill,kind=view,in=CurrentSkill,cardinality=Many" view:"CurrentSkill,table=report_skill_roots" sql:"uri=studio_report_skill_roots_writer_skill:sql/current_skill.sql"`
	_skillHandlerReadIndexes *SkillHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Skills bool
	SkillKeys bool
	CurrentSkill bool
}
