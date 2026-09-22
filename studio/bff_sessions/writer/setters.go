package writer

func (entity *SessionRevocation) GetSessionIdHash() string {
	return entity.SessionIdHash
}
func (entity *SessionRevocation) SetSessionIdHash(value string) {
	entity.SessionIdHash = value
	if entity.Has == nil {
		entity.Has = &SessionRevocationHas{}
	}
	entity.Has.SessionIdHash = true
}
func (entity *SessionRevocation) GetSubjectId() string {
	return entity.SubjectId
}
func (entity *SessionRevocation) SetSubjectId(value string) {
	entity.SubjectId = value
	if entity.Has == nil {
		entity.Has = &SessionRevocationHas{}
	}
	entity.Has.SubjectId = true
}
func (entity *SessionRevocation) GetExpiresAtUnix() *int {
	return entity.ExpiresAtUnix
}
func (entity *SessionRevocation) SetExpiresAtUnix(value *int) {
	entity.ExpiresAtUnix = value
	if entity.Has == nil {
		entity.Has = &SessionRevocationHas{}
	}
	entity.Has.ExpiresAtUnix = true
}
