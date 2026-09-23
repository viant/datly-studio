package access

import (
	"errors"
	"testing"
)

// bounded builds a bounded decision from type/id pairs to keep tables compact.
func bounded(pairs ...string) Decision {
	d := Decision{Bounded: true}
	for i := 0; i+1 < len(pairs); i += 2 {
		d.Entities = append(d.Entities, Entity{Type: pairs[i], ID: pairs[i+1]})
	}
	return d
}

// snapshot deep-copies a decision so later comparisons cannot be fooled by
// shared backing arrays.
func snapshot(d Decision) Decision {
	out := Decision{Bounded: d.Bounded}
	if d.Entities != nil {
		out.Entities = append([]Entity{}, d.Entities...)
	}
	return out
}

// sameScope compares bounded-ness and ordered entity content; nil and empty
// slices are treated as equal because callers only observe len and elements.
func sameScope(a, b Decision) bool {
	if a.Bounded != b.Bounded || len(a.Entities) != len(b.Entities) {
		return false
	}
	for i := range a.Entities {
		if a.Entities[i] != b.Entities[i] {
			return false
		}
	}
	return true
}

// sameSet compares bounded-ness and entity membership ignoring order.
func sameSet(a, b Decision) bool {
	if a.Bounded != b.Bounded || len(a.Entities) != len(b.Entities) {
		return false
	}
	seen := map[Entity]int{}
	for _, e := range a.Entities {
		seen[e]++
	}
	for _, e := range b.Entities {
		if seen[e] == 0 {
			return false
		}
		seen[e]--
	}
	return true
}

func containsEntity(entities []Entity, e Entity) bool {
	for _, candidate := range entities {
		if candidate == e {
			return true
		}
	}
	return false
}

func hasDuplicateEntities(entities []Entity) bool {
	seen := map[Entity]bool{}
	for _, e := range entities {
		if seen[e] {
			return true
		}
		seen[e] = true
	}
	return false
}

func TestIntersectScopes(t *testing.T) {
	unbounded := Decision{}
	cases := []struct {
		name  string
		left  Decision
		right Decision
		want  Decision // ignored when deny is true
		deny  bool
	}{
		// --- unbounded combinations -------------------------------------
		{name: "unbounded/unbounded stays unbounded", left: unbounded, right: unbounded, want: unbounded},
		{name: "unbounded with non-nil empty entities is still unbounded", left: Decision{Entities: []Entity{}}, right: Decision{Entities: []Entity{}}, want: unbounded},
		{name: "bounded/unbounded keeps bounded scope", left: bounded("project", "101", "project", "102"), right: unbounded, want: bounded("project", "101", "project", "102")},
		{name: "unbounded/bounded keeps bounded scope", left: unbounded, right: bounded("project", "101", "project", "102"), want: bounded("project", "101", "project", "102")},
		{name: "bounded single/unbounded", left: bounded("project", "101"), right: unbounded, want: bounded("project", "101")},

		// --- bounded intersections --------------------------------------
		{name: "identical bounded scopes", left: bounded("project", "101", "project", "102"), right: bounded("project", "101", "project", "102"), want: bounded("project", "101", "project", "102")},
		{name: "partial intersection narrows to common entities", left: bounded("project", "101", "project", "102", "project", "103"), right: bounded("project", "102", "project", "103", "project", "104"), want: bounded("project", "102", "project", "103")},
		{name: "partial intersection ignores operand order", left: bounded("project", "103", "project", "101"), right: bounded("project", "101", "project", "103", "project", "200"), want: bounded("project", "101", "project", "103")},
		{name: "subset on the right narrows to the subset", left: bounded("project", "101", "project", "102", "project", "103"), right: bounded("project", "102"), want: bounded("project", "102")},
		{name: "subset on the left narrows to the subset", left: bounded("project", "102"), right: bounded("project", "101", "project", "102", "project", "103"), want: bounded("project", "102")},

		// --- disjoint / incompatible ------------------------------------
		{name: "disjoint ids deny", left: bounded("project", "101"), right: bounded("project", "102"), deny: true},
		{name: "disjoint many ids deny", left: bounded("project", "1", "project", "2", "project", "3"), right: bounded("project", "4", "project", "5", "project", "6"), deny: true},
		{name: "same id different entity type denies", left: bounded("project", "101"), right: bounded("account", "101"), deny: true},
		{name: "entity type is case sensitive", left: bounded("project", "101"), right: bounded("Project", "101"), deny: true},
		{name: "entity id is compared exactly without trimming", left: bounded("project", "101"), right: bounded("project", " 101"), deny: true},
		{name: "entity id is case sensitive", left: bounded("project", "abc"), right: bounded("project", "ABC"), deny: true},

		// --- empty bounded scopes ---------------------------------------
		{name: "empty bounded left denies even with unbounded right", left: Decision{Bounded: true}, right: unbounded, deny: true},
		{name: "empty bounded right denies even with unbounded left", left: unbounded, right: Decision{Bounded: true}, deny: true},
		{name: "empty bounded both deny", left: Decision{Bounded: true}, right: Decision{Bounded: true}, deny: true},
		{name: "empty bounded left denies against populated right", left: Decision{Bounded: true, Entities: []Entity{}}, right: bounded("project", "101"), deny: true},
		{name: "empty bounded right denies against populated left", left: bounded("project", "101"), right: Decision{Bounded: true, Entities: []Entity{}}, deny: true},

		// --- duplicates ---------------------------------------------------
		{name: "duplicate ids on left deny", left: bounded("project", "101", "project", "101"), right: bounded("project", "101"), deny: true},
		{name: "duplicate ids on right deny", left: bounded("project", "101"), right: bounded("project", "101", "project", "101"), deny: true},
		{name: "duplicate ids on left deny even against unbounded", left: bounded("project", "101", "project", "101"), right: unbounded, deny: true},
		{name: "duplicate ids on right deny even against unbounded", left: unbounded, right: bounded("project", "101", "project", "101"), deny: true},
		{name: "non adjacent duplicate denies", left: bounded("project", "101", "project", "102", "project", "101"), right: unbounded, deny: true},

		// --- malformed entities -----------------------------------------
		{name: "empty id on left denies", left: bounded("project", ""), right: unbounded, deny: true},
		{name: "empty id on right denies", left: unbounded, right: bounded("project", ""), deny: true},
		{name: "empty id mixed with valid denies whole scope", left: bounded("project", "101", "project", ""), right: bounded("project", "101"), deny: true},
		{name: "empty type on left denies", left: bounded("", "101"), right: unbounded, deny: true},
		{name: "empty type on right denies", left: unbounded, right: bounded("", "101"), deny: true},
		{name: "empty type mixed with valid denies whole scope", left: bounded("project", "101", "", "102"), right: bounded("project", "101"), deny: true},
		{name: "zero entity denies", left: Decision{Bounded: true, Entities: []Entity{{}}}, right: unbounded, deny: true},
		{name: "mixed types within left denies", left: bounded("project", "101", "account", "101"), right: unbounded, deny: true},
		{name: "mixed types within right denies", left: unbounded, right: bounded("project", "101", "account", "101"), deny: true},
		{name: "mixed types within left denies even when right matches first type", left: bounded("project", "101", "account", "7"), right: bounded("project", "101"), deny: true},
		{name: "mixed types within right denies even when left matches first type", left: bounded("project", "101"), right: bounded("project", "101", "account", "7"), deny: true},

		// --- malformed unbounded carrying entities -----------------------
		{name: "unbounded left with entities denies", left: Decision{Entities: []Entity{{"project", "101"}}}, right: unbounded, deny: true},
		{name: "unbounded right with entities denies", left: unbounded, right: Decision{Entities: []Entity{{"project", "101"}}}, deny: true},
		{name: "unbounded left with entities denies against bounded right", left: Decision{Entities: []Entity{{"project", "101"}}}, right: bounded("project", "101"), deny: true},
		{name: "unbounded right with entities denies against bounded left", left: bounded("project", "101"), right: Decision{Entities: []Entity{{"project", "101"}}}, deny: true},
		{name: "unbounded with malformed entity denies", left: Decision{Entities: []Entity{{"", ""}}}, right: unbounded, deny: true},
		{name: "both unbounded with entities deny", left: Decision{Entities: []Entity{{"project", "101"}}}, right: Decision{Entities: []Entity{{"project", "101"}}}, deny: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			leftBefore, rightBefore := snapshot(tc.left), snapshot(tc.right)

			got, err := Intersect(tc.left, tc.right)

			// Operands must never be mutated regardless of outcome.
			if !sameScope(tc.left, leftBefore) {
				t.Fatalf("left operand mutated: before %+v after %+v", leftBefore, tc.left)
			}
			if !sameScope(tc.right, rightBefore) {
				t.Fatalf("right operand mutated: before %+v after %+v", rightBefore, tc.right)
			}

			if tc.deny {
				if !errors.Is(err, ErrDenied) {
					t.Fatalf("expected ErrDenied, got decision %+v err %v", got, err)
				}
				// A denied result must not leak partial scope alongside the error.
				if got.Bounded || len(got.Entities) != 0 {
					t.Fatalf("denied result leaked scope: %+v", got)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error %v (decision %+v)", err, got)
				}
				if !sameScope(got, tc.want) {
					t.Fatalf("want %+v got %+v", tc.want, got)
				}
			}

			// Non-widening invariants that hold independent of the table's
			// expected value: a bounded operand can never yield an unbounded
			// result, and every returned entity must be present in every
			// bounded operand.
			if err == nil {
				if (tc.left.Bounded || tc.right.Bounded) && !got.Bounded {
					t.Fatalf("bounded operand widened to unbounded: left %+v right %+v got %+v", tc.left, tc.right, got)
				}
				for _, operand := range []Decision{tc.left, tc.right} {
					if !operand.Bounded {
						continue
					}
					for _, e := range got.Entities {
						if !containsEntity(operand.Entities, e) {
							t.Fatalf("result entity %+v absent from bounded operand %+v", e, operand)
						}
					}
				}
				if hasDuplicateEntities(got.Entities) {
					t.Fatalf("result contains duplicates: %+v", got)
				}
			}

			// Intersection must be commutative in outcome and membership.
			swapped, swappedErr := Intersect(tc.right, tc.left)
			if (swappedErr == nil) != (err == nil) {
				t.Fatalf("asymmetric outcome: (l,r) err=%v (r,l) err=%v", err, swappedErr)
			}
			if err == nil && !sameSet(got, swapped) {
				t.Fatalf("asymmetric scope: (l,r)=%+v (r,l)=%+v", got, swapped)
			}
		})
	}
}

// TestIntersectDoesNotWidenAfterRepeatedNarrowing checks that chaining
// intersections is monotonic: each step can only remove entities, and a later
// unbounded decision cannot restore anything an earlier bounded one excluded.
func TestIntersectDoesNotWidenAfterRepeatedNarrowing(t *testing.T) {
	current := bounded("project", "1", "project", "2", "project", "3", "project", "4")
	steps := []Decision{
		bounded("project", "1", "project", "2", "project", "3"),
		{}, // unbounded: must not re-admit "4"
		bounded("project", "2", "project", "3", "project", "9"),
		{}, // unbounded again: must not re-admit "1"
	}
	previous := snapshot(current)
	for i, step := range steps {
		next, err := Intersect(current, step)
		if err != nil {
			t.Fatalf("step %d unexpected deny: %v", i, err)
		}
		if !next.Bounded {
			t.Fatalf("step %d widened to unbounded", i)
		}
		if len(next.Entities) > len(previous.Entities) {
			t.Fatalf("step %d grew scope from %+v to %+v", i, previous, next)
		}
		for _, e := range next.Entities {
			if !containsEntity(previous.Entities, e) {
				t.Fatalf("step %d admitted %+v not in previous %+v", i, e, previous)
			}
		}
		previous = snapshot(next)
		current = next
	}
	if !sameSet(current, bounded("project", "2", "project", "3")) {
		t.Fatalf("final scope %+v", current)
	}
	// Narrowing to nothing must deny rather than yield an empty bounded scope.
	if _, err := Intersect(current, bounded("project", "4")); !errors.Is(err, ErrDenied) {
		t.Fatalf("expected deny on disjoint final step, got %v", err)
	}
}

// TestIntersectResultIsIndependentOfInputsWhenBothBounded verifies that a
// computed intersection owns its storage: mutating the result afterwards must
// not alter either operand, and mutating an operand afterwards must not alter
// the result.
func TestIntersectResultIsIndependentOfInputsWhenBothBounded(t *testing.T) {
	left := bounded("project", "101", "project", "102", "project", "103")
	right := bounded("project", "102", "project", "103", "project", "104")
	leftBefore, rightBefore := snapshot(left), snapshot(right)

	got, err := Intersect(left, right)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	want := snapshot(got)

	// Mutate the result in place and by append; inputs must be untouched.
	got.Entities[0] = Entity{"project", "evil"}
	got.Entities = append(got.Entities, Entity{"project", "extra"})
	if !sameScope(left, leftBefore) {
		t.Fatalf("mutating result changed left: %+v", left)
	}
	if !sameScope(right, rightBefore) {
		t.Fatalf("mutating result changed right: %+v", right)
	}

	// Recompute and then mutate operands; the earlier result must be stable.
	got2, err := Intersect(left, right)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	left.Entities[1] = Entity{"project", "tampered"}
	right.Entities[0] = Entity{"project", "tampered"}
	if !sameScope(got2, want) {
		t.Fatalf("mutating operands changed result: want %+v got %+v", want, got2)
	}
}

// TestIntersectPassthroughAliasing documents (without mandating) that when one
// operand is unbounded the returned scope is the other operand and may share
// its backing array. Intersect itself never writes to either operand and the
// contract treats decisions as read-only values, so aliasing does not widen or
// mutate scope. The test asserts only contract-relevant properties: returned
// content equals the bounded operand, Intersect wrote nothing to either
// operand, and a caller that needs private ownership can copy the result. If a
// future change makes Intersect copy defensively this test still passes.
func TestIntersectPassthroughAliasing(t *testing.T) {
	for _, tc := range []struct {
		name        string
		left, right Decision
	}{
		{name: "bounded left", left: bounded("project", "101", "project", "102"), right: Decision{}},
		{name: "bounded right", left: Decision{}, right: bounded("project", "101", "project", "102")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := tc.left
			if !source.Bounded {
				source = tc.right
			}
			sourceBefore := snapshot(source)

			got, err := Intersect(tc.left, tc.right)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if !sameScope(got, sourceBefore) {
				t.Fatalf("passthrough altered content: want %+v got %+v", sourceBefore, got)
			}
			if !sameScope(source, sourceBefore) {
				t.Fatalf("Intersect mutated bounded operand: %+v", source)
			}

			// A caller that needs private ownership can snapshot the result;
			// the snapshot must be equal yet detached from the operand.
			owned := snapshot(got)
			source.Entities[0] = Entity{"project", "tampered"}
			if !sameScope(owned, sourceBefore) {
				t.Fatalf("snapshot shares storage with operand: %+v", owned)
			}
		})
	}
}

// TestIntersectDeniedReturnsZeroDecision guards the failure shape: every deny
// must return the sentinel error and a zero decision so an inattentive caller
// cannot mistake a partially built scope for a grant.
func TestIntersectDeniedReturnsZeroDecision(t *testing.T) {
	for _, tc := range []struct {
		name        string
		left, right Decision
	}{
		{name: "disjoint", left: bounded("project", "1"), right: bounded("project", "2")},
		{name: "empty bounded", left: Decision{Bounded: true}, right: Decision{}},
		{name: "malformed unbounded", left: Decision{Entities: []Entity{{"project", "1"}}}, right: Decision{}},
		{name: "duplicate", left: bounded("project", "1", "project", "1"), right: Decision{}},
		{name: "mixed type", left: bounded("project", "1", "account", "1"), right: Decision{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Intersect(tc.left, tc.right)
			if err != ErrDenied {
				t.Fatalf("want ErrDenied sentinel, got %v", err)
			}
			if got.Bounded || got.Entities != nil {
				t.Fatalf("denied decision not zero: %+v", got)
			}
		})
	}
}
