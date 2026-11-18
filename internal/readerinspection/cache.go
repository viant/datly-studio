package readerinspection

import "github.com/viant/datly/spec"

// AvailableCacheNames projects trusted storage candidates, matching native
// reader initialization's non-nil/enabled selection. Policy-only entries are
// excluded; endpoints, storage validity, and connectivity remain native concerns.
// Keep map keys unchanged: native named storage lookup uses those exact keys.
func AvailableCacheNames(caches map[string]*spec.CacheSettings) []string {
	var names []string
	for name, settings := range caches {
		if settings != nil && settings.Enabled {
			names = append(names, name)
		}
	}
	return names
}
