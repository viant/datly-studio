SELECT lock_component.id, lock_component.namespace_id, lock_component.last_version_no FROM components lock_component
WHERE $criteria.In("lock_component.id", $ComponentIds.Values)
ORDER BY lock_component.id
#if($ComponentIds.Values) ${View.ForUpdate()} #end