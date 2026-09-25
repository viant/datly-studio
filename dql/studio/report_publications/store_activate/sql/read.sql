SELECT p.report_id, p.active_version_no, p.desired_version_no,
       p.desired_generation, p.active_generation,
       p.publication_status, p.activated_at, p.failure_json
FROM report_publications p
