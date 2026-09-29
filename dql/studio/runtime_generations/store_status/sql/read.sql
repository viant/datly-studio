SELECT g.generation_no, g.status, g.report_count, g.activated_at,
       g.diagnostics_json
FROM runtime_generations g
WHERE g.namespace_id=$NamespaceId
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND")
).Build("AND")}
