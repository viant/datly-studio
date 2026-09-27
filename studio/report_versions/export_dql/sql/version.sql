SELECT version."report_id", version."version_no", version."dql" FROM  (SELECT v.report_id, v.version_no,
       COALESCE(NULLIF(v.generated_dql, ''), NULLIF(v.authored_dql, ''),
                COALESCE(v.authored_sql, '')) AS dql
FROM report_versions v
JOIN components r ON r.id = v.report_id AND r.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("WHERE")}
)  version WHERE 1 = 1