SELECT r."report_id", r."active_version_no", r."desired_version_no", r."desired_generation", r."active_generation", r."publication_status", r."activated_at", r."failure_json" FROM (SELECT publication."report_id", publication."active_version_no", publication."desired_version_no", publication."desired_generation", publication."active_generation", publication."publication_status", publication."activated_at", publication."failure_json" FROM  (SELECT p.report_id, p.active_version_no, p.desired_version_no,
       p.desired_generation, p.active_generation,
       p.publication_status, p.activated_at, p.failure_json
FROM report_publications p
)  publication) r WHERE $criteria.CompositeIn("r", $PublicationKeys)