SELECT publication."report_id", publication."desired_generation", publication."publication_status" FROM  (SELECT p.report_id, p.desired_generation, p.publication_status
FROM report_publications p
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND")
).Build("WHERE")}
)  publication WHERE 1 = 1