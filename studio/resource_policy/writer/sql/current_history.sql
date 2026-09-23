SELECT r."tenant_id", r."resource_kind", r."resource_id", r."resource_version", r."actor_id", r."policies_json", r."revision", r."occurred_at" FROM (SELECT history."tenant_id", history."resource_kind", history."resource_id", history."resource_version", history."revision", history."policies_json", history."actor_id", history."occurred_at" FROM (
    SELECT r.*
FROM resource_policy_revisions r

) history) r WHERE $criteria.CompositeIn("r", $Unsafe.ProjectCurrentHistoryParentKeys($CurrentPolicy))