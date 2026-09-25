package store_snapshot

// Input is the generated input scaffold for skill.
type Input struct {
	ReportId  string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true" predicate:"equal,s,report_id"`
	VersionNo int       `parameter:"VersionNo,kind=query,in=versionNo,dataType=int,required=true" predicate:"equal,s,version_no"`
	SkillId   string    `parameter:"SkillId,kind=query,in=skillId,dataType=string,required=false" predicate:"equal,s,skill_id"`
	Has       *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId  bool
	VersionNo bool
	SkillId   bool
}
