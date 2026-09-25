package store_readers

// RuntimeReaderScope exposes the server-bound generation and verified
// principal to the typed predicate. This component is never publicly routed.
func (input Input) RuntimeReaderScope() (generation int64, subject string, scoped bool) {
	return input.Generation, input.Subject, input.Scoped
}
