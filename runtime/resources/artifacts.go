package resources

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	mcpresource "github.com/viant/datly/mcp/resource"
	"github.com/viant/forge/backend/artifactcohort"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/mcp-protocol/server"
)

// SourceArtifact retains producer ownership beside closed wire bytes. It is
// preparation output, not a publication or a permission grant.
type SourceArtifact struct {
	Source      Version
	OriginalURI string
	Artifact    *artifactcohort.Artifact
}

// SkillArtifacts adapts the existing native compiler's complete immutable skill
// inventory. The release owner supplies the exact component versions; this does
// not select latest versions, publish a release, or contact an MCP server.
func (l *Loaded) SkillArtifacts(ctx context.Context, namespace string, versions []Version) ([]SourceArtifact, error) {
	if l == nil || ctx == nil {
		return nil, fmt.Errorf("resource snapshot is required")
	}
	owners := map[string]Version{}
	for _, v := range versions {
		if _, ok := l.Fingerprints[v]; !ok {
			return nil, fmt.Errorf("component version was not loaded")
		}
		if prior, ok := owners[v.ReportID]; ok && prior != v {
			return nil, fmt.Errorf("ambiguous component version")
		}
		owners[v.ReportID] = v
	}
	plans, err := (mcpresource.Publisher{Resources: l.Store}).Compile(ctx, l.Folders)
	if err != nil {
		return nil, err
	}
	catalog, err := mcpresource.NewCatalog(plans)
	if err != nil {
		return nil, err
	}
	registry := server.NewRegistry()
	if err = catalog.RegisterSkills(registry); err != nil {
		return nil, err
	}
	reader := mcpresource.NewHandler(catalog, nil)
	builder := artifactcohort.ArtifactBuilder[map[string]any]{}
	result := []SourceArtifact{}
	seen := map[string]bool{}
	for _, skill := range registry.ListRegisteredSkills() {
		var source Version
		for prefix, owner := range l.ResourceReports {
			if strings.HasPrefix(skill.Uri, prefix) {
				candidate, ok := owners[owner]
				if !ok {
					return nil, fmt.Errorf("skill owner was not selected")
				}
				if source.ReportID != "" && source != candidate {
					return nil, fmt.Errorf("ambiguous skill owner")
				}
				source = candidate
			}
		}
		if source.ReportID == "" {
			return nil, fmt.Errorf("skill source is unavailable")
		}
		name, _ := skill.Frontmatter["name"].(string)
		if name == "" || strings.ContainsAny(name, "/:?#%\\") {
			return nil, fmt.Errorf("invalid skill name")
		}
		base := strings.TrimSuffix(skill.Uri, "SKILL.md")
		files := map[string][]byte{}
		var document []byte
		for _, file := range skill.Resources.Files {
			if !strings.HasPrefix(file.Uri, base) {
				return nil, fmt.Errorf("skill file outside entrypoint tree")
			}
			read, readErr := reader.Handle(ctx, &schema.ReadResourceRequest{Params: schema.ReadResourceRequestParams{Uri: file.Uri}})
			if readErr != nil {
				return nil, readErr
			}
			if read == nil || len(read.Contents) != 1 {
				return nil, fmt.Errorf("skill file content unavailable")
			}
			content := read.Contents[0]
			data := []byte(content.Text)
			if content.Blob != "" {
				data, err = base64.StdEncoding.DecodeString(content.Blob)
				if err != nil {
					return nil, err
				}
			}
			if file.Uri == skill.Uri {
				document = data
			} else {
				files[strings.TrimPrefix(file.Uri, base)] = data
			}
		}
		if len(document) == 0 {
			return nil, fmt.Errorf("skill entrypoint unavailable")
		}
		value, err := builder.Build(ctx, artifactcohort.RootRef{URI: "skill://" + namespace + "/" + source.ReportID + "/" + name, Kind: "skill", Version: uint64(source.VersionNo)}, map[string]any{"document": document, "files": files})
		if err != nil {
			return nil, err
		}
		if seen[value.Root.URI] {
			return nil, fmt.Errorf("duplicate logical skill identity")
		}
		seen[value.Root.URI] = true
		result = append(result, SourceArtifact{Source: source, OriginalURI: skill.Uri, Artifact: value})
	}
	return result, nil
}
