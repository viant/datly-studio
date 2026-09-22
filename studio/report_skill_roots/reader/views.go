package reader

// ReportSkillRoot is generated canonical view metadata for skill.
type ReportSkillRoot struct {
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	SkillId *string `sqlx:"skill_id"`
	FolderId *string `sqlx:"folder_id"`
	SkillRoot *string `sqlx:"skill_root"`
	Ordinal *int `sqlx:"ordinal"`
}
