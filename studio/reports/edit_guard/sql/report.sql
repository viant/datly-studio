SELECT report."id" FROM  (SELECT report.id
FROM components report
WHERE report.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
)  report WHERE 1 = 1