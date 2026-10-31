SELECT u.* FROM (artifact_release_bundles) u
${predicate.Builder().CombineAnd($predicate.FilterGroup(3,"AND")).Build("WHERE")}
