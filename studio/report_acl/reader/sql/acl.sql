SELECT report_acl."report_id", report_acl."subject_type", report_acl."subject_id", report_acl."can_view", report_acl."can_run", report_acl."can_edit", report_acl."can_publish", report_acl."can_use_dql", report_acl."etag" FROM  (SELECT a.report_id, a.subject_type, a.subject_id, a.can_view, a.can_run,
       a.can_edit, a.can_publish, a.can_use_dql, a.etag
FROM report_acl a
WHERE a.report_id = $ReportId
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(2, "AND"), $predicate.FilterGroup(3, "AND")).Build("AND")}
)  report_acl WHERE 1 = 1