package store_validation

// ValidationSkill is generated canonical view metadata for skill.
type ValidationSkill struct {
	ReportId  string `sqlx:"report_id"`
	VersionNo int    `sqlx:"version_no"`
	SkillId   string `sqlx:"skill_id"`
	SkillRoot string `sqlx:"skill_root"`
	Namespace string `sqlx:"namespace"`
	RootPath  string `sqlx:"root_path"`
}
