// Package managementcatalog builds portable administrative resource definitions.
// The managing producer authorizes discovery and all invoked SDK operations.
package managementcatalog

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/viant/datly-studio/nativeapp"
	"github.com/viant/datly-studio/runtime/managementwindow"
	ds "github.com/viant/forge/backend/datasource/contract"
	"github.com/viant/forge/backend/types"
	"go.yaml.in/yaml/v3"
)

type Config struct {
	Namespace, ProviderIdentity, MCPService                    string
	PublicURL, ApplicationID, ApplicationName, ApplicationIcon string
	Navigation                                                 json.RawMessage
	Sections                                                   []Section
}
type Definition = managementwindow.Definition

//go:embed windows/*/shared/*
var windowSources embed.FS

// NamespaceWindow returns functional management windows and their datasource
// dependencies. Only screens supplied by this producer are bound in navigation.
func NamespaceWindow(config Config) ([]Definition, error) {
	names := []string{"namespaces"}
	if config.ApplicationID == "" || config.ApplicationID == "datastudio" {
		names = append(names, "connectors", "components")
	}
	var definitions []Definition
	if config.ApplicationID == "" || config.ApplicationID == "datastudio" {
		definitions = append(definitions, Definition{URI: "style://" + config.Namespace + "/workspace", JSON: nativeapp.WorkspaceStyle()})
	}
	for _, name := range names {
		built, err := managementWindow(config, name)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, built...)
	}
	nav, err := navigation(config)
	if err != nil {
		return nil, err
	}
	if config.PublicURL != "" {
		base, _ := url.Parse(config.PublicURL) // navigation already validated the host URL.
		base.Path = strings.TrimRight(base.Path, "/") + "/v1/studio/sdk-mcp/mcp"
		base.RawQuery = ""
		uri := "mcp://" + config.Namespace + "/admin"
		connection, err := json.Marshal(map[string]any{"name": uri, "transport": map[string]any{"type": "streamable", "url": base.String()}, "auth": map[string]any{"backendForFrontend": true, "useIdToken": true}, "toolsListVisibility": "private"})
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, Definition{URI: uri, JSON: connection})
	}
	return append(definitions, nav), nil
}
func managementWindow(config Config, name string) ([]Definition, error) {
	raw, err := windowSources.ReadFile("windows/" + name + "/shared/main.yaml")
	if err != nil {
		return nil, err
	}
	var source struct {
		Window      *types.Window             `yaml:"window"`
		DataSources map[string]*ds.DataSource `yaml:"dataSources"`
	}
	decoder := yaml.NewDecoder(strings.NewReader(string(raw)))
	if err = decoder.Decode(&source); err != nil {
		return nil, fmt.Errorf("management %s source: %w", name, err)
	}
	if source.Window == nil || len(source.DataSources) == 0 {
		return nil, fmt.Errorf("management %s window and datasources required", name)
	}
	code, err := windowSources.ReadFile("windows/" + name + "/shared/main.js")
	if err != nil {
		return nil, err
	}
	if name == "connectors" {
		behavior, err := windowSources.ReadFile("windows/connectors/shared/behavior.js")
		if err != nil {
			return nil, err
		}
		code = []byte("(() => {\n" + strings.Replace(string(behavior), "export default ", "", 1) + "\nreturn " + string(code) + ";\n})()")
	}
	source.Window.Actions = &types.Actions{Code: string(code)}
	source.Window.ActionAliases = []string{"management"}
	operations := []string{"get", "create", "update"}
	if name == "namespaces" {
		operations = append(operations, "delete")
	}
	prefix := "namespace"
	if name == "connectors" {
		prefix = "connector"
		operations = append(operations, "test", "activate", "disable", "delete")
	}
	if name == "components" {
		prefix = "component"
		operations = []string{"components.get", "components.create", "components.update", "versions.list", "versions.create", "versions.clone", "versions.inspect", "versions.apply", "versions.validate", "versions.builder", "resources.get", "resources.upsert_file", "preview.execute", "publications.get", "publications.publish", "publications.unpublish", "publications.rollback"}
	}
	disabled := false
	for _, operation := range operations {
		id := prefix + "_" + strings.ReplaceAll(operation, ".", "_")
		method := "studio.sdk." + name + "." + operation
		if name == "components" {
			method = "studio.sdk." + operation
		}
		source.DataSources[id] = &ds.DataSource{ID: id, DataSource: types.DataSource{Cardinality: "object"}, Backend: &ds.Backend{Kind: ds.BackendMCPTool, Method: method}, Cache: &ds.CachePolicy{Enabled: &disabled}}
	}
	return managementwindow.Build(managementwindow.Config{Namespace: config.Namespace, ProviderIdentity: config.ProviderIdentity, MCPService: config.MCPService}, name, source.Window, source.DataSources)
}
