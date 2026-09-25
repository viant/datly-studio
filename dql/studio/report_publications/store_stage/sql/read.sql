SELECT p.report_id, p.desired_version_no, p.desired_generation,
       p.publication_status, p.runtime_revision, p.spec_hash,
       p.published_by, p.published_at, p.failure_json
FROM report_publications p
