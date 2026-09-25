SELECT p.report_id, p.active_version_no, p.desired_version_no,
       p.desired_generation, p.active_generation, p.publication_status,
       p.runtime_revision, p.spec_hash, p.published_by,
       p.published_at, p.activated_at
FROM report_publications p
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(2, "AND")
).Build("WHERE")}
