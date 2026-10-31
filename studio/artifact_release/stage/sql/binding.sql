SELECT * FROM (SELECT b.* FROM artifact_release_bundle_bindings b
${predicate.Builder().CombineAnd($predicate.FilterGroup(2,"AND")).Build("WHERE")}
) binding