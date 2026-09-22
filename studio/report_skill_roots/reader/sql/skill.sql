SELECT skill_root."report_id", skill_root."version_no", skill_root."skill_id", skill_root."folder_id", skill_root."skill_root", skill_root."ordinal" FROM  (SELECT s.report_id, s.version_no, s.skill_id, s.folder_id, s.skill_root, s.ordinal
FROM report_skill_roots s
WHERE s.report_id = $ReportId AND s.version_no = $VersionNo
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(2, "AND"), $predicate.FilterGroup(3, "AND")).Build("AND")}
)  skill_root WHERE 1 = 1