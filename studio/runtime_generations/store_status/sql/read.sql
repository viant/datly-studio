SELECT generation."generation_no", generation."status", generation."report_count", generation."activated_at", generation."diagnostics_json" FROM  (SELECT g.generation_no, g.status, g.report_count, g.activated_at,
       g.diagnostics_json
FROM runtime_generations g
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND")
).Build("WHERE")}
)  generation WHERE 1 = 1 ORDER BY generation.generation_no DESC