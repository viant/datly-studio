package catalog

import (
	"context"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/forge/backend/types"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	"github.com/viant/scy/auth/jwt"
	"testing"
)

func TestManagementWindowReadValidatesPresentationTarget(t *testing.T) {
	t.Setenv("STUDIO_MANAGEMENT_NAMESPACE", "management")
	t.Setenv("STUDIO_MANAGEMENT_PROVIDER_ID", "provider")
	t.Setenv("STUDIO_MANAGEMENT_MCP_SERVICE", "service")
	t.Setenv("STUDIO_MANAGEMENT_PUBLIC_URL", "https://studio.example.com")
	claims := &jwt.Claims{}
	claims.Subject = "alice"
	in := &ResourceInput{Jwt: claims, Auth: &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "alice"}}, Request: namespaceext.ResourceGetRequest{URI: "window://management/connectors"}}
	original := &ResourceOutput{}
	if err := runResource(context.Background(), nil, in, original); err != nil {
		t.Fatal(err)
	}
	in.Target = &types.WindowTarget{Platform: "web", FormFactor: "desktop", Surface: "browser"}
	out := &ResourceOutput{}
	if err := runResource(context.Background(), nil, in, out); err != nil {
		t.Fatal(err)
	}
	if string(out.Response) != string(original.Response) {
		t.Fatal("target validation changed immutable definition")
	}
	in.Target.FormFactor = "unsupported"
	if err := runResource(context.Background(), nil, in, &ResourceOutput{}); err == nil {
		t.Fatal("unsupported target accepted")
	}
	in.Target = &types.WindowTarget{SelectionToken: "caller-proof"}
	if err := runResource(context.Background(), nil, in, &ResourceOutput{}); err == nil {
		t.Fatal("authority target accepted")
	}
}
