package readerinspection

import "github.com/viant/datly/spec"

// HydrateColumns enriches authoring metadata without replacing its source spans,
// embed references, relation keys, or settings with a compiled SQL wrapper.
func HydrateColumns(authored, resolved *spec.View) {
	if authored == nil || resolved == nil {
		return
	}
	authored.Columns = resolved.Columns
	for _, relation := range authored.Relations {
		if relation == nil || relation.View == nil {
			continue
		}
		for _, candidate := range resolved.Relations {
			if candidate == nil || candidate.View == nil {
				continue
			}
			a, b := relation.View, candidate.View
			if a.Name != "" && a.Name == b.Name || a.Namespace != "" && a.Namespace == b.Namespace {
				HydrateColumns(a, b)
				break
			}
		}
	}
}
