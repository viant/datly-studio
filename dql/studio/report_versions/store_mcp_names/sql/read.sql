SELECT v.report_id, v.version_no, v.generated_dql, v.authored_dql
FROM reports r
JOIN report_versions v ON v.report_id = r.id
LEFT JOIN report_publications p ON p.report_id = r.id
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("WHERE")}
