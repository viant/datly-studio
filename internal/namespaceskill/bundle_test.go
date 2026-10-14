package namespaceskill

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
)

func exampleBundle() Bundle {
	return Bundle{Content: "---\nname: customer-guide\ndescription: Query customer records.\n---\nRead [the reference](references/query.md).\n", Files: []File{{Path: "references/query.md", Content: "Use the customer MCP tool."}, {Path: "assets/icon.bin", Content: base64.StdEncoding.EncodeToString([]byte{0, 255, 1}), IsBinary: true}}}
}

func TestImmutableBundle(t *testing.T) {
	ctx := context.Background()
	bundle := exampleBundle()
	first, err := Compile(ctx, "ns-1", "skill-1", 1, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "customer-guide" || first.Description != "Query customer records." || len(first.Plans) != 3 {
		t.Fatalf("unexpected snapshot: %+v", first)
	}
	if first.Folder.URIPrefix != "skill://ns-1/skill-1/1/customer-guide" {
		t.Fatalf("skill URI namespace authority: %s", first.Folder.URIPrefix)
	}
	if first.Plans[0].URI() == "" {
		t.Fatal("missing resource URI")
	}
	bundle.Files[0].Content = "mutated"
	data, err := fs.ReadFile(first.Folder.FS, "references/query.md")
	if err != nil || string(data) != "Use the customer MCP tool." {
		t.Fatalf("snapshot changed: %q %v", data, err)
	}
	binary, err := fs.ReadFile(first.Folder.FS, "assets/icon.bin")
	if err != nil || string(binary) != string([]byte{0, 255, 1}) {
		t.Fatalf("binary corruption: %v %v", binary, err)
	}
	reordered := exampleBundle()
	reordered.Files[0], reordered.Files[1] = reordered.Files[1], reordered.Files[0]
	second, err := Compile(ctx, "ns-1", "skill-1", 1, reordered)
	if err != nil || first.SHA256 != second.SHA256 {
		t.Fatalf("unstable digest: %v", err)
	}
	restored, err := Restore(ctx, "ns-1", "skill-1", 1, first.JSON, first.SHA256)
	if err != nil || restored.Description != first.Description {
		t.Fatalf("restore: %v", err)
	}
	revisionTwo, err := Compile(ctx, "ns-1", "skill-1", 2, reordered)
	if err != nil || revisionTwo.Plans[0].URI() == first.Plans[0].URI() {
		t.Fatalf("revision URI must differ: %v", err)
	}
	otherNamespace, err := Compile(ctx, "ns-2", "skill-1", 1, reordered)
	if err != nil || otherNamespace.Plans[0].URI() == first.Plans[0].URI() {
		t.Fatalf("namespace URI must differ: %v", err)
	}
	if _, err = Restore(ctx, "ns-1", "skill-1", 1, append(first.JSON, ' '), first.SHA256); err != nil {
		t.Fatalf("JSON whitespace changed bundle identity: %v", err)
	}
}

func TestInvalidBundles(t *testing.T) {
	tests := map[string]func(*Bundle){
		"traversal":  func(b *Bundle) { b.Files = []File{{Path: "../outside", Content: "x"}} },
		"absolute":   func(b *Bundle) { b.Files = []File{{Path: "/outside", Content: "x"}} },
		"escaped":    func(b *Bundle) { b.Files = []File{{Path: "%2e%2e/outside", Content: "x"}} },
		"backslash":  func(b *Bundle) { b.Files = []File{{Path: "a\\b", Content: "x"}} },
		"entrypoint": func(b *Bundle) { b.Files = []File{{Path: "SKILL.md", Content: "x"}} },
		"duplicate":  func(b *Bundle) { b.Files = append(b.Files, b.Files[0]) },
		"file directory collision": func(b *Bundle) {
			b.Files = []File{{Path: "references", Content: "x"}, {Path: "references/a.md", Content: "y"}}
		},
		"entry directory collision": func(b *Bundle) { b.Files = []File{{Path: "SKILL.md/a", Content: "y"}} },
		"bad binary":                func(b *Bundle) { b.Files = []File{{Path: "x.bin", Content: "?", IsBinary: true}} },
		"bad UTF8":                  func(b *Bundle) { b.Files = []File{{Path: "x.txt", Content: string([]byte{255})}} },
		"bad frontmatter":           func(b *Bundle) { b.Content = "# No frontmatter" },
		"empty description":         func(b *Bundle) { b.Content = "---\nname: guide\ndescription: ''\n---\n" },
		"oversized":                 func(b *Bundle) { b.Files = []File{{Path: "x.txt", Content: strings.Repeat("x", 16*1024*1024)}} },
		"file count":                func(b *Bundle) { b.Files = make([]File, 512) },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			b := exampleBundle()
			change(&b)
			if _, err := Compile(context.Background(), "ns", "skill", 1, b); err == nil {
				t.Fatal("invalid bundle accepted")
			}
		})
	}
	for _, identity := range []string{"", "../ns", "ns/a", "ns%2f", strings.Repeat("a", 65)} {
		if _, err := Compile(context.Background(), identity, "skill", 1, exampleBundle()); err == nil {
			t.Fatalf("invalid namespace identity %q accepted", identity)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Compile(ctx, "ns", "skill", 1, exampleBundle()); err != context.Canceled {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestRestoreDatabaseNormalizedJSON(t *testing.T) {
	ctx := context.Background()
	original, err := Compile(ctx, "ns-1", "skill-1", 1, exampleBundle())
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]any
	if err := json.Unmarshal(original.JSON, &values); err != nil {
		t.Fatal(err)
	}
	stored, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Restore(ctx, "ns-1", "skill-1", 1, stored, original.SHA256)
	if err != nil {
		t.Fatalf("database JSON formatting changed bundle identity: %v", err)
	}
	if restored.Bundle.Content != original.Bundle.Content || restored.SHA256 != original.SHA256 {
		t.Fatal("authored content changed during restore")
	}
	values["content"] = original.Bundle.Content + "Changed instructions.\n"
	changed, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Restore(ctx, "ns-1", "skill-1", 1, changed, original.SHA256); err == nil {
		t.Fatal("accepted changed skill instructions")
	}
}
