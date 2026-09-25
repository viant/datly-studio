SELECT a.report_id, a.subject_type, a.subject_id, a.can_view, a.can_run,
       a.can_edit, a.can_publish, a.can_use_dql, a.etag
FROM report_acl a
WHERE a.report_id = $ReportId
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(2, "AND"), $predicate.FilterGroup(3, "AND")).Build("AND")}
