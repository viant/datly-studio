// Package value owns pure release metadata validation shared by native lifecycle hooks and compatibility adapters.
package value

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/viant/authz"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

type ComponentBinding struct {
	Root               artifactcohort.RootRef `json:"root"`
	DataSourceID       string                 `json:"dataSourceId"`
	ComponentKind      string                 `json:"componentKind"`
	ComponentID        string                 `json:"componentId"`
	ComponentRevision  string                 `json:"componentRevision"`
	Method             string                 `json:"method"`
	Route              string                 `json:"route"`
	SchemaFingerprint  string                 `json:"schemaFingerprint"`
	ContentFingerprint string                 `json:"contentFingerprint"`
}

func ValidateBindings(bundle contract.Bundle, bindings []ComponentBinding) ([]ComponentBinding, []byte, string, error) {
	if contract.ValidateBundle(bundle) != nil {
		return nil, nil, "", contract.ErrInvalid
	}
	allowed := map[string]artifactcohort.RootRef{}
	for _, root := range bundle.Roots {
		allowed[artifactcohort.RootKey(root)] = root
	}
	copy := append([]ComponentBinding(nil), bindings...)
	sort.Slice(copy, func(i, j int) bool { return BindingKey(copy[i]) < BindingKey(copy[j]) })
	prior := ""
	for _, binding := range copy {
		key := BindingKey(binding)
		if key <= prior || allowed[artifactcohort.RootKey(binding.Root)] != binding.Root || binding.Root.Kind != "window" || binding.DataSourceID == "" || strings.TrimSpace(binding.DataSourceID) != binding.DataSourceID || binding.ComponentID == "" || strings.TrimSpace(binding.ComponentID) != binding.ComponentID || binding.ComponentRevision == "" || strings.TrimSpace(binding.ComponentRevision) != binding.ComponentRevision || (binding.ComponentKind != "dynamic" && binding.ComponentKind != "linked") || (binding.Method != http.MethodGet && binding.Method != http.MethodPost) || !strings.HasPrefix(binding.Route, "/") || strings.HasPrefix(binding.Route, "//") || strings.ContainsAny(binding.Route, "?#") || !ValidHash(binding.SchemaFingerprint) || !ValidHash(binding.ContentFingerprint) {
			return nil, nil, "", contract.ErrInvalid
		}
		prior = key
	}
	if copy == nil {
		copy = []ComponentBinding{}
	}
	raw, err := json.Marshal(copy)
	if err != nil {
		return nil, nil, "", err
	}
	sum := sha256.Sum256(raw)
	return copy, raw, hex.EncodeToString(sum[:]), nil
}

func BindingKey(value ComponentBinding) string {
	return artifactcohort.RootKey(value.Root) + "\x00" + value.DataSourceID + "\x00" + value.ComponentKind + "\x00" + value.ComponentID + "\x00" + value.ComponentRevision
}

func ValidHash(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// StageRevisionWithBindings writes immutable producer closure metadata and the
// pending operation in one caller-independent DB transaction. Its commitment
// does not activate the channel; Publisher projection and Complete follow.

func MarshalDependencies(input []authz.NamedDependency) ([]byte, error) {
	copy := append([]authz.NamedDependency(nil), input...)
	sort.Slice(copy, func(i, j int) bool { return DependencyKey(copy[i]) < DependencyKey(copy[j]) })
	prior := ""
	for _, item := range copy {
		key := DependencyKey(item)
		if item.Revision < 1 || item.Resource.Tenant == "" || item.Resource.Kind == "" || item.Resource.ID == "" || item.Resource.Version == "" || key <= prior {
			return nil, contract.ErrInvalid
		}
		prior = key
	}
	if copy == nil {
		copy = []authz.NamedDependency{}
	}
	return json.Marshal(copy)
}

func DependencyKey(value authz.NamedDependency) string {
	return value.Resource.Tenant + "\x00" + value.Resource.Kind + "\x00" + value.Resource.ID + "\x00" + value.Resource.Version
}

func DecodeDependencies(raw []byte) ([]authz.NamedDependency, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var value []authz.NamedDependency
	if StrictJSON(raw, &value) != nil || value == nil {
		return nil, contract.ErrInvalid
	}
	canonical, err := MarshalDependencies(value)
	if err != nil {
		return nil, contract.ErrInvalid
	}
	var ordered []authz.NamedDependency
	if json.Unmarshal(canonical, &ordered) != nil || !reflect.DeepEqual(value, ordered) {
		return nil, contract.ErrInvalid
	}
	return value, nil
}

// SQL JSON columns may rewrite whitespace and object-key order. Decode exact
// typed values while still rejecting unknown fields and trailing JSON values;
// callers then validate ordering and immutable content separately.

func StrictJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return contract.ErrInvalid
		}
		return err
	}
	return nil
}

func DecodeBindings(bundle contract.Bundle, raw []byte, hash string) ([]ComponentBinding, error) {
	var bindings []ComponentBinding
	if StrictJSON(raw, &bindings) != nil || bindings == nil {
		return nil, contract.ErrInvalid
	}
	validated, canonical, expected, err := ValidateBindings(bundle, bindings)
	if err != nil || hash != expected || !reflect.DeepEqual(bindings, validated) || len(canonical) == 0 {
		return nil, contract.ErrInvalid
	}
	return validated, nil
}
