// Package predicatecatalog defines the explicit authorization-predicate link
// boundary used by Studio and embedding applications.
package predicatecatalog

import (
	"fmt"
	"github.com/viant/datly/typecatalog"
	x "github.com/viant/x"
	xpredicate "github.com/viant/xdatly/predicate"
	"github.com/viant/xunsafe"
	"reflect"
	"sort"
	"strings"
)

type Package struct {
	Alias string
	Path  string
	Types []reflect.Type
}

type Descriptor struct {
	Alias    string
	Package  string
	TypeName string
}

type Catalog struct {
	entries map[string]Descriptor
	types   *typecatalog.Catalog
}

// RuntimeTypes returns detached package authority retaining executable Go types.
func (c *Catalog) RuntimeTypes() (*typecatalog.Catalog, error) {
	if c == nil {
		return typecatalog.NewCatalog(), nil
	}
	return c.types.Clone()
}

func New(packages ...Package) (*Catalog, error) {
	result := &Catalog{entries: map[string]Descriptor{}, types: typecatalog.NewCatalog()}
	for _, pkg := range packages {
		path := strings.TrimSpace(pkg.Path)
		if path == "" {
			return nil, fmt.Errorf("predicate package path is required")
		}
		alias := strings.TrimSpace(pkg.Alias)
		types := pkg.Types
		if types == nil {
			types = linkedPredicateTypes(path)
			if len(types) == 0 {
				return nil, fmt.Errorf("predicate package %s has no linked handler types", path)
			}
		}
		for _, typ := range types {
			for typ != nil && typ.Kind() == reflect.Pointer {
				typ = typ.Elem()
			}
			if typ == nil || typ.Name() == "" {
				return nil, fmt.Errorf("predicate package %s contains an unnamed type", path)
			}
			key := path + "#" + typ.Name()
			if _, exists := result.entries[key]; exists {
				return nil, fmt.Errorf("duplicate predicate link %s.%s", path, typ.Name())
			}
			result.entries[key] = Descriptor{Alias: alias, Package: path, TypeName: typ.Name()}
			if err := result.types.Register(typecatalog.TypeOriginPackage, x.NewType(typ, x.WithPkgPath(path))); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// linkedPredicateTypes selects already-linked implementations from one trusted
// package path. Package discovery/linking remains an explicit AST/build step;
// this never registers a type or makes source-only declarations executable.
func linkedPredicateTypes(packagePath string) []reflect.Type {
	handler := reflect.TypeFor[xpredicate.Handler]()
	seen := map[reflect.Type]bool{}
	var result []reflect.Type
	for _, typ := range xunsafe.PackageTypes(packagePath) {
		for typ != nil && typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ == nil || typ.PkgPath() != packagePath || typ.Name() == "" || seen[typ] || !reflect.PointerTo(typ).Implements(handler) {
			continue
		}
		seen[typ] = true
		result = append(result, typ)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name() < result[j].Name() })
	return result
}

func (c *Catalog) Contains(packagePath, typeName string) bool {
	if c == nil {
		return false
	}
	_, ok := c.entries[strings.TrimSpace(packagePath)+"#"+strings.TrimSpace(typeName)]
	return ok
}

func (c *Catalog) Descriptors() []Descriptor {
	if c == nil {
		return nil
	}
	result := make([]Descriptor, 0, len(c.entries))
	for _, entry := range c.entries {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Package == result[j].Package {
			return result[i].TypeName < result[j].TypeName
		}
		return result[i].Package < result[j].Package
	})
	return result
}
