SELECT p.report_id, p.active_version_no, p.desired_version_no,
       p.desired_generation, p.active_generation, p.publication_status,
       p.runtime_revision, p.spec_hash, p.failure_json
FROM report_publications p
