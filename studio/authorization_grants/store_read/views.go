package store_read

// Grant is generated canonical view metadata for grant.
type Grant struct {
	ReportId    string `sqlx:"report_id"`
	SubjectId   string `sqlx:"subject_id"`
	SubjectType string `sqlx:"subject_type"`
	GrantSource string `sqlx:"grant_source"`
	CanView     bool   `sqlx:"can_view"`
	CanEdit     bool   `sqlx:"can_edit"`
	CanPublish  bool   `sqlx:"can_publish"`
	IsLive      bool   `sqlx:"is_live"`
}
