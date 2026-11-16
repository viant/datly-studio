SELECT d.component_id, row_lock(d,'component_deletions d','d.component_id') FROM component_deletions d
WHERE $criteria.In("d.component_id", $ComponentIds)
ORDER BY d.component_id
