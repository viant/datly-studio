SELECT lock_version.report_id,lock_version.version_no
FROM component_versions lock_version
WHERE $criteria.In("lock_version.report_id", $ComponentIds)
ORDER BY lock_version.report_id,lock_version.version_no
#if($ComponentIds) ${View.ForUpdate()} #end
