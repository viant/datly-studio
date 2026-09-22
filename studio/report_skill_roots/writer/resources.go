package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const SkillDatlyResourceNamespace = "studio_report_skill_roots_writer_skill"

//go:embed "sql/current_skill.sql" "sql/skill.sql" "sql/skill_keys.sql"
var SkillDatlyResources embed.FS
