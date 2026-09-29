SELECT p.resource_id FROM resource_policy_heads p
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
