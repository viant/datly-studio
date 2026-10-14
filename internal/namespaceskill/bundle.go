// Package namespaceskill validates independent, immutable namespace skill bundles.
package namespaceskill

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"sort"
	"strings"
	"testing/fstest"
	"unicode/utf8"

	"github.com/viant/datly-studio/sdk"
	mcpresource "github.com/viant/datly/mcp/resource"
	skillformat "github.com/viant/mcp-protocol/extension/skills"
)

// File is relative to the skill root. Binary content uses standard base64.
type File = sdk.NamespaceSkillFile

// Bundle preserves the authored SKILL.md verbatim.
type Bundle = sdk.NamespaceSkillBundle

type Snapshot struct {
	Bundle      Bundle
	Name        string
	Description string
	JSON        json.RawMessage
	SHA256      string
	Folder      mcpresource.Folder
	Plans       []*mcpresource.Plan
}

// Compile validates the complete inventory without fetching any external links.
// Identity and revision are supplied by the server, never by bundle content.
func Compile(ctx context.Context, namespaceID, skillID string, revision int, bundle Bundle) (*Snapshot, error) {
	if ctx == nil {
		return nil, fmt.Errorf("skill context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !validIdentity(namespaceID) || !validIdentity(skillID) || revision < 1 {
		return nil, fmt.Errorf("namespace skill identity and positive revision are required")
	}
	if len(bundle.Files) >= 512 || len(bundle.Content) > 16*1024*1024 {
		return nil, fmt.Errorf("skill exceeds SEP-2640 inventory limits")
	}
	front, err := skillformat.Frontmatter([]byte(bundle.Content))
	if err != nil {
		return nil, err
	}
	name := front["name"].(string)
	description := front["description"].(string)
	bundle.Title = strings.TrimSpace(bundle.Title)
	if bundle.Title == "" {
		bundle.Title = name
	}
	if !utf8.ValidString(bundle.Title) || utf8.RuneCountInString(bundle.Title) > 300 {
		return nil, fmt.Errorf("skill title must contain at most 300 characters")
	}
	bundle.Files = append([]File(nil), bundle.Files...)
	sort.Slice(bundle.Files, func(i, j int) bool { return bundle.Files[i].Path < bundle.Files[j].Path })
	source := fstest.MapFS{"SKILL.md": &fstest.MapFile{Data: []byte(bundle.Content), Mode: 0444}}
	total := len(bundle.Content)
	for _, file := range bundle.Files {
		if !fs.ValidPath(file.Path) || file.Path == "." || strings.ContainsAny(file.Path, "\\%?#:") || !utf8.ValidString(file.Path) {
			return nil, fmt.Errorf("invalid skill file path %q", file.Path)
		}
		if _, exists := source[file.Path]; exists {
			return nil, fmt.Errorf("duplicate skill file %q", file.Path)
		}
		if len(file.Content) > base64.StdEncoding.EncodedLen(16*1024*1024) {
			return nil, fmt.Errorf("oversized skill file %q", file.Path)
		}
		data := []byte(file.Content)
		if file.IsBinary {
			data, err = base64.StdEncoding.Strict().DecodeString(file.Content)
			if err != nil {
				return nil, fmt.Errorf("invalid base64 skill file %q", file.Path)
			}
		} else if !utf8.Valid(data) {
			return nil, fmt.Errorf("text skill file %q must be UTF-8", file.Path)
		}
		total += len(data)
		if total > 16*1024*1024 {
			return nil, fmt.Errorf("skill exceeds SEP-2640 inventory limits")
		}
		source[file.Path] = &fstest.MapFile{Data: data, Mode: 0444}
	}
	// A file cannot also be a directory; MapFS otherwise masks such collisions.
	for name := range source {
		parts := strings.Split(name, "/")
		for i := 1; i < len(parts); i++ {
			if _, exists := source[strings.Join(parts[:i], "/")]; exists {
				return nil, fmt.Errorf("skill file/directory collision %q", name)
			}
		}
	}
	prefix := (&url.URL{Scheme: "skill", Host: namespaceID, Path: fmt.Sprintf("/%s/%d/%s", skillID, revision, name)}).String()
	folder := mcpresource.Folder{Namespace: "namespace-skill-" + namespaceID + "-" + skillID, Root: ".", URIPrefix: prefix, Description: description, FS: source, Skills: []string{"."}}
	plans, err := (mcpresource.Publisher{}).Compile(ctx, []mcpresource.Folder{folder})
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(raw)
	return &Snapshot{Bundle: bundle, Name: name, Description: description, JSON: raw, SHA256: hex.EncodeToString(digest[:]), Folder: folder, Plans: plans}, nil
}

// Restore verifies canonical bundle content before compiling the revision.
// JSON databases may change object key order and whitespace during storage.
// Authored skill text and supporting-file contents remain byte-for-byte inputs.
func Restore(ctx context.Context, namespaceID, skillID string, revision int, raw json.RawMessage, digest string) (*Snapshot, error) {
	var bundle Bundle
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&bundle); err != nil {
		return nil, fmt.Errorf("invalid namespace skill bundle: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("namespace skill bundle must contain one JSON object")
	}
	canonical, err := json.Marshal(bundle)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canonical)
	if digest != hex.EncodeToString(sum[:]) {
		return nil, fmt.Errorf("namespace skill content digest does not match")
	}
	return Compile(ctx, namespaceID, skillID, revision, bundle)
}

func validIdentity(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
