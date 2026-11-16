SELECT lock_component.id,lock_component.namespace_id,lock_component.last_version_no,
 CAST(lock_component.id AS string), CAST(lock_component.namespace_id AS string), CAST(lock_component.last_version_no AS int),
 tag(lock_component.id,'sqlx:"id,primaryKey"')
FROM components lock_component
WHERE $criteria.In("lock_component.id", $ComponentIds.Values)
ORDER BY lock_component.id
#if($ComponentIds.Values) ${View.ForUpdate()} #end
