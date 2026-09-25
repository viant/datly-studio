package access

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// EntityID is an opaque authorized ID. JSON strings remain strings. Positive
// JSON integers through MaxUint64 become exact decimal strings; floats,
// exponents, negatives, zero and larger integers are rejected.
type EntityID string

func (id *EntityID) UnmarshalJSON(raw []byte) error {
	if len(raw) == 0 {
		return ErrDenied
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil || value == "" || strings.TrimSpace(value) != value {
			return ErrDenied
		}
		*id = EntityID(value)
		return nil
	}
	if raw[0] == '0' {
		return ErrDenied
	}
	for _, digit := range raw {
		if digit < '0' || digit > '9' {
			return ErrDenied
		}
	}
	if _, err := strconv.ParseUint(string(raw), 10, 64); err != nil {
		return ErrDenied
	}
	*id = EntityID(raw)
	return nil
}

// EntityGroups is the canonical allowedEntities fact: entity type -> IDs.
// A missing type or an empty ID list grants no entity access.
type EntityGroups map[string][]EntityID

// MarshalJSON always publishes the grouped contract, including when an older
// provider supplied only the flat compatibility view.
func (f Facts) MarshalJSON() ([]byte, error) {
	flat, err := f.FlatEntities()
	if err != nil {
		return nil, err
	}
	groups := f.EntityGroups
	if groups == nil {
		groups, err = GroupEntities(flat)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(struct {
		Subject         string       `json:"subject"`
		Tenant          string       `json:"tenant"`
		Issuer          string       `json:"issuer"`
		Roles           []string     `json:"roles"`
		Exposures       []string     `json:"exposures"`
		AllowedEntities EntityGroups `json:"allowedEntities"`
		ValidUntil      time.Time    `json:"validUntil"`
	}{f.Subject, f.Tenant, f.Issuer, f.Roles, f.Exposures, groups, f.ValidUntil})
}

// NormalizeEntityGroups returns the existing flat evaluator view. Malformed
// types, IDs, or duplicate IDs fail closed. Map keys are sorted for stable
// decision and editor output; the ID order within each type is preserved.
func NormalizeEntityGroups(groups EntityGroups) ([]Entity, error) {
	types := make([]string, 0, len(groups))
	for entityType := range groups {
		types = append(types, entityType)
	}
	sort.Strings(types)
	var flat []Entity
	for _, entityType := range types {
		if !validEntityType(entityType) {
			return nil, ErrDenied
		}
		ids := groups[entityType]
		if ids == nil {
			return nil, ErrDenied
		}
		seen := map[EntityID]bool{}
		for _, id := range ids {
			if id == "" || strings.TrimSpace(string(id)) != string(id) || seen[id] {
				return nil, ErrDenied
			}
			seen[id] = true
			flat = append(flat, Entity{Type: entityType, ID: string(id)})
		}
	}
	return flat, nil
}

// GroupEntities is the explicit compatibility conversion for legacy flat
// facts. Duplicate type/ID pairs are rejected rather than merged silently.
func GroupEntities(flat []Entity) (EntityGroups, error) {
	groups := EntityGroups{}
	seen := map[Entity]bool{}
	for _, entity := range flat {
		if !validEntityType(entity.Type) || entity.ID == "" || strings.TrimSpace(entity.ID) != entity.ID || seen[entity] {
			return nil, ErrDenied
		}
		seen[entity] = true
		groups[entity.Type] = append(groups[entity.Type], EntityID(entity.ID))
	}
	return groups, nil
}

func validEntityType(entityType string) bool {
	if entityType == "" || strings.ContainsAny(entityType, "/\\") {
		return false
	}
	for _, char := range entityType {
		if unicode.IsSpace(char) || unicode.IsControl(char) {
			return false
		}
	}
	return true
}

// FlatEntities uses the canonical map when present. When both public and
// legacy views are populated, they must describe the same authority exactly.
func (f Facts) FlatEntities() ([]Entity, error) {
	if f.EntityGroups == nil {
		groups, err := GroupEntities(f.Entities)
		if err != nil {
			return nil, err
		}
		return NormalizeEntityGroups(groups)
	}
	flat, err := NormalizeEntityGroups(f.EntityGroups)
	if err != nil {
		return nil, err
	}
	if f.Entities != nil {
		legacyGroups, err := GroupEntities(f.Entities)
		if err != nil {
			return nil, err
		}
		legacy, err := NormalizeEntityGroups(legacyGroups)
		if err != nil || !sameEntities(flat, legacy) {
			return nil, ErrDenied
		}
	}
	return flat, nil
}

func sameEntities(a, b []Entity) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[Entity]bool{}
	for _, entity := range a {
		seen[entity] = true
	}
	for _, entity := range b {
		if !seen[entity] {
			return false
		}
	}
	return true
}

// IDsForType exposes a typed auth-context field usable by native bindings.
// It returns no IDs for absent or empty types and never trusts a conflicting
// flat compatibility view.
func (f Facts) IDsForType(entityType string) ([]EntityID, error) {
	flat, err := f.FlatEntities()
	if err != nil {
		return nil, err
	}
	var ids []EntityID
	for _, entity := range flat {
		if entity.Type == entityType {
			ids = append(ids, EntityID(entity.ID))
		}
	}
	return ids, nil
}

// DecodeEntityGroups parses the canonical JSON object without float64,
// rejecting duplicate keys, null ID lists and malformed or duplicate IDs.
func DecodeEntityGroups(raw []byte) (EntityGroups, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	opening, err := dec.Token()
	if err != nil || opening != json.Delim('{') {
		return nil, ErrDenied
	}
	groups := EntityGroups{}
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return nil, ErrDenied
		}
		entityType, ok := keyToken.(string)
		if !ok {
			return nil, ErrDenied
		}
		if _, duplicate := groups[entityType]; duplicate {
			return nil, ErrDenied
		}
		var ids []EntityID
		if err := dec.Decode(&ids); err != nil || ids == nil {
			return nil, ErrDenied
		}
		groups[entityType] = ids
	}
	closing, err := dec.Token()
	if err != nil || closing != json.Delim('}') {
		return nil, ErrDenied
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, ErrDenied
	}
	if _, err := NormalizeEntityGroups(groups); err != nil {
		return nil, err
	}
	return groups, nil
}
