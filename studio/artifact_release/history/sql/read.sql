SELECT records.* FROM  (SELECT r.* FROM (artifact_release_operations) r
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}
)  records WHERE 1 = 1