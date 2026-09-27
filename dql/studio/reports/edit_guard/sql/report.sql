SELECT report.id
FROM components report
WHERE report.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
