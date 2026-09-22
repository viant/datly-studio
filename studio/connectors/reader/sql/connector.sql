SELECT connectors."name", connectors."driver", connectors."secret_configured", connectors."description", connectors."owner_id", connectors."status", connectors."options_json", connectors."last_test_status", connectors."last_test_error_code", connectors."last_tested_at", connectors."etag", connectors."created_at", connectors."updated_at" FROM  (
    SELECT c.name, c.driver,
       CASE WHEN c.secret_ref IS NOT NULL AND TRIM(c.secret_ref) <> '' THEN 1 ELSE 0 END AS secret_configured,
       c.description, c.owner_id, c.status, c.options_json, c.last_test_status,
       c.last_test_error_code, c.last_tested_at, c.etag,
       c.created_at, c.updated_at
FROM connectors c
WHERE c.deleted_at IS NULL
  AND (c.owner_id = $Auth.Auth.Subject OR EXISTS (
      SELECT 1 FROM reports auth_report
      JOIN report_acl auth_acl ON auth_acl.report_id = auth_report.id
      WHERE auth_report.default_connector_name = c.name
        AND auth_acl.subject_type = 'user'
        AND auth_acl.subject_id = $Auth.Auth.Subject
        AND auth_acl.can_view = TRUE
  ))
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "OR"), $predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(2, "AND")).Build("AND")}

)  connectors WHERE 1 = 1