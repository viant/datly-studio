package store_write

// Output is the generated output scaffold for skill.
type Output struct {
	Data []*StoredSkill `parameter:"Data,kind=output,in=body,dataType=[]*StoredSkill"`
}
