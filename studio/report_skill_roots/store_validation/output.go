package store_validation

// Output is the generated output scaffold for skill.
type Output struct {
	Skills []*ValidationSkill `parameter:"Skills,kind=output,in=view,dataType=[]*ValidationSkill" view:"skill,type=ValidationSkill,selectorNoLimit=true" sql:"uri=studio_report_skill_roots_store_validation_skill:sql/read.sql"`
}
