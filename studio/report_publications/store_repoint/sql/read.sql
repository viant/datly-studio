SELECT publication."report_id", publication."desired_generation", publication."active_generation", publication."publication_status" FROM  (SELECT p.report_id, p.desired_generation, p.active_generation,
       p.publication_status
FROM report_publications p
)  publication