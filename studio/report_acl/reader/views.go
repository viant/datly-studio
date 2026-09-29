package reader

// ReportACL is generated canonical view metadata for acl.
type ReportACL struct {
	ReportId    *string `sqlx:"report_id,refTable=components,refColumn=id,required=true,primaryKey=true"`
	SubjectType *string `sqlx:"subject_type,required=true,primaryKey=true"`
	SubjectId   *string `sqlx:"subject_id,required=true,primaryKey=true"`
	Etag        *int    `sqlx:"etag,required=true"`
	CanView     bool    `sqlx:"can_view,required=true"`
	CanRun      bool    `sqlx:"can_run,required=true"`
	CanEdit     bool    `sqlx:"can_edit,required=true"`
	CanPublish  bool    `sqlx:"can_publish,required=true"`
	CanUseDql   bool    `sqlx:"can_use_dql,required=true"`
}
