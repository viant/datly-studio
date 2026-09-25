SELECT s.report_id, s.version_no, s.skill_id, s.folder_id, s.skill_root, s.ordinal
FROM report_skill_roots s
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
ORDER BY s.ordinal, s.skill_id
