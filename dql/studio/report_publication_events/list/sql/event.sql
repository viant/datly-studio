SELECT e.event_id, e.report_id, e.owner_id, e.operation, e.version_no,
       e.generation_no, e.status, e.requested_by, e.reason, e.failure_code,
       e.failure_message, e.occurred_at,
       CASE WHEN e.version_no IS NOT NULL AND NOT EXISTS (
         SELECT 1 FROM component_versions retained_version
         WHERE retained_version.report_id=e.report_id AND retained_version.version_no=e.version_no
       ) THEN TRUE ELSE FALSE END AS version_deleted
FROM component_publication_events e
WHERE e.report_id = $ReportId
  AND ($Input.Operation = '' OR e.operation = $Input.Operation)
  AND ($Input.Status = '' OR e.status = $Input.Status)
${predicate.Builder().CombineAnd($predicate.FilterGroup(3, "AND")).Build("AND")}
ORDER BY e.occurred_at DESC, e.event_id DESC
LIMIT $Input.Limit OFFSET $Input.Offset
