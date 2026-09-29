SELECT catalog.kind,catalog.id,catalog.name,catalog.tenant,catalog.version,
       catalog.policy_kind,catalog.policy_id,catalog.owner_name,catalog.owner_id,
       catalog.component_id,catalog.namespace_id,catalog.has_policy,catalog.policy_json,catalog.source_dql,
       CASE WHEN EXISTS (SELECT 1 FROM report_publications p WHERE p.report_id=catalog.component_id AND p.active_version_no=CAST(catalog.version AS DECIMAL) AND p.publication_status <> 'unpublished' AND p.active_generation IS NOT NULL) THEN TRUE ELSE FALSE END AS is_published
FROM (
SELECT result.kind,result.id,result.name,result.tenant,result.version,
       result.policy_kind,result.policy_id,result.owner_name,result.owner_id,
       result.component_id,result.namespace_id,result.has_policy,result.policy_json,result.source_dql
FROM (
SELECT h.resource_kind AS kind,h.resource_id AS id,
       COALESCE(c.title,CASE WHEN s.skill_root='.' THEN f.root_path ELSE s.skill_root END,h.resource_id) AS name,
       h.tenant_id AS tenant,h.resource_version AS version,
       h.resource_kind AS policy_kind,h.resource_id AS policy_id,
       COALESCE(c.title,sc.title,'') AS owner_name,COALESCE(c.owner_id,sc.owner_id,'') AS owner_id,
       COALESCE(c.id,sc.id,'') AS component_id,COALESCE(c.namespace_id,sc.namespace_id,'') AS namespace_id,TRUE AS has_policy,
       r.policies_json AS policy_json,'' AS source_dql
FROM resource_policy_heads h
JOIN resource_policy_revisions r ON r.tenant_id=h.tenant_id AND r.resource_kind=h.resource_kind
 AND r.resource_id=h.resource_id AND r.resource_version=h.resource_version AND r.revision=h.revision
LEFT JOIN components c ON h.resource_kind='component' AND c.id=h.resource_id
LEFT JOIN report_skill_roots s ON h.resource_kind='skill' AND s.skill_id=h.resource_id AND CAST(s.version_no AS CHAR)=h.resource_version
LEFT JOIN report_resource_folders f ON f.report_id=s.report_id AND f.version_no=s.version_no AND f.folder_id=s.folder_id
LEFT JOIN components sc ON sc.id=s.report_id
UNION ALL
SELECT 'component',c.id,c.title,'',CAST(v.version_no AS CHAR),'component',c.id,c.title,c.owner_id,c.id,c.namespace_id,FALSE,'{}',COALESCE(v.generated_dql,v.authored_dql,'')
FROM components c JOIN report_versions v ON v.report_id=c.id
WHERE c.deleted_at IS NULL AND NOT EXISTS (
 SELECT 1 FROM resource_policy_heads h WHERE h.resource_kind='component' AND h.resource_id=c.id AND h.resource_version=CAST(v.version_no AS CHAR))
UNION ALL
SELECT 'skill',s.skill_id,CASE WHEN s.skill_root='.' THEN f.root_path ELSE s.skill_root END,
       COALESCE(h.tenant_id,''),CAST(s.version_no AS CHAR),'component',s.report_id,c.title,c.owner_id,c.id,c.namespace_id,
       CASE WHEN h.resource_id IS NULL THEN FALSE ELSE TRUE END,COALESCE(r.policies_json,'{}'),COALESCE(v.generated_dql,v.authored_dql,'')
FROM report_skill_roots s JOIN components c ON c.id=s.report_id
JOIN report_versions v ON v.report_id=s.report_id AND v.version_no=s.version_no
JOIN report_resource_folders f ON f.report_id=s.report_id AND f.version_no=s.version_no AND f.folder_id=s.folder_id
LEFT JOIN resource_policy_heads h ON h.resource_kind='component' AND h.resource_id=s.report_id AND h.resource_version=CAST(s.version_no AS CHAR)
LEFT JOIN resource_policy_revisions r ON r.tenant_id=h.tenant_id AND r.resource_kind=h.resource_kind AND r.resource_id=h.resource_id AND r.resource_version=h.resource_version AND r.revision=h.revision
WHERE c.deleted_at IS NULL AND NOT EXISTS (
 SELECT 1 FROM resource_policy_heads p WHERE p.resource_kind='skill' AND p.resource_id=s.skill_id AND p.resource_version=CAST(s.version_no AS CHAR))
) result ORDER BY kind,name,id,tenant,version
LIMIT $Limit OFFSET $Offset
) catalog
