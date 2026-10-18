// Package codeccatalog links host-approved Go codecs for dynamic DQL compilation.
package codeccatalog

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/viant/datly/typecatalog"
	"github.com/viant/x"
	xcodec "github.com/viant/xdatly/codec"
	"github.com/viant/xunsafe"
)

// Package names an already-linked package. Nil Types discovers linked codec
// factories/instances; an explicit slice restricts the available codec types.
// This is startup authority, never browser or DQL-supplied executable code.
type Package struct {
	Path  string
	Types []reflect.Type
}

// Link adds codec types to a runtime catalog without classifying them as predicates.
func Link(catalog *typecatalog.Catalog, packages ...Package) error {
	if catalog == nil {
		return fmt.Errorf("runtime type catalog is required")
	}
	seen := map[string]bool{}
	for _, pkg := range packages {
		path := strings.TrimSpace(pkg.Path)
		if path == "" || seen[path] {
			return fmt.Errorf("empty or duplicate codec package %q", path)
		}
		seen[path] = true
		types := pkg.Types
		if types == nil {
			types = linkedTypes(path)
		}
		if len(types) == 0 {
			return fmt.Errorf("codec package %s has no linked codec types", path)
		}
		names := map[string]bool{}
		for _, typ := range types {
			for typ != nil && typ.Kind() == reflect.Pointer {
				typ = typ.Elem()
			}
			if typ == nil || typ.Name() == "" || typ.PkgPath() != path || !isCodec(typ) || names[typ.Name()] {
				return fmt.Errorf("invalid or duplicate codec type in %s", path)
			}
			names[typ.Name()] = true
			key := path + "." + typ.Name()
			existing, found, err := catalog.ResolveRuntimeType(typecatalog.PackageAuthority, key)
			if err != nil {
				return err
			}
			if found {
				if existing != typ {
					return fmt.Errorf("conflicting runtime codec %s", key)
				}
				continue
			}
			if err := catalog.Register(typecatalog.TypeOriginPackage, x.NewType(typ, x.WithPkgPath(path))); err != nil {
				return err
			}
		}
	}
	return nil
}
func isCodec(typ reflect.Type) bool {
	pointer := reflect.PointerTo(typ)
	return pointer.Implements(reflect.TypeFor[xcodec.Factory]()) || pointer.Implements(reflect.TypeFor[xcodec.Instance]())
}
func linkedTypes(path string) []reflect.Type {
	seen := map[reflect.Type]bool{}
	var result []reflect.Type
	for _, typ := range xunsafe.PackageTypes(path) {
		for typ != nil && typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ == nil || typ.Name() == "" || typ.PkgPath() != path || seen[typ] || !isCodec(typ) {
			continue
		}
		seen[typ] = true
		result = append(result, typ)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name() < result[j].Name() })
	return result
}
