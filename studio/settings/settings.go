// Package settings defines typed namespace policies stored independently of DQL.
package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const SecurityKey = "security"
const ResourceLimitsKey = "resourceLimits"

type Security struct {
	// Nil and empty lists select no external packages; each package is opt-in.
	PredicatePackages        []string `json:"predicatePackages"`
	RequireSecurityPredicate bool     `json:"requireSecurityPredicate"`
}

type ResourceLimits struct {
	MaxBytesProcessed   *int64 `json:"maxBytesProcessed,omitempty"`
	MaxReferencedTables *int   `json:"maxReferencedTables,omitempty"`
	MaxReturnedRows     *int64 `json:"maxReturnedRows,omitempty"`
}

type Namespace struct {
	Security       *Security       `json:"security,omitempty"`
	ResourceLimits *ResourceLimits `json:"resourceLimits,omitempty"`
}

// Decode rejects unknown policy keys and malformed values rather than ignoring them.
func Decode(key string, data json.RawMessage) (any, error) {
	var value any
	switch key {
	case SecurityKey:
		value = &Security{}
	case ResourceLimitsKey:
		value = &ResourceLimits{}
	default:
		return nil, fmt.Errorf("unsupported namespace setting %q", key)
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, fmt.Errorf("setting %s must be an object", key)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("setting must contain exactly one JSON value")
	}
	switch v := value.(type) {
	case *Security:
		seen := map[string]bool{}
		for _, p := range v.PredicatePackages {
			if p == "" || strings.TrimSpace(p) != p || seen[p] {
				return nil, fmt.Errorf("predicate packages must be distinct nonempty package paths")
			}
			seen[p] = true
		}
		if v.RequireSecurityPredicate && len(v.PredicatePackages) == 0 {
			return nil, fmt.Errorf("required security predicates need an eligible package")
		}
	case *ResourceLimits:
		if v.MaxBytesProcessed != nil && *v.MaxBytesProcessed < 0 || v.MaxReferencedTables != nil && *v.MaxReferencedTables < 0 || v.MaxReturnedRows != nil && *v.MaxReturnedRows < 0 {
			return nil, fmt.Errorf("resource thresholds must be nonnegative")
		}
	}
	return value, nil
}

// ResourceCheckEnabled requires explicit true or false, preserving sparse semantics.
func ResourceCheckEnabled(data json.RawMessage) (bool, error) {
	var value struct {
		Enabled *bool `json:"resourceCheckEnabled"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return false, err
	}
	if value.Enabled == nil {
		return false, fmt.Errorf("resourceCheckEnabled must be explicitly supplied")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return false, fmt.Errorf("resource check command must contain one JSON value")
	}
	return *value.Enabled, nil
}
