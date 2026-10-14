// Package datly_studio exposes the linked Studio assembly metadata to embedding
// applications without requiring a source checkout beside their executable.
package datly_studio

import (
	_ "embed"
	"go.yaml.in/yaml/v3"
)

//go:embed datly.yaml
var nativeConfiguration []byte

// LinkedComponentPackages returns the canonical native Studio package list.
// Hosts supply their own listeners, connectors and identity verification settings.
// Each call returns a fresh slice; it never installs configuration or starts a server.
func LinkedComponentPackages() ([]string, error) {
	var document struct {
		GoBootstrap struct {
			Packages []string `yaml:"Packages"`
		} `yaml:"GoBootstrap"`
	}
	if err := yaml.Unmarshal(nativeConfiguration, &document); err != nil {
		return nil, err
	}
	return document.GoBootstrap.Packages, nil
}
