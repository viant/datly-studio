package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	stored "github.com/viant/datly-studio/studio/namespaces/store_read"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly/response"
)

func TestSkillCatalogPreservesOpaqueAndNamedNamespaceURIs(t *testing.T) {
	id := strings.Repeat("a", 64)
	canonical := "skill://" + id + "/guide/2/guide/SKILL.md"
	named := "skill://example/guide/revisions/2/fingerprint/guide/SKILL.md"
	rows := []*stored.StoredNamespace{{NamespaceId: id, Name: "example", Status: "active", McpEnabled: true}}
	session := releaseTestSession{binder: releaseTestBinder{invoker: logicalAliasDirectory{rows: rows}}}
	claims := &jwt.Claims{}
	claims.Subject = "publisher"
	auth := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "publisher"}}
	var calls atomic.Int64
	var foreign atomic.Bool
	skill := func(uri string) schema.Skill {
		return schema.Skill{Uri: uri, Frontmatter: map[string]any{"name": "guide", "description": "Synthetic guide"}, Resources: schema.SkillResources{Files: []schema.SkillResource{{Uri: uri, Digest: "sha256:" + strings.Repeat("0", 64), Size: 1}}}}
	}
	producer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer verified-test" || r.Header.Get("X-Studio-Runtime-Token") != "" {
			t.Error("catalog did not preserve caller authority")
		}
		var request struct {
			Method string `json:"method"`
			Params struct {
				URI string `json:"uri"`
			} `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		var result any
		switch request.Method {
		case "skills/list":
			alias := named
			if foreign.Load() {
				alias = "skill://unrelated/guide/SKILL.md"
			}
			result = schema.ListSkillsResult{Skills: []schema.Skill{skill(canonical), skill(alias)}}
		case "skills/get":
			if request.Params.URI != canonical && request.Params.URI != named {
				t.Error("skill URI rewritten")
			}
			result = schema.GetSkillResult{Skill: skill(request.Params.URI)}
		default:
			t.Error("unexpected producer operation")
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result}))
	}))
	t.Cleanup(producer.Close)
	client := runtimeadmin.NewInProcess(nil, func(ctx context.Context) (*sdk.RuntimeHost, error) {
		selected, ok := sdk.NamespaceSelectionFromContext(ctx)
		require.True(t, ok)
		require.Equal(t, id, selected)
		return &sdk.RuntimeHost{NamespaceID: id, Status: "ready", MCPURL: producer.URL}, nil
	}, nil)
	ctx := runtimeadmin.WithClient(t.Context(), client)
	listed := new(SkillListOutput)
	require.NoError(t, runSkillList(ctx, session, &SkillListInput{Jwt: claims, Auth: auth}, listed))
	require.Len(t, listed.Response.Skills, 2)
	require.Equal(t, int64(1), calls.Load(), "one producer list, without per-skill lookup")
	for _, uri := range []string{canonical, named} {
		out := new(SkillGetOutput)
		require.NoError(t, runSkillGet(ctx, session, &SkillGetInput{Jwt: claims, Auth: auth, Request: schema.GetSkillRequestParams{Uri: uri}}, out))
		require.Equal(t, uri, out.Response.Skill.Uri)
	}
	before := calls.Load()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, runSkillList(canceled, session, &SkillListInput{Jwt: claims, Auth: auth}, new(SkillListOutput)), context.Canceled)
	require.ErrorIs(t, runSkillGet(canceled, session, &SkillGetInput{Jwt: claims, Auth: auth, Request: schema.GetSkillRequestParams{Uri: named}}, new(SkillGetOutput)), context.Canceled)
	visitor := &jwt.Claims{}
	visitor.Subject = "visitor"
	visitorAuth := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "visitor"}}
	err := runSkillGet(ctx, session, &SkillGetInput{Jwt: visitor, Auth: visitorAuth, Request: schema.GetSkillRequestParams{Uri: named}}, new(SkillGetOutput))
	require.Equal(t, 404, response.ErrorStatusCode(err, 500))
	ambiguous := append(append([]*stored.StoredNamespace{}, rows...), &stored.StoredNamespace{NamespaceId: strings.Repeat("b", 64), Name: "example", Status: "active", McpEnabled: true})
	ambiguousSession := releaseTestSession{binder: releaseTestBinder{invoker: logicalAliasDirectory{rows: ambiguous}}}
	err = runSkillGet(ctx, ambiguousSession, &SkillGetInput{Jwt: claims, Auth: auth, Request: schema.GetSkillRequestParams{Uri: named}}, new(SkillGetOutput))
	require.Equal(t, 409, response.ErrorStatusCode(err, 500))
	require.Equal(t, before, calls.Load(), "denied/ambiguous lookup reached producer")
	foreign.Store(true)
	err = runSkillList(ctx, session, &SkillListInput{Jwt: claims, Auth: auth}, new(SkillListOutput))
	require.Equal(t, 503, response.ErrorStatusCode(err, 500))
	body, ok := response.ErrorBody(err)
	require.True(t, ok)
	require.NotNil(t, body, "identity failure must have an actionable public body")
}

func TestSkillNamespaceRejectsNoncanonicalPaths(t *testing.T) {
	for _, uri := range []string{
		"skill://user@example/guide/SKILL.md", "skill://example/guide/SKILL.md?x=1",
		"skill://example/guide/SKILL.md#x", "skill://example/guide/../SKILL.md",
		"skill://example/guide/%2e%2e/SKILL.md", "skill://example/%67uide/SKILL.md",
		"skill://example/guide//SKILL.md", "skill://example:80/guide/SKILL.md",
	} {
		_, err := skillNamespace(uri)
		require.Error(t, err, uri)
	}
}
