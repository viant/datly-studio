INSERT INTO connectors (
    name, driver, dsn_template, description, owner_id, status,
    options_json, last_test_status, last_tested_at, etag, created_at, updated_at
) VALUES (
    'reporting', 'sqlite', 'file:reporting-unit.db',
    'Datly Studio SQLite unit-test reporting database', 'unit', 'active',
    '{"max_idle_conns":1,"max_open_conns":1}', 'passed',
    '2026-09-17 00:00:00', 1, '2026-09-17 00:00:00', '2026-09-17 00:00:00'
);

INSERT INTO namespaces (
    owner_id, name, title, description, status, etag, created_at, updated_at
) VALUES (
    'unit', 'general', 'General', 'Default unit-test namespace', 'active', 1,
    '2026-09-17 00:00:00', '2026-09-17 00:00:00'
);
