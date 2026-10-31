SELECT s.* FROM (artifact_release_channels) s
${predicate.Builder().CombineAnd($predicate.FilterGroup(2,"AND")).Build("WHERE")}
