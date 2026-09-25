SELECT g.generation_no, g.status, g.report_count, g.activated_at,
       g.diagnostics_json
FROM runtime_generations g
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND")
).Build("WHERE")}
