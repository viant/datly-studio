SELECT * FROM (WITH candidates AS (
SELECT c.id,n.namespace_id,c.owner_id,c.audit_enabled,
 EXISTS(SELECT 1 FROM component_publications p
 JOIN component_mcp_exposures e ON e.report_id=p.report_id AND e.version_no=p.active_version_no
 LEFT JOIN component_cube_configs cube ON cube.report_id=e.report_id AND cube.version_no=e.version_no
 WHERE p.report_id=c.id AND p.active_generation IS NOT NULL AND e.enabled=TRUE AND e.kind='tool'
 AND (e.name=$ToolName OR (cube.cube_enabled=TRUE AND $ToolName=CONCAT(e.name,'Cube')) OR (cube.compose_enabled=TRUE AND $ToolName=CONCAT(e.name,'CubeCompose')))) AS matches_tool
FROM components c JOIN namespaces n ON n.owner_id=c.owner_id AND n.name=c.namespace
WHERE n.namespace_id=$NamespaceId AND c.deleted_at IS NULL
)
SELECT * FROM candidates WHERE id=$ComponentId OR matches_tool=TRUE
) component