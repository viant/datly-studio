SELECT e.event_id, e.report_id, e.owner_id, e.operation, e.version_no,
       e.generation_no, e.status, e.requested_by, e.reason, e.failure_code,
       e.failure_message, e.occurred_at
FROM report_publication_events e
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(3, "AND")).Build("WHERE")}
