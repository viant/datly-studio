package catalog

import (
	"context"
	"encoding/json"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	"testing"

	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/scy/auth/jwt"
)

func TestManagementResourceListingPagesAndFilterBoundCursor(t *testing.T) {
	t.Setenv("STUDIO_MANAGEMENT_NAMESPACE", "management")
	t.Setenv("STUDIO_MANAGEMENT_PROVIDER_ID", "provider")
	t.Setenv("STUDIO_MANAGEMENT_MCP_SERVICE", "service")
	t.Setenv("STUDIO_MANAGEMENT_PUBLIC_URL", "https://studio.example.com")
	claims := &jwt.Claims{}
	claims.Subject = "alice"
	input := &ResourcesListInput{Jwt: claims, Auth: &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "alice"}}}
	input.Request.Meta.AdditionalProperties = map[string]any{resourceFilterKey: map[string]any{"namespace": "ns://management", "limit": 1}}
	_, definitions, err := management(claims, input.Auth)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{}
	for _, definition := range definitions {
		expected[definition.URI] = true
	}
	seen := map[string]bool{}
	for page := 0; page <= len(expected); page++ {
		out := &ResourcesListOutput{}
		if err := runResourcesList(context.Background(), nil, input, out); err != nil {
			t.Fatal(err)
		}
		if len(out.Response.Resources) != 1 {
			t.Fatalf("page=%+v", out.Response)
		}
		uri := out.Response.Resources[0].Uri
		if !expected[uri] {
			t.Fatalf("unexpected resource %s", uri)
		}
		if seen[uri] {
			t.Fatalf("duplicate %s", uri)
		}
		seen[uri] = true
		fetched := &ResourceOutput{}
		read := &ResourceInput{Jwt: claims, Auth: input.Auth, Request: namespaceext.ResourceGetRequest{URI: uri}}
		if err := runResource(context.Background(), nil, read, fetched); err != nil || !json.Valid(fetched.Response) {
			t.Fatalf("resource %s: %v", uri, err)
		}
		read.Jwt = nil
		if err := runResource(context.Background(), nil, read, &ResourceOutput{}); err == nil {
			t.Fatal("unauthenticated definition read succeeded")
		}
		input.Request.Cursor = out.Response.NextCursor
		if input.Request.Cursor == nil {
			break
		}
	}
	if input.Request.Cursor != nil || len(seen) != len(expected) {
		t.Fatalf("listing never completed: cursor=%+v resources=%d", input.Request.Cursor, len(seen))
	}
	first := &ResourcesListOutput{}
	if err := runResourcesList(context.Background(), nil, input, first); err != nil {
		t.Fatal(err)
	}
	input.Request.Cursor = first.Response.NextCursor
	input.Request.Meta.AdditionalProperties = map[string]any{resourceFilterKey: map[string]any{"namespace": "management", "kind": "window", "limit": 1}}
	if err := runResourcesList(context.Background(), nil, input, &ResourcesListOutput{}); err == nil {
		t.Fatal("cursor reused across filter scope")
	}
	input.Jwt = nil
	if err := runResourcesList(context.Background(), nil, input, &ResourcesListOutput{}); err == nil {
		t.Fatal("unauthenticated metadata exposed")
	}
}
