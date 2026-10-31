-- Apply explicitly to a version-26 store after backup.
BEGIN IMMEDIATE;
CREATE TABLE namespace_settings (
    owner_id        VARCHAR(128) NOT NULL,
    namespace       VARCHAR(200) NOT NULL,
    setting_key     VARCHAR(128) NOT NULL,
    value_json      JSON NOT NULL,
    updated_at      DATETIME NOT NULL,
    updated_by      VARCHAR(128) NOT NULL,
    PRIMARY KEY (owner_id, namespace, setting_key),
    CONSTRAINT fk_namespace_settings_namespace
        FOREIGN KEY (owner_id, namespace)
        REFERENCES namespaces(owner_id, name)
);
ALTER TABLE component_versions ADD COLUMN resource_check_enabled BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE schema_version SET version=27;
COMMIT;
