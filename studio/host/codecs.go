package host

import (
	"fmt"
	"strings"
	"sync"

	"github.com/viant/datly-studio/studio/codeccatalog"
	"github.com/viant/datly/typecatalog"
)

var linkedCodecPackages struct {
	sync.RWMutex
	packages []codeccatalog.Package
}

// RegisterCodecPackage approves an already-linked codec package at startup.
// Existing authoring/runtime snapshots remain unchanged by later registration.
func RegisterCodecPackage(pkg codeccatalog.Package) error {
	pkg.Path = strings.TrimSpace(pkg.Path)
	if err := codeccatalog.Link(typecatalog.NewCatalog(), pkg); err != nil {
		return err
	}
	linkedCodecPackages.Lock()
	defer linkedCodecPackages.Unlock()
	for _, existing := range linkedCodecPackages.packages {
		if existing.Path == pkg.Path {
			return fmt.Errorf("codec package %s is already registered", pkg.Path)
		}
	}
	if pkg.Types != nil {
		pkg.Types = append(pkg.Types[:0:0], pkg.Types...)
	}
	linkedCodecPackages.packages = append(linkedCodecPackages.packages, pkg)
	return nil
}
func (c Config) codecPackages() []codeccatalog.Package {
	explicit := map[string]bool{}
	for _, pkg := range c.CodecPackages {
		explicit[strings.TrimSpace(pkg.Path)] = true
	}
	linkedCodecPackages.RLock()
	defer linkedCodecPackages.RUnlock()
	var result []codeccatalog.Package
	for _, pkg := range linkedCodecPackages.packages {
		if !explicit[pkg.Path] {
			result = append(result, pkg)
		}
	}
	return append(result, c.CodecPackages...)
}
