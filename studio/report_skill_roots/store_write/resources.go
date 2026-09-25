package store_write

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const SkillDatlyResourceNamespace = "studio_report_skill_roots_store_write_skill"

//go:embed "sql/current_skill.sql" "sql/read.sql" "sql/skill_keys.sql"
var SkillDatlyResources embed.FS
