package connectorinit

import (
	"bytes"
	"encoding/json"
)

// ConnectionOptionsEqual excludes discovery lookup preferences. Editing lookup
// lists changes no credentials or connection setup and does not invalidate a probe.
func ConnectionOptionsEqual(left, right json.RawMessage) bool {
	clean := func(raw json.RawMessage) ([]byte, error) {
		value := map[string]json.RawMessage{}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, err
			}
		}
		delete(value, "discovery")
		if len(value) == 0 {
			return []byte("{}"), nil
		}
		return json.Marshal(value)
	}
	a, errA := clean(left)
	b, errB := clean(right)
	if errA != nil || errB != nil {
		return bytes.Equal(left, right)
	}
	return bytes.Equal(a, b)
}
