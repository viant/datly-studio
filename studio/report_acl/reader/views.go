package reader

// ReportACL is generated canonical view metadata for acl.
type ReportACL struct {
	ReportId    *string `sqlx:"report_id"`
	SubjectType *string `sqlx:"subject_type"`
	SubjectId   *string `sqlx:"subject_id"`
	CanView     *int    `sqlx:"can_view"`
	CanRun      *int    `sqlx:"can_run"`
	CanEdit     *int    `sqlx:"can_edit"`
	CanPublish  *int    `sqlx:"can_publish"`
	CanUseDql   *int    `sqlx:"can_use_dql"`
	Etag        *int    `sqlx:"etag"`
}
