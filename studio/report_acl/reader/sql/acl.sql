SELECT component_acl."report_id", component_acl."subject_type", component_acl."subject_id", component_acl."can_view", component_acl."can_run", component_acl."can_edit", component_acl."can_publish", component_acl."can_use_dql", component_acl."etag" FROM  (SELECT a.report_id, a.subject_type, a.subject_id, a.can_view, a.can_run,
       a.can_edit, a.can_publish, a.can_use_dql, a.etag
FROM component_acl a
WHERE a.report_id = $ReportId
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(2, "AND"), $predicate.FilterGroup(3, "AND")).Build("AND")}
)  component_acl WHERE 1 = 1