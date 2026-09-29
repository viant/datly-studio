package presence

// Output is the generated output scaffold for head.
type Output struct {
	Heads []*PolicyHead `parameter:"Heads,kind=output,in=view,dataType=[]*PolicyHead" view:"head,type=PolicyHead,table=resource_policy_heads,limit=1" sql:"uri=studio_resource_policy_presence_head:sql/read.sql"`
}
