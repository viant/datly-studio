SELECT r.* FROM (artifact_release_operations) r
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}
