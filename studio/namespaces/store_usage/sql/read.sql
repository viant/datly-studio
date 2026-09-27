SELECT usage."used" FROM  (SELECT COUNT(1) AS used FROM components
WHERE owner_id = $OwnerId AND namespace = $Name AND deleted_at IS NULL
)  usage WHERE 1 = 1