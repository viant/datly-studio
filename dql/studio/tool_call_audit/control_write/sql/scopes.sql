SELECT owner_id,name,namespace_id,etag,audit_enabled,updated_at FROM namespaces WHERE deleted_at IS NULL AND status='active'
