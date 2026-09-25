SELECT definition."report_id", definition."version_no", definition."component_scope", definition."component_name", definition."default_connector_name", definition."driver", definition."dsn_template", definition."secret_ref", definition."generated_dql", definition."authored_dql" FROM  (SELECT r.id AS report_id, v.version_no, r.component_scope, r.component_name,
       r.default_connector_name, c.driver, c.dsn_template,
       COALESCE(c.secret_ref, '') AS secret_ref,
       COALESCE(v.generated_dql, '') AS generated_dql,
       COALESCE(v.authored_dql, '') AS authored_dql
FROM report_publications p
JOIN reports r ON r.id = p.report_id
JOIN report_versions v ON v.report_id = p.report_id AND v.version_no =
  CASE WHEN p.publication_status = 'pending'
             AND p.desired_generation = $CandidateGeneration
             AND p.desired_version_no IS NOT NULL
       THEN p.desired_version_no ELSE p.active_version_no END
JOIN connectors c ON c.name = r.default_connector_name
WHERE (
  (p.publication_status = 'pending'
   AND p.desired_generation = $CandidateGeneration
   AND p.desired_version_no IS NOT NULL)
  OR (p.active_generation IS NOT NULL
      AND NOT (p.publication_status = 'unpublishing'
               AND p.desired_generation = $CandidateGeneration))
)
  AND r.deleted_at IS NULL AND c.deleted_at IS NULL
)  definition WHERE 1 = 1