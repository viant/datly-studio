SELECT reader."report_id", reader."title", reader."namespace", reader."owner_id", reader."default_connector_name", reader."component_name", reader."version_no", reader."publication_status", reader."runtime_revision", reader."activated_at" FROM  (SELECT r.id AS report_id, r.title, r.namespace, r.owner_id,
       r.default_connector_name, r.component_name,
       p.active_version_no AS version_no, p.publication_status,
       p.runtime_revision, p.activated_at
FROM report_publications p
JOIN reports r ON r.id = p.report_id
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("WHERE")}
)  reader WHERE 1 = 1 ORDER BY reader.namespace, reader.title, reader.report_id