SELECT r."report_id", r."desired_generation", r."active_generation", r."publication_status" FROM (SELECT publication."report_id", publication."desired_generation", publication."active_generation", publication."publication_status" FROM  (SELECT p.report_id, p.desired_generation, p.active_generation,
       p.publication_status
FROM report_publications p
)  publication) r WHERE $criteria.CompositeIn("r", $PublicationKeys)