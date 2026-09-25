SELECT c.name, c.driver,
       CASE WHEN c.secret_ref IS NOT NULL AND TRIM(c.secret_ref) <> '' THEN 1 ELSE 0 END AS secret_configured,
       c.description, c.owner_id, c.status, c.options_json, c.last_test_status,
       c.last_test_error_code, c.last_tested_at, c.etag,
       c.created_at, c.updated_at
FROM connectors c
WHERE c.deleted_at IS NULL
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "OR"), $predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(2, "AND")).Build("AND")}
