package catalog

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/viant/datly-studio/internal/runtimeadmin"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	stored "github.com/viant/datly-studio/studio/namespaces/store_read"
	"github.com/viant/datly/exec"
	"github.com/viant/mcp"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	mcpconfig "github.com/viant/mcp/client/config"
	"github.com/viant/scy/auth/jwt"
)

type logicalAliasDirectory struct{ rows []*stored.StoredNamespace }

func (d logicalAliasDirectory) InvokeComponent(_ context.Context, request exec.ComponentRequest) (any, error) {
	input, ok := request.Input.(*stored.Input)
	if !ok || !input.Scoped || input.Subject == "" || request.Target.Route.Path != "/_studio/namespace-store/read" {
		return nil, namespaceext.ErrCursor
	}
	var selected []*stored.StoredNamespace
	for _, row := range d.rows {
		if row == nil || input.Subject != "publisher" || input.NamespaceId != "" && input.NamespaceId != row.NamespaceId || input.Name != "" && input.Name != row.Name || input.McpOnly && !row.McpEnabled {
			continue
		}
		selected = append(selected, row)
	}
	return &stored.Output{Namespaces: selected}, nil
}

func TestLogicalMCPAliasUsesVisibleActiveNamespaceAndKeepsOpaqueURI(t *testing.T) {
	const id = "f560c53611b11f84e5b5b4a4ac2b82117875bc9430bfbdd31c65c6d9f813883b"
	claims := &jwt.Claims{}
	claims.Subject = "publisher"
	auth := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "publisher"}}
	connection := &mcpconfig.MCPClient{ClientOptions: &mcp.ClientOptions{Name: "ar", Transport: mcp.ClientTransport{Type: "streamable", ClientTransportHTTP: mcp.ClientTransportHTTP{URL: "https://producer.example/api/namespaces/ar/mcp"}}}}
	encoded, err := json.Marshal(connection)
	if err != nil {
		t.Fatal(err)
	}
	ctx := runtimeadmin.WithClient(context.Background(), runtimeadmin.NewInProcess(nil, nil, func(context.Context) (json.RawMessage, error) { return encoded, nil }))
	row := &stored.StoredNamespace{NamespaceId: id, Name: "ar", McpEnabled: true, Status: "active"}
	session := releaseTestSession{binder: releaseTestBinder{invoker: logicalAliasDirectory{rows: []*stored.StoredNamespace{row}}}}
	listed := &GetOutput{}
	if err = runGet(ctx, session, &GetInput{Jwt: claims, Auth: auth, Request: namespaceext.GetRequest{URI: "ns://" + id, Kind: "mcp"}}, listed); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{VirtualURI(id, "ar"): false, LogicalURI("ar"): false}
	for _, item := range listed.Response.Resources {
		if _, ok := want[item.URI]; ok {
			want[item.URI] = true
		}
	}
	for uri, found := range want {
		if !found {
			t.Fatalf("MCP URI %q missing from producer index", uri)
		}
	}
	for uri := range want {
		read := &ResourceOutput{}
		if err = runResource(ctx, session, &ResourceInput{Jwt: claims, Auth: auth, Request: namespaceext.ResourceGetRequest{URI: uri}}, read); err != nil {
			t.Fatalf("read %s: %v", uri, err)
		}
		var got mcpconfig.MCPClient
		if json.Unmarshal(read.Response, &got) != nil || got.ClientOptions == nil || got.Name != "ar" || got.Transport.URL != connection.Transport.URL {
			t.Fatalf("alias %s changed producer connection identity", uri)
		}
	}
	visitor := &jwt.Claims{}
	visitor.Subject = "visitor"
	visitorAuth := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "visitor"}}
	if err = runResource(ctx, session, &ResourceInput{Jwt: visitor, Auth: visitorAuth, Request: namespaceext.ResourceGetRequest{URI: LogicalURI("ar")}}, &ResourceOutput{}); err == nil {
		t.Fatal("other account read logical alias")
	}
	ambiguous := &stored.StoredNamespace{NamespaceId: strings.Repeat("a", 64), Name: "ar", McpEnabled: true, Status: "active"}
	session.binder = releaseTestBinder{invoker: logicalAliasDirectory{rows: []*stored.StoredNamespace{row, ambiguous}}}
	if err = runResource(ctx, session, &ResourceInput{Jwt: claims, Auth: auth, Request: namespaceext.ResourceGetRequest{URI: LogicalURI("ar")}}, &ResourceOutput{}); err == nil {
		t.Fatal("ambiguous visible alias selected one account")
	}
	row.McpEnabled = false
	session.binder = releaseTestBinder{invoker: logicalAliasDirectory{rows: []*stored.StoredNamespace{row}}}
	if err = runGet(ctx, session, &GetInput{Jwt: claims, Auth: auth, Request: namespaceext.GetRequest{URI: "ns://" + id, Kind: "mcp"}}, listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Response.Resources) != 0 {
		t.Fatal("disabled MCP namespace advertised alias")
	}
	if err = runResource(ctx, session, &ResourceInput{Jwt: claims, Auth: auth, Request: namespaceext.ResourceGetRequest{URI: LogicalURI("ar")}}, &ResourceOutput{}); err == nil {
		t.Fatal("disabled MCP namespace returned alias")
	}
}
