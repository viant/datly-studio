SELECT component_acl."report_id", component_acl."subject_type", component_acl."subject_id", component_acl."can_view", component_acl."can_run", component_acl."can_edit", component_acl."can_publish", component_acl."can_use_dql", component_acl."etag", component_acl."should_delete" FROM  (SELECT a.*, '' AS should_delete FROM component_acl a
)  component_acl WHERE 1 = 1
