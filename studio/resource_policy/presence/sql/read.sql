SELECT head."resource_id" FROM  (SELECT p.resource_id FROM resource_policies p
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
)  head WHERE 1 = 1