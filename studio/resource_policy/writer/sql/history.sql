SELECT history."tenant_id", history."resource_kind", history."resource_id", history."resource_version", history."revision", history."policies_json", history."actor_id", history."occurred_at" FROM (
    SELECT r.*
FROM resource_policy_revisions r

) history