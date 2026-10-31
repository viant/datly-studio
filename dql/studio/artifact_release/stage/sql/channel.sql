SELECT c.scope_id,c.name,c.revision,c.pending_operation_id,c.updated_at,c.updated_by,
       c.tenant_id AS expected_tenant_id,c.account_id AS expected_account_id,
       c.workspace_id AS expected_workspace_id,c.namespace AS expected_namespace,
       c.description AS expected_description,c.policy_refs_json AS expected_policy_refs_json,
       c.active_bundle_hash AS expected_active_bundle_hash,
       $TargetBundleHash AS target_bundle_hash
FROM artifact_release_channels c
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}
