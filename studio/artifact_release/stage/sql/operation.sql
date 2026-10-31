SELECT * FROM (SELECT o.* FROM artifact_release_operations o
${predicate.Builder().CombineAnd($predicate.FilterGroup(1,"AND")).Build("WHERE")}
) operation