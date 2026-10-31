SELECT i.* FROM (artifact_release_operations) i
${predicate.Builder().CombineAnd($predicate.FilterGroup(3,"AND")).Build("WHERE")}
