// Package predicatecatalog defines the explicit authorization-predicate link
// boundary used by Studio and embedding applications.
package predicatecatalog

import (
	"fmt"
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

type Catalog struct{ entries map[string]Descriptor }

func New(packages ...Package) (*Catalog, error) {
	result := &Catalog{entries: map[string]Descriptor{}}
	for _, pkg := range packages {
		path := strings.TrimSpace(pkg.Path)
		if path == "" {
			return nil, fmt.Errorf("predicate package path is required")
		}
		alias := strings.TrimSpace(pkg.Alias)
		for _, typ := range pkg.Types {
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
		}
	}
	return result, nil
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
