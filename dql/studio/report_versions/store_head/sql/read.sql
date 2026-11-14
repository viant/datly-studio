SELECT c.id, c.namespace_id, c.last_version_no, c.last_version_no AS max_version_no FROM components c WHERE c.id = $ReportId AND c.namespace_id = $NamespaceId
