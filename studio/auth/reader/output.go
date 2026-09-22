package reader

// Output is the generated output scaffold for context.
type Output struct {
	Auth *AuthContext `parameter:"Auth,kind=output,in=view,dataType=*AuthContext" view:"context,type=AuthContext,limit=1" sql:"uri=studio_auth_reader_context:sql/context.sql"`
}
