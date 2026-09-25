SELECT p.report_id, p.desired_generation, p.publication_status,
       FALSE AS should_delete
FROM report_publications p
