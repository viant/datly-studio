package access

// Intersect combines mandatory scopes without ever widening either decision.
// A bounded empty result, malformed entity or incompatible dimension denies.
func Intersect(left, right Decision) (Decision, error) {
	validate := func(d Decision) bool {
		if !d.Bounded {
			return len(d.Entities) == 0
		}
		if len(d.Entities) == 0 {
			return false
		}
		kind := d.Entities[0].Type
		seen := map[Entity]bool{}
		for _, e := range d.Entities {
			if kind == "" || e.Type != kind || e.ID == "" || seen[e] {
				return false
			}
			seen[e] = true
		}
		return true
	}
	if !validate(left) || !validate(right) {
		return Decision{}, ErrDenied
	}
	if !left.Bounded {
		return right, nil
	}
	if !right.Bounded {
		return left, nil
	}
	allowed := map[Entity]bool{}
	for _, e := range left.Entities {
		allowed[e] = true
	}
	result := Decision{Bounded: true}
	for _, e := range right.Entities {
		if allowed[e] {
			result.Entities = append(result.Entities, e)
		}
	}
	if len(result.Entities) == 0 {
		return Decision{}, ErrDenied
	}
	return result, nil
}
