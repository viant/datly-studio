SELECT p.report_id, p.active_version_no, p.desired_version_no,
       p.desired_generation, p.active_generation, p.publication_status,
       p.runtime_revision, p.spec_hash, p.published_by,
       p.published_at, p.activated_at, p.failure_json
FROM report_publications p
