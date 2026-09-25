SELECT skill."report_id", skill."version_no", skill."skill_id", skill."folder_id", skill."skill_root", skill."ordinal" FROM  (SELECT s.report_id, s.version_no, s.skill_id, s.folder_id, s.skill_root, s.ordinal
FROM report_skill_roots s
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
ORDER BY s.ordinal, s.skill_id
)  skill WHERE 1 = 1 ORDER BY skill.ordinal, skill.skill_id