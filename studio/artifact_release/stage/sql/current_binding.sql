SELECT r.scope_id, r.bundle_hash, r.bindings_json, r.bindings_hash, r.created_at, r.created_by, r.updated_at, r.updated_by FROM (SELECT * FROM (SELECT b.* FROM artifact_release_bundle_bindings b
${predicate.Builder().CombineAnd($predicate.FilterGroup(2,"AND")).Build("WHERE")}
) binding) r WHERE $criteria.CompositeIn("r", $Unsafe.ProjectCurrentBindingParentKeys($CurrentStage))