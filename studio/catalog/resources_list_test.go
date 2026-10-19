package catalog

import (
	"context"
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
	seen := map[string]bool{}
	for page := 0; page < 3; page++ {
		out := &ResourcesListOutput{}
		if err := runResourcesList(context.Background(), nil, input, out); err != nil {
			t.Fatal(err)
		}
		if len(out.Response.Resources) != 1 {
			t.Fatalf("page=%+v", out.Response)
		}
		uri := out.Response.Resources[0].Uri
		if seen[uri] {
			t.Fatalf("duplicate %s", uri)
		}
		seen[uri] = true
		input.Request.Cursor = out.Response.NextCursor
	}
	if input.Request.Cursor != nil || len(seen) != 3 {
		t.Fatalf("listing never completed: %+v", input.Request.Cursor)
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
