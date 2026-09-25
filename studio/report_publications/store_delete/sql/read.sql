SELECT publication."report_id", publication."desired_generation", publication."publication_status", publication."should_delete" FROM  (SELECT p.report_id, p.desired_generation, p.publication_status,
       FALSE AS should_delete
FROM report_publications p
)  publication