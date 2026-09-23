SELECT r."tenant_id", r."resource_kind", r."resource_id", r."resource_version", r."revision" FROM (SELECT head."tenant_id", head."resource_kind", head."resource_id", head."resource_version", head."revision" FROM  (
    SELECT h.*
FROM resource_policy_heads h

)  head WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $PolicyKeys)