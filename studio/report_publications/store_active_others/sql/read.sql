SELECT publication."report_id", publication."desired_generation", publication."active_generation", publication."publication_status" FROM  (SELECT p.report_id, p.desired_generation, p.active_generation,
       p.publication_status
FROM component_publications p
WHERE 1=1
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
AND ($NamespaceId = '' OR EXISTS (
 SELECT 1 FROM components r JOIN namespaces n ON n.owner_id=r.owner_id AND n.name=r.namespace
 WHERE r.id=p.report_id AND n.namespace_id=$NamespaceId
))
)  publication WHERE 1 = 1