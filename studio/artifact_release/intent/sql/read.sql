SELECT intent.* FROM  (SELECT i.* FROM (artifact_release_operations) i
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}
)  intent WHERE 1 = 1