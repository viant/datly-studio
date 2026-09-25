SELECT event."event_id", event."report_id", event."owner_id", event."operation", event."version_no", event."generation_no", event."status", event."requested_by", event."reason", event."failure_code", event."failure_message", event."occurred_at" FROM  (SELECT e.event_id, e.report_id, e.owner_id, e.operation, e.version_no,
       e.generation_no, e.status, e.requested_by, e.reason, e.failure_code,
       e.failure_message, e.occurred_at
FROM report_publication_events e
WHERE e.report_id = $ReportId
  AND ($Input.Operation = '' OR e.operation = $Input.Operation)
  AND ($Input.Status = '' OR e.status = $Input.Status)
${predicate.Builder().CombineAnd($predicate.FilterGroup(3, "AND")).Build("AND")}
ORDER BY e.occurred_at DESC, e.event_id DESC
LIMIT $Input.Limit OFFSET $Input.Offset
)  event WHERE 1 = 1 ORDER BY event.occurred_at DESC, event.event_id DESC