SELECT claim."namespace", claim."report_id", claim."created_at", claim."created_by", claim."updated_at", claim."updated_by", claim."should_delete" FROM  (SELECT claim.namespace, claim.report_id, claim.created_at, claim.created_by,
       claim.updated_at, claim.updated_by, FALSE AS should_delete
FROM resource_namespace_claims claim
)  claim