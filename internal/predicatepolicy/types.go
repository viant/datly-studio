package predicatepolicy

import (
	"github.com/viant/datly-studio/studio/settings"
	"github.com/viant/datly/typecatalog"
	x "github.com/viant/x"
	xpredicate "github.com/viant/xdatly/predicate"
	"reflect"
)

// Types removes unselected external handlers while retaining codecs,
// dependency shapes and Studio's own authorization types.
func Types(source *typecatalog.Catalog, policy *settings.Namespace) (*typecatalog.Catalog, error) {
	if source == nil {
		source = typecatalog.NewCatalog()
	}
	allowed := map[string]bool{"github.com/viant/datly-studio/studio/authorization": true}
	if policy != nil && policy.Security != nil {
		for _, pkg := range policy.Security.PredicatePackages {
			allowed[pkg] = true
		}
	}
	result := typecatalog.NewCatalog()
	registry, err := source.Registry(typecatalog.PackageAuthority)
	if err != nil {
		return nil, err
	}
	var failure error
	registry.ForEach(func(_ string, typ *x.Type) bool {
		rt := typ.Type
		for rt != nil && rt.Kind() == reflect.Pointer {
			rt = rt.Elem()
		}
		if rt != nil && rt.Kind() != reflect.Interface && reflect.PointerTo(rt).Implements(reflect.TypeFor[xpredicate.Handler]()) && !allowed[typ.PkgPath] {
			return true
		}
		failure = result.Register(typecatalog.TypeOriginPackage, typ)
		return failure == nil
	})
	return result, failure
}

// AllowsPackage applies namespace selection to linked-type discovery.
func AllowsPackage(policy *settings.Namespace, pkg string) bool {
	if policy == nil || pkg == "github.com/viant/datly-studio/studio/authorization" {
		return true
	}
	if policy.Security == nil {
		return false
	}
	for _, selected := range policy.Security.PredicatePackages {
		if selected == pkg {
			return true
		}
	}
	return false
}
