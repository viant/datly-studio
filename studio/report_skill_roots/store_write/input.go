package store_write

// Input is the generated input scaffold for skill.
type Input struct {
	Skills                   []*StoredSkill           `parameter:"Skills,kind=body,in=data,dataType=[]*StoredSkill" view:"skill,type=StoredSkill,entityHooks=SkillStoreRules,table=report_skill_roots" sql:"uri=studio_report_skill_roots_store_write_skill:sql/read.sql"`
	SkillKeys                []SkillKeysRow           `parameter:"SkillKeys,kind=param,in=Skills,cardinality=Many" codec:"structql,'uri=studio_report_skill_roots_store_write_skill:sql/skill_keys.sql'"`
	CurrentSkill             []*CurrentSkillView      `parameter:"CurrentSkill,kind=view,in=CurrentSkill,cardinality=Many" view:"CurrentSkill,table=report_skill_roots" sql:"uri=studio_report_skill_roots_store_write_skill:sql/current_skill.sql"`
	_skillHandlerReadIndexes *SkillHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                      *InputHas                `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Skills       bool
	SkillKeys    bool
	CurrentSkill bool
}
