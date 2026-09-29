SELECT report."id", report."owner_id", report."namespace" FROM  (SELECT report.id, report.owner_id, report.namespace
FROM components report
WHERE report.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
)  report WHERE 1 = 1