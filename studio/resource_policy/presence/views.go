package presence

// PolicyHead is generated canonical view metadata for head.
type PolicyHead struct {
	ResourceId string `sqlx:"resource_id,required=true,primaryKey=true"`
}
