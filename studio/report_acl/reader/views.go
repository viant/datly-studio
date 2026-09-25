package reader

// ReportACL is generated canonical view metadata for acl.
type ReportACL struct {
	CanView     bool    `sqlx:"can_view"`
	CanRun      bool    `sqlx:"can_run"`
	CanEdit     bool    `sqlx:"can_edit"`
	CanPublish  bool    `sqlx:"can_publish"`
	CanUseDql   bool    `sqlx:"can_use_dql"`
	ReportId    *string `sqlx:"report_id"`
	SubjectType *string `sqlx:"subject_type"`
	SubjectId   *string `sqlx:"subject_id"`
	Etag        *int    `sqlx:"etag"`
}
