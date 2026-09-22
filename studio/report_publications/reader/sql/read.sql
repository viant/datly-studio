SELECT publication."report_id", publication."active_version_no", publication."desired_version_no", publication."desired_generation", publication."active_generation", publication."publication_status", publication."runtime_revision", publication."spec_hash", publication."published_by", publication."published_at", publication."activated_at", publication."failure_json" FROM  (SELECT p.report_id, p.active_version_no, p.desired_version_no, p.desired_generation, p.active_generation,
       p.publication_status, p.runtime_revision, p.spec_hash, p.published_by,
       p.published_at, p.activated_at, p.failure_json
FROM report_publications p
WHERE p.report_id = $ReportId
${predicate.Builder().CombineAnd($predicate.FilterGroup(3, "AND")).Build("AND")}
)  publication WHERE 1 = 1