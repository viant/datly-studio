package writer

// ReportSkillRoot is generated canonical view metadata for skill.
type ReportSkillRoot struct {
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_resource_folders,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_resource_folders,refColumn=version_no,required=true"`
	SkillId *string `sqlx:"skill_id,primaryKey,required=true" validate:"required"`
	FolderId *string `validate:"required" sqlx:"folder_id,refTable=report_resource_folders,refColumn=folder_id,required=true"`
	SkillRoot *string `validate:"required" sqlx:"skill_root,required=true"`
	ShouldDelete bool `sqlx:"-" writer:"delete"`
	Ordinal *int `sqlx:"ordinal,required=true"`
	Has *ReportSkillRootHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ReportSkillRootHas"`
}

type ReportSkillRootHas struct {
	ReportId bool
	VersionNo bool
	SkillId bool
	FolderId bool
	SkillRoot bool
	ShouldDelete bool
	Ordinal bool
}

// CurrentSkillView is generated canonical view metadata for skill.
type CurrentSkillView struct {
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_resource_folders,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_resource_folders,refColumn=version_no,required=true"`
	SkillId *string `sqlx:"skill_id,primaryKey,required=true" validate:"required"`
	FolderId *string `validate:"required" sqlx:"folder_id,refTable=report_resource_folders,refColumn=folder_id,required=true"`
	SkillRoot *string `validate:"required" sqlx:"skill_root,required=true"`
	Ordinal *int `sqlx:"ordinal,required=true"`
}

type SkillKeysRow struct {
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	SkillId *string `sqlx:"skill_id"`
}
