SELECT r.id AS report_id, r.title, r.namespace, r.owner_id,
       r.default_connector_name, r.component_name,
       p.active_version_no AS version_no, p.publication_status,
       p.runtime_revision, p.activated_at
FROM report_publications p
JOIN components r ON r.id = p.report_id
WHERE ($NamespaceId = '' OR EXISTS(SELECT 1 FROM namespaces n WHERE n.namespace_id=$NamespaceId AND n.owner_id=r.owner_id AND n.name=r.namespace))
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
