SELECT c.name, c.driver, c.dsn_template, c.secret_ref, c.description,
       c.owner_id, c.status, c.options_json, c.last_test_status,
       c.last_test_error_code, c.last_tested_at, c.etag, c.created_at,
       c.updated_at, c.is_live
FROM (
    SELECT connector.name, connector.driver, connector.dsn_template,
           connector.secret_ref, connector.description, connector.owner_id,
           connector.status, connector.options_json, connector.last_test_status,
           connector.last_test_error_code, connector.last_tested_at,
           connector.etag, connector.created_at, connector.updated_at,
           CASE WHEN connector.deleted_at IS NULL THEN TRUE ELSE FALSE END AS is_live
    FROM connectors connector
) c
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND")).Build("WHERE")}
