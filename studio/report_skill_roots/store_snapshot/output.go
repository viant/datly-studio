package store_snapshot

// Output is the generated output scaffold for skill.
type Output struct {
	Skills []*SnapshotSkill `parameter:"Skills,kind=output,in=view,dataType=[]*SnapshotSkill" view:"skill,type=SnapshotSkill,table=report_skill_roots,selectorNoLimit=true" sql:"uri=studio_report_skill_roots_store_snapshot_skill:sql/read.sql"`
}
