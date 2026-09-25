package store_skill_content

// Output is the generated output scaffold for file.
type Output struct {
	Files []*SkillContent `parameter:"Files,kind=output,in=view,dataType=[]*SkillContent" view:"file,type=SkillContent,limit=2" sql:"uri=studio_report_resource_files_store_skill_content_file:sql/read.sql"`
}
