SELECT history."tenant_id", history."resource_kind", history."resource_id", history."resource_version", history."revision", history."policies_json", history."actor_id", history."occurred_at", history."created_at", history."created_by", history."updated_at", history."updated_by" FROM (
    SELECT r.*
FROM resource_policy_revisions r

) history