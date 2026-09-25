SELECT r."tenant_id", r."resource_kind", r."resource_id", r."resource_version", r."actor_id", r."created_at", r."created_by", r."updated_at", r."updated_by", r."policies_json", r."revision", r."occurred_at" FROM (SELECT history."tenant_id", history."resource_kind", history."resource_id", history."resource_version", history."revision", history."policies_json", history."actor_id", history."occurred_at", history."created_at", history."created_by", history."updated_at", history."updated_by" FROM (
    SELECT r.*
FROM resource_policy_revisions r

) history) r WHERE $criteria.CompositeIn("r", $Unsafe.ProjectCurrentHistoryParentKeys($CurrentPolicy))