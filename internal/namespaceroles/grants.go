// Package namespaceroles stores namespace grants in the existing roles JSON.
package namespaceroles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Grants accepts the legacy viewer array and writes a document only when
// management is delegated. No role acquires management through migration.
type Grants struct {
	View   []string `json:"view"`
	Manage []string `json:"manage"`
}

func Decode(value *string) (Grants, error) {
	g := Grants{View: []string{}, Manage: []string{}}
	if value == nil {
		return g, nil
	}
	raw := strings.TrimSpace(*value)
	if raw == "null" || raw == "" {
		return g, fmt.Errorf("invalid namespace role grants")
	}
	d := json.NewDecoder(bytes.NewBufferString(raw))
	d.DisallowUnknownFields()
	var err error
	if strings.HasPrefix(raw, "[") {
		err = d.Decode(&g.View)
	} else {
		err = d.Decode(&g)
	}
	if err != nil || d.Decode(new(any)) != io.EOF {
		return g, fmt.Errorf("invalid namespace role grants")
	}
	if g.View == nil {
		g.View = []string{}
	}
	if g.Manage == nil {
		g.Manage = []string{}
	}
	return g, g.Validate()
}

func (g Grants) Validate() error {
	for kind, roles := range [][]string{g.View, g.Manage} {
		seen := map[string]bool{}
		for _, role := range roles {
			if role == "" || kind == 1 && role == "*" || strings.TrimSpace(role) != role || seen[role] {
				return fmt.Errorf("namespace roles must be distinct, exact nonempty role names")
			}
			seen[role] = true
		}
	}
	return nil
}

func (g Grants) Encode() (string, error) {
	if err := g.Validate(); err != nil {
		return "", err
	}
	if g.View == nil {
		g.View = []string{}
	}
	var value any = g
	if len(g.Manage) == 0 {
		value = g.View
	}
	raw, err := json.Marshal(value)
	return string(raw), err
}

func Matches(required, actual []string) bool {
	for _, want := range required {
		for _, role := range actual {
			if want != "" && want == role {
				return true
			}
		}
	}
	return false
}
