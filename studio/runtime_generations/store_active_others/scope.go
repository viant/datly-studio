package store_active_others

// ExcludedRuntimeGeneration passes the server-bound current generation to
// the typed predicate. This component is never publicly routed.
func (input Input) ExcludedRuntimeGeneration() int64 { return input.ExcludeGeneration }
