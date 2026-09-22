package reader

// AuthContext is generated canonical view metadata for context.
type AuthContext struct {
	Subject string `sqlx:"subject"`
	Username string `sqlx:"username"`
	Email string `sqlx:"email"`
	UserId int `sqlx:"user_id"`
}
