SELECT p.resource_id FROM resource_policies p
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
