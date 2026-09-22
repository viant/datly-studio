SELECT s.report_id, s.version_no, s.skill_id, s.folder_id, s.skill_root, s.ordinal
FROM report_skill_roots s
WHERE s.report_id = $ReportId AND s.version_no = $VersionNo
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(2, "AND"), $predicate.FilterGroup(3, "AND")).Build("AND")}
