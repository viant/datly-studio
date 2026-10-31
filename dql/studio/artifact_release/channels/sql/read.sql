SELECT r.* FROM (artifact_release_channels) r
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}
