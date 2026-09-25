SELECT publication."report_id", publication."active_version_no", publication."desired_version_no", publication."desired_generation", publication."active_generation", publication."publication_status", publication."runtime_revision", publication."spec_hash", publication."published_by", publication."published_at", publication."activated_at" FROM  (SELECT p.report_id, p.active_version_no, p.desired_version_no,
       p.desired_generation, p.active_generation, p.publication_status,
       p.runtime_revision, p.spec_hash, p.published_by,
       p.published_at, p.activated_at
FROM report_publications p
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(2, "AND")
).Build("WHERE")}
)  publication WHERE 1 = 1