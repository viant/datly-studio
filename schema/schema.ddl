CREATE TABLE resource_policy_heads (
    tenant_id VARCHAR(128) NOT NULL,
    resource_kind VARCHAR(64) NOT NULL,
    resource_id VARCHAR(200) NOT NULL,
    resource_version VARCHAR(64) NOT NULL,
    revision BIGINT NOT NULL,
    PRIMARY KEY (tenant_id, resource_kind, resource_id, resource_version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE resource_policy_revisions (
    tenant_id VARCHAR(128) NOT NULL,
    resource_kind VARCHAR(64) NOT NULL,
    resource_id VARCHAR(200) NOT NULL,
    resource_version VARCHAR(64) NOT NULL,
    revision BIGINT NOT NULL,
    policies_json JSON NOT NULL,
    actor_id VARCHAR(128) NOT NULL,
    occurred_at DATETIME(6) NOT NULL,
    PRIMARY KEY (tenant_id, resource_kind, resource_id, resource_version, revision)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE connectors (
    name                    VARCHAR(128) NOT NULL,
    driver                  VARCHAR(128) NOT NULL,
    dsn_template            TEXT NULL,
    secret_ref              VARCHAR(500) NULL,
    description             TEXT NULL,
    owner_id                VARCHAR(128) NOT NULL,
    status                  VARCHAR(32) NOT NULL,
    options_json            JSON NULL,
    last_test_status        VARCHAR(32) NULL,
    last_test_error_code    VARCHAR(128) NULL,
    last_tested_at          DATETIME(6) NULL,
    etag                    BIGINT NOT NULL DEFAULT 1,
    created_at              DATETIME(6) NOT NULL,
    updated_at              DATETIME(6) NOT NULL,
    deleted_at              DATETIME(6) NULL,
    PRIMARY KEY (name),
    CONSTRAINT chk_connectors_status
        CHECK (status IN ('draft', 'active', 'disabled', 'deleted')),
    CONSTRAINT chk_connectors_test_status
        CHECK (last_test_status IS NULL OR last_test_status IN ('passed', 'failed'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_connectors_status_updated
    ON connectors(status, updated_at DESC);

CREATE INDEX idx_connectors_owner_updated
    ON connectors(owner_id, updated_at DESC);

CREATE TABLE namespaces (
    owner_id                VARCHAR(128) NOT NULL,
    name                    VARCHAR(200) NOT NULL,
    title                   VARCHAR(300) NOT NULL,
    description             TEXT NULL,
    status                  VARCHAR(32) NOT NULL,
    etag                    BIGINT NOT NULL DEFAULT 1,
    created_at              DATETIME(6) NOT NULL,
    updated_at              DATETIME(6) NOT NULL,
    deleted_at              DATETIME(6) NULL,
    PRIMARY KEY (owner_id, name),
    CONSTRAINT chk_namespaces_status
        CHECK (status IN ('active', 'archived'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_namespaces_owner_status_updated
    ON namespaces(owner_id, status, updated_at DESC);

CREATE TABLE authorization_predicates (
    name                    VARCHAR(200) NOT NULL,
    title                   VARCHAR(300) NOT NULL,
    description             TEXT NULL,
    package_path            VARCHAR(1000) NOT NULL,
    type_name               VARCHAR(300) NOT NULL,
    sql_scope_json          TEXT NULL,
    owner_id                VARCHAR(128) NOT NULL,
    status                  VARCHAR(32) NOT NULL,
    etag                    BIGINT NOT NULL DEFAULT 1,
    created_at              DATETIME(6) NOT NULL,
    updated_at              DATETIME(6) NOT NULL,
    deleted_at              DATETIME(6) NULL,
    PRIMARY KEY (name),
    UNIQUE KEY uq_authorization_predicate_type (package_path, type_name),
    CONSTRAINT chk_authorization_predicates_status
        CHECK (status IN ('active', 'disabled'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_authorization_predicates_status_updated
    ON authorization_predicates(status, updated_at DESC);

CREATE TABLE reports (
    id                      VARCHAR(64) NOT NULL,
    namespace               VARCHAR(200) NOT NULL DEFAULT 'general',
    slug                    VARCHAR(200) NOT NULL,
    title                   VARCHAR(300) NOT NULL,
    description             TEXT NULL,
    owner_id                VARCHAR(128) NOT NULL,
    status                  VARCHAR(32) NOT NULL,
    default_connector_name  VARCHAR(128) NOT NULL,
    component_scope         VARCHAR(500) NOT NULL,
    component_name          VARCHAR(200) NOT NULL,
    current_draft_version   INT NULL,
    etag                    BIGINT NOT NULL DEFAULT 1,
    created_at              DATETIME(6) NOT NULL,
    updated_at              DATETIME(6) NOT NULL,
    deleted_at              DATETIME(6) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_reports_slug (slug),
    UNIQUE KEY uq_reports_component (component_scope, component_name),
    CONSTRAINT fk_reports_connector
        FOREIGN KEY (default_connector_name) REFERENCES connectors(name),
    CONSTRAINT fk_reports_namespace
        FOREIGN KEY (owner_id, namespace) REFERENCES namespaces(owner_id, name),
    CONSTRAINT chk_reports_status
        CHECK (status IN ('draft', 'active', 'disabled', 'archived'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_reports_owner_updated
    ON reports(owner_id, updated_at DESC);

CREATE INDEX idx_reports_owner_namespace_updated
    ON reports(owner_id, namespace, updated_at DESC);

CREATE INDEX idx_reports_status_updated
    ON reports(status, updated_at DESC);

CREATE TABLE report_versions (
    report_id                   VARCHAR(64) NOT NULL,
    version_no                  INT NOT NULL,
    state                       VARCHAR(32) NOT NULL,
    authoring_mode              VARCHAR(32) NOT NULL,
    authored_sql                LONGTEXT NULL,
    authored_dql                LONGTEXT NULL,
    component_spec_json         JSON NOT NULL,
    spec_format_version         VARCHAR(32) NOT NULL,
    spec_hash                   CHAR(64) NOT NULL,
    generated_dql               LONGTEXT NULL,
    dql_export_limits_json      JSON NULL,
    type_manifest_json          JSON NOT NULL,
    resource_manifest_json      JSON NULL,
    component_descriptor_json   JSON NULL,
    compile_status              VARCHAR(32) NOT NULL,
    compile_diagnostics_json    JSON NULL,
    datly_version               VARCHAR(128) NOT NULL,
    compiler_version            VARCHAR(128) NOT NULL,
    source_revision             BIGINT NOT NULL DEFAULT 1,
    notes                       TEXT NULL,
    created_by                  VARCHAR(128) NOT NULL,
    created_at                  DATETIME(6) NOT NULL,
    validated_at                DATETIME(6) NULL,
    published_at                DATETIME(6) NULL,
    PRIMARY KEY (report_id, version_no),
    UNIQUE KEY uq_report_versions_spec_hash (report_id, spec_hash),
    CONSTRAINT fk_report_versions_report
        FOREIGN KEY (report_id) REFERENCES reports(id) ON DELETE CASCADE,
    CONSTRAINT chk_report_versions_state
        CHECK (state IN ('draft', 'validated', 'published', 'superseded', 'failed')),
    CONSTRAINT chk_report_versions_authoring_mode
        CHECK (authoring_mode IN ('sql', 'dql', 'structured')),
    CONSTRAINT chk_report_versions_compile_status
        CHECK (compile_status IN ('pending', 'valid', 'invalid', 'error'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_report_versions_created
    ON report_versions(report_id, created_at DESC);

CREATE TABLE report_views (
    report_id               VARCHAR(64) NOT NULL,
    version_no              INT NOT NULL,
    view_id                 CHAR(64) NOT NULL,
    view_identity           TEXT NOT NULL,
    parent_view_id          CHAR(64) NULL,
    relation_name           VARCHAR(200) NULL,
    name                    VARCHAR(200) NOT NULL,
    namespace               VARCHAR(200) NULL,
    role                    VARCHAR(32) NOT NULL,
    cardinality             VARCHAR(32) NULL,
    connector_name          VARCHAR(128) NULL,
    source_kind             VARCHAR(32) NOT NULL,
    source_sql              LONGTEXT NULL,
    source_table            VARCHAR(500) NULL,
    metadata_json           JSON NULL,
    PRIMARY KEY (report_id, version_no, view_id),
    CONSTRAINT fk_report_views_version
        FOREIGN KEY (report_id, version_no)
        REFERENCES report_versions(report_id, version_no) ON DELETE CASCADE,
    CONSTRAINT chk_report_views_role
        CHECK (role IN ('root', 'independent', 'relation')),
    CONSTRAINT chk_report_views_source_kind
        CHECK (source_kind IN ('sql', 'table', 'resource', 'derived', 'virtual'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_report_views_parent
    ON report_views(report_id, version_no, parent_view_id);

CREATE TABLE report_fields (
    report_id               VARCHAR(64) NOT NULL,
    version_no              INT NOT NULL,
    view_id                 CHAR(64) NOT NULL,
    field_name              VARCHAR(200) NOT NULL,
    source_column           VARCHAR(300) NULL,
    expression              TEXT NULL,
    database_type           VARCHAR(128) NULL,
    go_type                 VARCHAR(500) NULL,
    nullable                BOOLEAN NOT NULL DEFAULT FALSE,
    ordinal                 INT NOT NULL,
    filterable              BOOLEAN NOT NULL DEFAULT FALSE,
    orderable               BOOLEAN NOT NULL DEFAULT FALSE,
    groupable               BOOLEAN NOT NULL DEFAULT FALSE,
    measurable              BOOLEAN NOT NULL DEFAULT FALSE,
    metadata_json           JSON NULL,
    PRIMARY KEY (report_id, version_no, view_id, field_name),
    CONSTRAINT fk_report_fields_view
        FOREIGN KEY (report_id, version_no, view_id)
        REFERENCES report_views(report_id, version_no, view_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE report_parameters (
    report_id               VARCHAR(64) NOT NULL,
    version_no              INT NOT NULL,
    parameter_id            CHAR(64) NOT NULL,
    parameter_identity      TEXT NOT NULL,
    name                    VARCHAR(200) NOT NULL,
    source_kind             VARCHAR(64) NOT NULL,
    source_name             VARCHAR(300) NULL,
    type_expr               VARCHAR(500) NULL,
    required                BOOLEAN NULL,
    emit_output             BOOLEAN NOT NULL DEFAULT FALSE,
    query_selector_json     JSON NULL,
    codec_json              JSON NULL,
    activation_json         JSON NULL,
    metadata_json           JSON NULL,
    ordinal                 INT NOT NULL,
    PRIMARY KEY (report_id, version_no, parameter_id),
    CONSTRAINT fk_report_parameters_version
        FOREIGN KEY (report_id, version_no)
        REFERENCES report_versions(report_id, version_no) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_report_parameters_name
    ON report_parameters(report_id, version_no, name);

CREATE TABLE report_predicates (
    report_id               VARCHAR(64) NOT NULL,
    version_no              INT NOT NULL,
    parameter_id            CHAR(64) NOT NULL,
    predicate_index         INT NOT NULL,
    predicate_group         INT NOT NULL DEFAULT 0,
    name                    VARCHAR(200) NOT NULL,
    args_json               JSON NULL,
    apply_when_absent       BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (report_id, version_no, parameter_id, predicate_index),
    CONSTRAINT fk_report_predicates_parameter
        FOREIGN KEY (report_id, version_no, parameter_id)
        REFERENCES report_parameters(report_id, version_no, parameter_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE report_cube_configs (
    report_id               VARCHAR(64) NOT NULL,
    version_no              INT NOT NULL,
    cube_enabled            BOOLEAN NOT NULL DEFAULT FALSE,
    cube_mcp_tool_enabled   BOOLEAN NULL,
    dimensions_json         JSON NULL,
    measures_json           JSON NULL,
    filters_json            JSON NULL,
    order_by_json           JSON NULL,
    input_layout_json       JSON NULL,
    linked_input_type       VARCHAR(500) NULL,
    compose_enabled         BOOLEAN NOT NULL DEFAULT FALSE,
    compose_mcp_tool_enabled BOOLEAN NULL,
    compose_max_cubes       INT NULL,
    compose_max_limit       INT NULL,
    compose_timeout_ms      INT NULL,
    PRIMARY KEY (report_id, version_no),
    CONSTRAINT fk_report_cube_configs_version
        FOREIGN KEY (report_id, version_no)
        REFERENCES report_versions(report_id, version_no) ON DELETE CASCADE,
    CONSTRAINT chk_report_cube_compose_max_cubes
        CHECK (compose_max_cubes IS NULL OR compose_max_cubes > 0),
    CONSTRAINT chk_report_cube_compose_max_limit
        CHECK (compose_max_limit IS NULL OR compose_max_limit > 0),
    CONSTRAINT chk_report_cube_compose_timeout
        CHECK (compose_timeout_ms IS NULL OR compose_timeout_ms > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE report_mcp_exposures (
    report_id               VARCHAR(64) NOT NULL,
    version_no              INT NOT NULL,
    exposure_id             CHAR(64) NOT NULL,
    route_id                CHAR(64) NOT NULL,
    route_method            VARCHAR(16) NOT NULL,
    route_path              VARCHAR(1000) NOT NULL,
    kind                    VARCHAR(32) NOT NULL,
    name                    VARCHAR(200) NOT NULL,
    description             TEXT NULL,
    description_path        VARCHAR(1000) NULL,
    mime_type               VARCHAR(200) NULL,
    enabled                 BOOLEAN NOT NULL DEFAULT TRUE,
    ordinal                 INT NOT NULL DEFAULT 0,
    PRIMARY KEY (report_id, version_no, exposure_id),
    UNIQUE KEY uq_report_mcp_exposure_name
        (report_id, version_no, kind, name),
    CONSTRAINT fk_report_mcp_exposures_version
        FOREIGN KEY (report_id, version_no)
        REFERENCES report_versions(report_id, version_no) ON DELETE CASCADE,
    CONSTRAINT chk_report_mcp_exposure_kind
        CHECK (kind IN ('tool', 'resource', 'resourceTemplate'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_report_mcp_exposures_route
    ON report_mcp_exposures(report_id, version_no, route_id, enabled);

CREATE TABLE report_resource_files (
    report_id               VARCHAR(64) NOT NULL,
    version_no              INT NOT NULL,
    resource_id             CHAR(64) NOT NULL,
    namespace               VARCHAR(200) NOT NULL,
    resource_path           VARCHAR(1000) NOT NULL,
    media_type              VARCHAR(200) NULL,
    content                 LONGBLOB NOT NULL,
    content_size            BIGINT NOT NULL,
    content_sha256          CHAR(64) NOT NULL,
    is_binary               BOOLEAN NOT NULL DEFAULT FALSE,
    created_at              DATETIME(6) NOT NULL,
    PRIMARY KEY (report_id, version_no, resource_id),
    CONSTRAINT fk_report_resource_files_version
        FOREIGN KEY (report_id, version_no)
        REFERENCES report_versions(report_id, version_no) ON DELETE CASCADE,
    CONSTRAINT chk_report_resource_files_size
        CHECK (content_size >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_report_resource_files_digest
    ON report_resource_files(content_sha256);

CREATE INDEX idx_report_resource_files_namespace
    ON report_resource_files(report_id, version_no, namespace);

CREATE TABLE report_resource_folders (
    report_id               VARCHAR(64) NOT NULL,
    version_no              INT NOT NULL,
    folder_id               CHAR(64) NOT NULL,
    namespace               VARCHAR(200) NOT NULL,
    root_path               VARCHAR(1000) NOT NULL,
    uri_prefix              VARCHAR(1000) NOT NULL,
    ordinal                 INT NOT NULL DEFAULT 0,
    PRIMARY KEY (report_id, version_no, folder_id),
    CONSTRAINT fk_report_resource_folders_version
        FOREIGN KEY (report_id, version_no)
        REFERENCES report_versions(report_id, version_no) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE report_skill_roots (
    report_id               VARCHAR(64) NOT NULL,
    version_no              INT NOT NULL,
    skill_id                CHAR(64) NOT NULL,
    folder_id               CHAR(64) NOT NULL,
    skill_root              VARCHAR(1000) NOT NULL,
    ordinal                 INT NOT NULL DEFAULT 0,
    PRIMARY KEY (report_id, version_no, skill_id),
    CONSTRAINT fk_report_skill_roots_folder
        FOREIGN KEY (report_id, version_no, folder_id)
        REFERENCES report_resource_folders(report_id, version_no, folder_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE runtime_generations (
    generation_no           BIGINT NOT NULL,
    source_revision         VARCHAR(128) NOT NULL,
    status                  VARCHAR(32) NOT NULL,
    report_count            INT NOT NULL,
    build_manifest_json     JSON NOT NULL,
    diagnostics_json        JSON NULL,
    requested_by            VARCHAR(128) NOT NULL,
    requested_at            DATETIME(6) NOT NULL,
    activated_at            DATETIME(6) NULL,
    retired_at              DATETIME(6) NULL,
    PRIMARY KEY (generation_no),
    UNIQUE KEY uq_runtime_generations_source_revision (source_revision),
    CONSTRAINT chk_runtime_generations_status
        CHECK (status IN ('building', 'active', 'failed', 'retired'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE report_warmup_runs (
    run_id                  VARCHAR(64) NOT NULL,
    report_id               VARCHAR(64) NOT NULL,
    version_no              INT NOT NULL,
    source_revision         BIGINT NOT NULL,
    spec_hash               CHAR(64) NOT NULL,
    plan_key                CHAR(64) NOT NULL,
    active_key              VARCHAR(128) NULL,
    status                  VARCHAR(32) NOT NULL,
    requested_by            VARCHAR(128) NOT NULL,
    cache_name              VARCHAR(128) NULL,
    cache_provider          VARCHAR(512) NULL,
    connector_name          VARCHAR(128) NULL,
    index_column            VARCHAR(128) NULL,
    planned_cases           INT NOT NULL DEFAULT 0,
    completed_cases         INT NOT NULL DEFAULT 0,
    max_cases               INT NULL,
    row_limit               INT NULL,
    entries                 INT NOT NULL DEFAULT 0,
    duration_ns             BIGINT NOT NULL DEFAULT 0,
    diagnostics_json        JSON NULL,
    target_json             JSON NOT NULL,
    requested_at            DATETIME(6) NOT NULL,
    started_at              DATETIME(6) NULL,
    completed_at            DATETIME(6) NULL,
    PRIMARY KEY (run_id),
    UNIQUE KEY uq_report_warmup_runs_active (active_key),
    CONSTRAINT fk_report_warmup_runs_version
        FOREIGN KEY (report_id, version_no)
        REFERENCES report_versions(report_id, version_no),
    CONSTRAINT chk_report_warmup_runs_status
        CHECK (status IN ('accepted', 'running', 'completed', 'partial', 'failed', 'canceled'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_report_warmup_runs_version_requested
    ON report_warmup_runs(report_id, version_no, requested_at DESC);

CREATE TABLE bff_sessions (
    session_id_hash         CHAR(64) NOT NULL,
    subject_id              VARCHAR(128) NOT NULL,
    payload_ciphertext      BLOB NOT NULL,
    expires_at_unix         BIGINT NOT NULL,
    created_at              DATETIME(6) NOT NULL,
    PRIMARY KEY (session_id_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_bff_sessions_expires
    ON bff_sessions(expires_at_unix);

CREATE INDEX idx_bff_sessions_subject_expires
    ON bff_sessions(subject_id, expires_at_unix DESC);

CREATE TABLE report_publications (
    report_id               VARCHAR(64) NOT NULL,
    active_version_no       INT NOT NULL,
    desired_version_no      INT NULL,
    desired_generation      BIGINT NOT NULL,
    active_generation       BIGINT NULL,
    publication_status      VARCHAR(32) NOT NULL,
    runtime_revision        VARCHAR(128) NULL,
    spec_hash               CHAR(64) NOT NULL,
    published_by            VARCHAR(128) NOT NULL,
    published_at            DATETIME(6) NOT NULL,
    activated_at            DATETIME(6) NULL,
    failure_json            JSON NULL,
    PRIMARY KEY (report_id),
    CONSTRAINT fk_report_publications_version
        FOREIGN KEY (report_id, active_version_no)
        REFERENCES report_versions(report_id, version_no),
    CONSTRAINT fk_report_publications_desired_version
        FOREIGN KEY (report_id, desired_version_no)
        REFERENCES report_versions(report_id, version_no),
    CONSTRAINT fk_report_publications_generation
        FOREIGN KEY (active_generation) REFERENCES runtime_generations(generation_no),
    CONSTRAINT chk_report_publications_status
        CHECK (publication_status IN ('pending', 'active', 'failed', 'unpublishing'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_report_publications_generation
    ON report_publications(active_generation, publication_status);

-- report_publication_events is intentionally append-only. It records the
-- outcome of a requested lifecycle transition without changing the active
-- publication state used by the runtime.
CREATE TABLE report_publication_events (
    event_id                CHAR(64) NOT NULL,
    report_id               VARCHAR(64) NOT NULL,
    owner_id                VARCHAR(128) NOT NULL,
    operation               VARCHAR(32) NOT NULL,
    version_no              INT NULL,
    generation_no           BIGINT NULL,
    status                  VARCHAR(32) NOT NULL,
    requested_by            VARCHAR(128) NOT NULL,
    reason                  VARCHAR(1000) NULL,
    failure_code            VARCHAR(128) NULL,
    failure_message         VARCHAR(1000) NULL,
    occurred_at             DATETIME(6) NOT NULL,
    PRIMARY KEY (event_id),
    CONSTRAINT fk_report_publication_events_report
        FOREIGN KEY (report_id) REFERENCES reports(id) ON DELETE CASCADE,
    CONSTRAINT fk_report_publication_events_version
        FOREIGN KEY (report_id, version_no)
        REFERENCES report_versions(report_id, version_no),
    CONSTRAINT fk_report_publication_events_generation
        FOREIGN KEY (generation_no) REFERENCES runtime_generations(generation_no),
    CONSTRAINT chk_report_publication_events_operation
        CHECK (operation IN ('publish', 'rollback', 'unpublish')),
    CONSTRAINT chk_report_publication_events_status
        CHECK (status IN ('succeeded', 'failed'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_report_publication_events_owner_time
    ON report_publication_events(owner_id, occurred_at DESC);

CREATE INDEX idx_report_publication_events_report_time
    ON report_publication_events(report_id, occurred_at DESC);

CREATE TABLE report_acl (
    report_id               VARCHAR(64) NOT NULL,
    subject_type            VARCHAR(32) NOT NULL,
    subject_id              VARCHAR(128) NOT NULL,
    can_view                BOOLEAN NOT NULL DEFAULT FALSE,
    can_run                 BOOLEAN NOT NULL DEFAULT FALSE,
    can_edit                BOOLEAN NOT NULL DEFAULT FALSE,
    can_publish             BOOLEAN NOT NULL DEFAULT FALSE,
    can_use_dql             BOOLEAN NOT NULL DEFAULT FALSE,
    etag                    BIGINT NOT NULL DEFAULT 1,
    PRIMARY KEY (report_id, subject_type, subject_id),
    CONSTRAINT fk_report_acl_report
        FOREIGN KEY (report_id) REFERENCES reports(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
