SELECT p.report_id, p.desired_generation, p.active_generation,
       p.publication_status
FROM report_publications p
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("WHERE")}
