package store_write

// StoredSkill is generated canonical view metadata for skill.
type StoredSkill struct {
	ReportId     string          `sqlx:"report_id,primaryKey"`
	VersionNo    int             `sqlx:"version_no,primaryKey"`
	SkillId      string          `sqlx:"skill_id,primaryKey"`
	FolderId     string          `sqlx:"folder_id"`
	SkillRoot    string          `sqlx:"skill_root"`
	Ordinal      int             `sqlx:"ordinal"`
	ShouldDelete bool            `sqlx:"-" writer:"delete"`
	Has          *StoredSkillHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredSkillHas"`
}

type StoredSkillHas struct {
	ReportId     bool
	VersionNo    bool
	SkillId      bool
	FolderId     bool
	SkillRoot    bool
	Ordinal      bool
	ShouldDelete bool
}

// CurrentSkillView is generated canonical view metadata for skill.
type CurrentSkillView struct {
	ReportId  string `sqlx:"report_id,primaryKey"`
	VersionNo int    `sqlx:"version_no,primaryKey"`
	SkillId   string `sqlx:"skill_id,primaryKey"`
	FolderId  string `sqlx:"folder_id"`
	SkillRoot string `sqlx:"skill_root"`
	Ordinal   int    `sqlx:"ordinal"`
}

type SkillKeysRow struct {
	ReportId  string `sqlx:"report_id"`
	VersionNo int    `sqlx:"version_no"`
	SkillId   string `sqlx:"skill_id"`
}
