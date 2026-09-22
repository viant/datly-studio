package reader

// Output is the generated output scaffold for skill.
type Output struct {
	Skills []*ReportSkillRoot `parameter:"Skills,kind=output,in=view,dataType=[]*ReportSkillRoot" view:"skill,type=ReportSkillRoot,table=report_skill_roots" sql:"uri=studio_report_skill_roots_reader_skill:sql/skill.sql"`
}
