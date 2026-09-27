SELECT usage."used" FROM  (SELECT COUNT(1) AS used FROM components
WHERE default_connector_name = $Name AND deleted_at IS NULL
)  usage WHERE 1 = 1