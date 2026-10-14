SELECT skill.* FROM  (SELECT s.* FROM namespace_skills s
WHERE s.deleted_at IS NULL
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND"),$predicate.FilterGroup(1,"OR")).Build("AND")}
ORDER BY s.name
)  skill