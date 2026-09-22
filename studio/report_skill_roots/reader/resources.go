package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const SkillDatlyResourceNamespace = "studio_report_skill_roots_reader_skill"

//go:embed "sql/skill.sql"
var SkillDatlyResources embed.FS
