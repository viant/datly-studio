SELECT * FROM  (SELECT r.* FROM (artifact_release_bundles) r
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}
)  bundle WHERE 1 = 1