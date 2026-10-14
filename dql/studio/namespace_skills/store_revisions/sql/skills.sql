SELECT r.* FROM namespace_skill_revisions r
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}
ORDER BY r.revision DESC
