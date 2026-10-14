package host

import (
	"fmt"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"strings"
	"sync"
)

var linkedPackages struct {
	sync.RWMutex
	packages []predicatecatalog.Package
}

// RegisterPredicatePackage is an opt-in startup hook for a linked extension.
// Importing host alone never imports application or example predicates.
// Handler discovery remains limited to the configured, already-linked package.
func RegisterPredicatePackage(pkg predicatecatalog.Package) error {
	pkg.Path = strings.TrimSpace(pkg.Path)
	if pkg.Path == "" || pkg.Path == studioAuthorizationPath {
		return fmt.Errorf("an external predicate package is required")
	}
	linkedPackages.Lock()
	defer linkedPackages.Unlock()
	for _, existing := range linkedPackages.packages {
		if existing.Path == pkg.Path {
			return fmt.Errorf("predicate package %s is already registered", pkg.Path)
		}
	}
	pkg.Types = append(pkg.Types[:0:0], pkg.Types...)
	if pkg.AuthorizationTypes != nil {
		pkg.AuthorizationTypes = append([]string{}, pkg.AuthorizationTypes...)
	}
	linkedPackages.packages = append(linkedPackages.packages, pkg)
	return nil
}
func registeredPredicatePackages() []predicatecatalog.Package {
	linkedPackages.RLock()
	defer linkedPackages.RUnlock()
	return append([]predicatecatalog.Package(nil), linkedPackages.packages...)
}
