package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	stored "github.com/viant/datly-studio/studio/namespaces/store_read"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	mcpconfig "github.com/viant/mcp/client/config"
	"github.com/viant/scy/auth/jwt"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
	"net/url"
	"reflect"
	"sort"
	"strings"
)

func fail(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message)}
}
func directory(ctx context.Context, session xhandler.Session, claims *jwt.Claims, auth *studioauth.Output, id, query string, limit, offset int) ([]*stored.StoredNamespace, error) {
	if claims == nil || claims.Subject == "" || auth == nil || auth.Auth == nil || auth.Auth.Subject != claims.Subject {
		return nil, fail(403, "verified Studio principal is required")
	}
	if id != "" && namespaceaccess.ValidateResourceOwnership(id, "") != nil {
		return nil, fail(400, "canonical namespace ID is required")
	}
	if session == nil || session.Binder() == nil {
		return nil, fmt.Errorf("namespace catalog session is unavailable")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return nil, err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return nil, fmt.Errorf("namespace catalog component invoker is unavailable")
	}
	var headers struct {
		Authorization string `bind:"kind=header,in=Authorization"`
	}
	if err = session.Binder().Bind(ctx, &headers); err != nil {
		return nil, err
	}
	ctx = oauth.WithBearer(ctx, strings.TrimPrefix(headers.Authorization, "Bearer "))
	input := &stored.Input{}
	input.NamespaceId = id
	input.SetScoped(true)
	input.Has.NamespaceId = true
	input.SetSubject(claims.Subject)
	input.SetStatus("active")
	input.SetPageLimit(limit)
	input.SetPageOffset(offset)
	if query = strings.TrimSpace(query); query != "" {
		input.SetQuery("%" + strings.ToLower(query) + "%")
	}
	holder := reflect.TypeFor[stored.NamespaceComponent]()
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: "namespace"}, Route: spec.RouteRef{Method: "GET", Path: "/_studio/namespace-store/read"}}, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok || output == nil {
		return nil, fmt.Errorf("namespace catalog reader returned %T", value)
	}
	for _, row := range output.Namespaces {
		if row == nil || id != "" && row.NamespaceId != id {
			return nil, fmt.Errorf("namespace catalog reader returned mismatched identity")
		}
	}
	return output.Namespaces, nil
}
func visible(ctx context.Context, session xhandler.Session, claims *jwt.Claims, auth *studioauth.Output, id string) (*stored.StoredNamespace, error) {
	if id == "" {
		return nil, fail(400, "namespaceId is required")
	}
	rows, err := directory(ctx, session, claims, auth, id, "", 2, 0)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fail(404, "namespace not found or access denied")
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("ambiguous namespace identity")
	}
	return rows[0], nil
}
func namespace(row *stored.StoredNamespace) *Namespace {
	result := &Namespace{URI: "ns://" + row.NamespaceId, Name: row.Name}
	if row.Description != nil {
		result.Description = *row.Description
	}
	return result
}

// VirtualURI is an opaque resource reference, not a remote URL or a filesystem
// path. Namespace identity avoids collisions between identically named owners.
func VirtualURI(namespaceID, name string) string {
	return (&url.URL{Scheme: "mcp", Host: namespaceID, Path: "/" + name}).String()
}
func ParseNamespaceURI(value string) (string, error) {
	id, err := namespaceext.ParseURI(value)
	if err != nil {
		return "", fail(400, "canonical namespace URI is required")
	}
	return id, nil
}
func ParseNamespacePath(value string) (id, kind string, err error) {
	parts := strings.Split(value, "/")
	if len(parts) != 3 && len(parts) != 4 || parts[0] != "" || parts[1] != "namespaces" {
		return "", "", fail(400, "expected /namespaces/<id> or /namespaces/<id>/<kind>")
	}
	id = parts[2]
	if namespaceaccess.ValidateResourceOwnership(id, "") != nil {
		return "", "", fail(400, "canonical namespace ID is required")
	}
	if len(parts) == 4 {
		kind = parts[3]
		if kind == "" {
			return "", "", fail(400, "resource kind is required")
		}
		for _, r := range kind {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
				return "", "", fail(400, "invalid resource kind")
			}
		}
	}
	return id, kind, nil
}
func ParseVirtualURI(value string) (id, name string, err error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "mcp" || parsed.User != nil || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawPath != "" {
		return "", "", fail(400, "invalid virtual MCP resource URI")
	}
	parts := strings.Split(parsed.Path, "/")
	if namespaceaccess.ValidateResourceOwnership(parsed.Host, "") != nil || len(parts) != 2 || parts[0] != "" || parts[1] == "" || parts[1] == "." || parts[1] == ".." || VirtualURI(parsed.Host, parts[1]) != value {
		return "", "", fail(400, "invalid virtual MCP resource URI")
	}
	return parsed.Host, parts[1], nil
}

func runList(ctx context.Context, session xhandler.Session, input *ListInput, output *ListOutput) error {
	if input == nil || output == nil {
		return fail(400, "namespace catalog input/output required")
	}
	entries := []Namespace{}
	adminNamespace, _, adminErr := management(input.Jwt, input.Auth)
	if adminErr != nil {
		return adminErr
	}
	if adminNamespace != nil {
		entries = append(entries, *adminNamespace)
	}
	seen := map[string]bool{}
	for offset := 0; ; offset += 500 {
		rows, err := directory(ctx, session, input.Jwt, input.Auth, "", "", 500, offset)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if !seen[row.NamespaceId] {
				entries = append(entries, *namespace(row))
				seen[row.NamespaceId] = true
			}
		}
		if len(rows) < 500 {
			break
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].URI < entries[j].URI })
	var err error
	output.Response = &namespaceext.ListResult{}
	output.Response.Namespaces, output.Response.NextCursor, err = namespaceext.Paginate(entries, input.Request.Cursor, 32)
	if errors.Is(err, namespaceext.ErrCursor) {
		return fail(400, err.Error())
	}
	return err
}
func runGet(ctx context.Context, session xhandler.Session, input *GetInput, output *GetOutput) error {
	if input == nil || output == nil {
		return fail(400, "namespace catalog input/output required")
	}
	id, err := ParseNamespaceURI(input.Request.URI)
	if err != nil {
		return err
	}
	adminNamespace, definitions, err := management(input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	if adminNamespace != nil && adminNamespace.URI == input.Request.URI {
		output.Response = &namespaceext.GetResult{Namespace: adminNamespace}
		output.Response.Resources, output.Response.NextCursor, err = namespaceext.Paginate(managementReferences(definitions, input.Request.Kind), input.Request.Cursor, 32)
		if errors.Is(err, namespaceext.ErrCursor) {
			return fail(400, err.Error())
		}
		return err
	}
	row, err := visible(ctx, session, input.Jwt, input.Auth, id)
	if err != nil {
		return err
	}
	output.Response = &namespaceext.GetResult{Namespace: namespace(row)}
	entries := []Reference{}
	if row.McpEnabled && (input.Request.Kind == "" || input.Request.Kind == "mcp") {
		_, connectionErr := connection(ctx, row)
		if connectionErr != nil && !errors.Is(connectionErr, runtimeadmin.ErrNamespaceUnavailable) {
			return connectionErr
		}
		description := ""
		if row.Description != nil {
			description = *row.Description
		}
		if connectionErr == nil {
			entries = append(entries, Reference{Kind: "mcp", URI: VirtualURI(row.NamespaceId, row.Name), Name: row.Name, Description: description, MimeType: "application/json"})
		}
	}
	output.Response.Resources, output.Response.NextCursor, err = namespaceext.Paginate(entries, input.Request.Cursor, 32)
	if errors.Is(err, namespaceext.ErrCursor) {
		return fail(400, err.Error())
	}
	return err
}
func runResource(ctx context.Context, session xhandler.Session, input *ResourceInput, output *ResourceOutput) error {
	if input == nil || output == nil {
		return fail(400, "resource input/output required")
	}
	_, definitions, err := management(input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	for _, definition := range definitions {
		if definition.URI == input.Request.URI {
			output.Response = append(json.RawMessage(nil), definition.JSON...)
			return nil
		}
	}
	id, name, err := ParseVirtualURI(input.Request.URI)
	if err != nil {
		return err
	}
	row, err := visible(ctx, session, input.Jwt, input.Auth, id)
	if err != nil {
		return err
	}
	if !row.McpEnabled || row.Name != name {
		return fail(404, "namespace MCP resource is disabled")
	}
	configuration, err := connection(ctx, row)
	if errors.Is(err, runtimeadmin.ErrNamespaceUnavailable) {
		return fail(404, "namespace MCP resource is unavailable")
	}
	if err != nil {
		return err
	}
	output.Response, err = json.Marshal(configuration)
	return err
}

// A persisted enable flag is not proof that a listener is running. Resolve the
// active generation's exported connection for both discovery and retrieval.
func connection(ctx context.Context, row *stored.StoredNamespace) (*mcpconfig.MCPClient, error) {
	client, err := runtimeadmin.Resolve(ctx)
	if err != nil {
		return nil, fail(503, "namespace runtime administration is not configured")
	}
	data, err := client.Connection(sdk.WithNamespaceSelection(ctx, row.NamespaceId))
	if errors.Is(err, runtimeadmin.ErrNamespaceUnavailable) {
		return nil, err
	}
	if err != nil {
		return nil, fail(503, "namespace MCP configuration is unavailable")
	}
	var result *mcpconfig.MCPClient
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if result == nil || result.ClientOptions == nil || result.Name != row.Name {
		return nil, fail(503, "namespace MCP configuration identity does not match")
	}
	return result, nil
}
