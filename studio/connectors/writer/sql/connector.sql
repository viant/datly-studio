SELECT connector."name", connector."driver", connector."dsn_template", connector."secret_ref", connector."description", connector."owner_id", connector."status", connector."options_json", connector."last_test_status", connector."last_test_error_code", connector."last_tested_at", connector."etag", connector."created_at", connector."updated_at", connector."deleted_at", connector."should_delete" FROM  (
    SELECT c.*, '' AS should_delete
FROM connectors c

)  connector WHERE 1 = 1