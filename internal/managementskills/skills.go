// Package managementskills publishes built-in agent workflows on management MCP.
package managementskills

import (
	"embed"
	"github.com/viant/datly/mcp/resource"
)

//go:embed skills
var source embed.FS

func Folder() resource.Folder {
	return resource.Folder{Namespace: "studio_management_skills", Root: "skills", URIPrefix: "mcp://datastudio/skills", Description: "Datly Studio authoring and release workflows", FS: source, Skills: []string{"datly-studio-authoring"}}
}
