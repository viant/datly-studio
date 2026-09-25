package store_readers

import (
	time "time"
)

// PublishedReader is generated canonical view metadata for reader.
type PublishedReader struct {
	ReportId             string               `sqlx:"report_id,primaryKey"`
	Title                string               `sqlx:"title"`
	Namespace            string               `sqlx:"namespace"`
	OwnerId              string               `sqlx:"owner_id"`
	DefaultConnectorName string               `sqlx:"default_connector_name"`
	ComponentName        string               `sqlx:"component_name"`
	VersionNo            int                  `sqlx:"version_no"`
	PublicationStatus    string               `sqlx:"publication_status"`
	RuntimeRevision      *string              `sqlx:"runtime_revision"`
	ActivatedAt          *time.Time           `sqlx:"activated_at"`
	Exposure             []*PublishedExposure `view:"exposure,type=PublishedExposure,table=report_mcp_exposures" on:"ReportId:reader.report_id=ReportId:exposure.report_id,VersionNo:reader.version_no=VersionNo:exposure.version_no" json:"exposure" sql:"uri=studio_runtime_generations_store_readers_reader:sql/exposures.sql"`
	Folder               []*PublishedFolder   `view:"folder,type=PublishedFolder,table=report_resource_folders" on:"ReportId:reader.report_id=ReportId:folder.report_id,VersionNo:reader.version_no=VersionNo:folder.version_no" json:"folder" sql:"uri=studio_runtime_generations_store_readers_reader:sql/folders.sql"`
	Skill                []*PublishedSkill    `view:"skill,type=PublishedSkill,table=report_skill_roots" on:"ReportId:reader.report_id=ReportId:skill.report_id,VersionNo:reader.version_no=VersionNo:skill.version_no" json:"skill" sql:"uri=studio_runtime_generations_store_readers_reader:sql/skills.sql"`
}

// PublishedExposure is generated canonical view metadata for reader.
type PublishedExposure struct {
	ReportId    string  `sqlx:"report_id,primaryKey"`
	VersionNo   int     `sqlx:"version_no,primaryKey"`
	ExposureId  string  `sqlx:"exposure_id,primaryKey"`
	Kind        string  `sqlx:"kind"`
	Name        string  `sqlx:"name"`
	RouteMethod string  `sqlx:"route_method"`
	RoutePath   string  `sqlx:"route_path"`
	Description *string `sqlx:"description"`
	MimeType    *string `sqlx:"mime_type"`
	Enabled     bool    `sqlx:"enabled"`
}

// PublishedFolder is generated canonical view metadata for reader.
type PublishedFolder struct {
	ReportId  string `sqlx:"report_id,primaryKey"`
	VersionNo int    `sqlx:"version_no,primaryKey"`
	FolderId  string `sqlx:"folder_id,primaryKey"`
	Namespace string `sqlx:"namespace"`
	RootPath  string `sqlx:"root_path"`
	UriPrefix string `sqlx:"uri_prefix"`
}

// PublishedSkill is generated canonical view metadata for reader.
type PublishedSkill struct {
	ReportId  string `sqlx:"report_id,primaryKey"`
	VersionNo int    `sqlx:"version_no,primaryKey"`
	SkillId   string `sqlx:"skill_id,primaryKey"`
	SkillRoot string `sqlx:"skill_root"`
	UriPrefix string `sqlx:"uri_prefix"`
}
