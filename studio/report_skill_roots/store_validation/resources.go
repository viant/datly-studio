package store_validation

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const SkillDatlyResourceNamespace = "studio_report_skill_roots_store_validation_skill"

//go:embed "sql/read.sql"
var SkillDatlyResources embed.FS
