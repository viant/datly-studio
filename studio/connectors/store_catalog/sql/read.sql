SELECT connector."name", connector."driver", connector."dsn_template", connector."secret_ref", connector."description", connector."owner_id", connector."status", connector."options_json", connector."last_test_status", connector."last_test_error_code", connector."last_tested_at", connector."etag", connector."created_at", connector."updated_at" FROM  (SELECT c.name, c.driver, c.dsn_template, c.secret_ref, c.description,
       c.owner_id, c.status, c.options_json, c.last_test_status,
       c.last_test_error_code, c.last_tested_at, c.etag, c.created_at, c.updated_at
FROM connectors c
WHERE c.deleted_at IS NULL
  AND ($Name = '' OR c.name = $Name)
  AND ($Query = '' OR LOWER(c.name) LIKE $Query OR LOWER(c.description) LIKE $Query
       OR LOWER(c.driver) LIKE $Query OR LOWER(c.owner_id) LIKE $Query)
  AND ($Status = '' OR c.status = $Status)
  AND ($OwnerId = '' OR c.owner_id = $OwnerId)
  AND ($Driver = '' OR c.driver = $Driver)
  AND ($Scoped = FALSE OR c.owner_id = $Subject OR EXISTS (
      SELECT 1 FROM reports studio_sdk_report
      JOIN report_acl studio_sdk_acl ON studio_sdk_acl.report_id = studio_sdk_report.id
      WHERE studio_sdk_report.default_connector_name = c.name
        AND studio_sdk_report.deleted_at IS NULL
        AND studio_sdk_acl.subject_type = 'user'
        AND studio_sdk_acl.subject_id = $Subject
        AND studio_sdk_acl.can_view = TRUE
  ))
ORDER BY c.updated_at DESC, c.name ASC
LIMIT $PageLimit OFFSET $PageOffset
)  connector WHERE 1 = 1 ORDER BY connector.updated_at DESC, connector.name ASC