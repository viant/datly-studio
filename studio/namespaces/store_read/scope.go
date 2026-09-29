package store_read

// NamespaceDirectoryScope is a server-only directory selection.
func (input Input) NamespaceDirectoryScope() (string, bool) { return input.Subject, input.Scoped }
