SELECT lock_component.id
FROM components lock_component
WHERE $criteria.In("lock_component.id", $ComponentIds)
ORDER BY lock_component.id
#if($ComponentIds) ${View.ForUpdate()} #end
