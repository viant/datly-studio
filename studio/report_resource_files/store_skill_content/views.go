package store_skill_content

// SkillContent is generated canonical view metadata for file.
type SkillContent struct {
	ReportId     string `sqlx:"report_id"`
	VersionNo    int    `sqlx:"version_no"`
	Namespace    string `sqlx:"namespace"`
	ResourcePath string `sqlx:"resource_path"`
	Content      []byte `sqlx:"content"`
}
