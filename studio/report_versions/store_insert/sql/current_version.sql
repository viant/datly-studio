SELECT r.id, r.namespace_id, r.last_version_no FROM (SELECT * FROM  (SELECT c.id, c.namespace_id, c.last_version_no FROM components c
)  allocation) r WHERE $criteria.CompositeIn("r", $VersionKeys)