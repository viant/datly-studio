SELECT settings.* FROM  (SELECT s.* FROM namespace_settings s
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
)  settings