package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	stored "github.com/viant/datly-studio/studio/namespaces/store_read"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	identity "github.com/viant/forge/backend/resource"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type SkillListInput struct {
	Jwt     *jwt.Claims                    `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth    *studioauth.Output             `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Request schema.ListSkillsRequestParams `parameter:"Request,kind=body,in=,dataType=schema.ListSkillsRequestParams,required=false" anonymous:"true"`
}
type SkillGetInput struct {
	Jwt     *jwt.Claims                  `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth    *studioauth.Output           `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Request schema.GetSkillRequestParams `parameter:"Request,kind=body,in=,dataType=schema.GetSkillRequestParams,required=true" anonymous:"true"`
}
type SkillListOutput struct {
	Response *schema.ListSkillsResult `parameter:"Response,kind=output,in=body,dataType=*schema.ListSkillsResult" json:"-"`
}
type SkillGetOutput struct {
	Response *schema.GetSkillResult `parameter:"Response,kind=output,in=body,dataType=*schema.GetSkillResult" json:"-"`
}

func (*SkillListOutput) JSONWireType() reflect.Type {
	return reflect.TypeFor[schema.ListSkillsResult]()
}
func (*SkillGetOutput) JSONWireType() reflect.Type { return reflect.TypeFor[schema.GetSkillResult]() }
func (o *SkillListOutput) MarshalJSON() ([]byte, error) {
	if o == nil {
		return []byte("null"), nil
	}
	return json.Marshal(o.Response)
}
func (o *SkillGetOutput) MarshalJSON() ([]byte, error) {
	if o == nil {
		return []byte("null"), nil
	}
	return json.Marshal(o.Response)
}

type SkillListComponent struct {
	Contract xdatly.Component[SkillListInput, SkillListOutput] `component:"catalog_skills_list,path=/v1/studio/catalog/skills/list,method=POST,connector=studio,handler=NewSkillList" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.catalog.skills.list\",\"description\":\"List authorized published namespace skills\"}]" caseFormat:"lc"`
}
type SkillGetComponent struct {
	Contract xdatly.Component[SkillGetInput, SkillGetOutput] `component:"catalog_skills_get,path=/v1/studio/catalog/skills/get,method=POST,connector=studio,handler=NewSkillGet" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.catalog.skills.get\",\"description\":\"Retrieve an authorized published namespace skill\"}]" caseFormat:"lc"`
}

var SkillListDatly = new(SkillListComponent)
var SkillGetDatly = new(SkillGetComponent)
var SkillListDatlyLinkedType = reflect.TypeFor[SkillListComponent]()
var SkillGetDatlyLinkedType = reflect.TypeFor[SkillGetComponent]()

func (SkillListComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewSkillList" {
		return custom.Factory(NewSkillList)
	}
	return nil
}
func (SkillGetComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewSkillGet" {
		return custom.Factory(NewSkillGet)
	}
	return nil
}
func NewSkillList() xhandler.Contract[SkillListInput, SkillListOutput] {
	return &handler[SkillListInput, SkillListOutput]{run: runSkillList}
}
func NewSkillGet() xhandler.Contract[SkillGetInput, SkillGetOutput] {
	return &handler[SkillGetInput, SkillGetOutput]{run: runSkillGet}
}

func catalogRuntime(ctx context.Context) (*runtimeadmin.Client, error) {
	client, err := runtimeadmin.Resolve(ctx)
	if err != nil {
		return nil, fail(503, "namespace runtime administration is not configured")
	}
	return client, nil
}
func catalogBearer(ctx context.Context, session xhandler.Session) (string, error) {
	if session == nil || session.Binder() == nil {
		return "", fail(401, "verified caller credential is required")
	}
	var headers struct {
		Authorization string `bind:"kind=header,in=Authorization"`
	}
	if err := session.Binder().Bind(ctx, &headers); err != nil {
		return "", err
	}
	fields := strings.Fields(headers.Authorization)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", fail(401, "verified caller credential is required")
	}
	return fields[1], nil
}
func skillNamespace(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "skill" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawPath != "" || parsed.Path == "" || parsed.Path == "/" || parsed.String() != value {
		return "", fail(400, "namespace skill URI is required")
	}
	if namespaceaccess.ValidateResourceOwnership(parsed.Host, "") != nil && !(identity.ResourceURI{Kind: "skill", Namespace: parsed.Host, Name: "validate"}).Valid() {
		return "", fail(400, "namespace skill URI is required")
	}
	for _, part := range strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/") {
		if part == "" || part == "." || part == ".." {
			return "", fail(400, "invalid skill URI")
		}
	}
	return parsed.Host, nil
}
func runSkillList(ctx context.Context, session xhandler.Session, input *SkillListInput, output *SkillListOutput) error {
	if input == nil || output == nil {
		return fail(400, "skill input/output required")
	}
	entries := []schema.Skill{}
	seen := map[string]bool{}
	var client *runtimeadmin.Client
	var bearer string
	for offset := 0; ; offset += 500 {
		rows, err := directory(ctx, session, input.Jwt, input.Auth, "", "", 500, offset)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if !row.McpEnabled {
				continue
			}
			if client == nil {
				client, err = catalogRuntime(ctx)
				if err != nil {
					return err
				}
				bearer, err = catalogBearer(ctx, session)
				if err != nil {
					return err
				}
			}
			var cursor *string
			cursors := map[string]bool{}
			for {
				raw, err := client.ReadCatalog(sdk.WithNamespaceSelection(ctx, row.NamespaceId), bearer, "skills/list", "", cursor)
				var rpc *runtimeadmin.CatalogError
				if errors.Is(err, runtimeadmin.ErrNamespaceUnavailable) || errors.As(err, &rpc) && rpc.Code == -32601 {
					break
				}
				if err != nil {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					if errors.Is(err, context.DeadlineExceeded) {
						return errors.Join(fail(504, "Namespace skill catalog deadline expired."), err)
					}
					return errors.Join(fail(503, "Namespace skills could not be loaded."), fmt.Errorf("read namespace %s skill catalog: %w", row.NamespaceId, err))
				}
				var page schema.ListSkillsResult
				if err := json.Unmarshal(raw, &page); err != nil {
					return errors.Join(fail(503, "The namespace returned an invalid skill catalog."), err)
				}
				for _, skill := range page.Skills {
					id, err := skillNamespace(skill.Uri)
					if err != nil || (id != row.NamespaceId && id != row.Name) || seen[skill.Uri] {
						return fail(503, "namespace skill identity does not match")
					}
					seen[skill.Uri] = true
					entries = append(entries, skill)
					if len(entries) > 4096 {
						return fail(503, "skill catalog exceeds inventory limit")
					}
				}
				if page.NextCursor == nil {
					break
				}
				if cursors[*page.NextCursor] || len(cursors) >= 128 {
					return fail(503, "invalid namespace skill pagination")
				}
				cursors[*page.NextCursor] = true
				cursor = page.NextCursor
			}
		}
		if len(rows) < 500 {
			break
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Uri < entries[j].Uri })
	page, next, err := namespaceext.Paginate(entries, input.Request.Cursor, 32)
	if err != nil {
		return fail(400, "invalid or stale skill catalog cursor")
	}
	zero := 0
	output.Response = &schema.ListSkillsResult{Skills: page, NextCursor: next, CacheScope: schema.CacheableResultCacheScopePrivate, TtlMs: &zero}
	return nil
}
func runSkillGet(ctx context.Context, session xhandler.Session, input *SkillGetInput, output *SkillGetOutput) error {
	if input == nil || output == nil {
		return fail(400, "skill input/output required")
	}
	id, err := skillNamespace(input.Request.Uri)
	if err != nil {
		return err
	}
	var row *stored.StoredNamespace
	if namespaceaccess.ValidateResourceOwnership(id, "") == nil {
		row, err = visible(ctx, session, input.Jwt, input.Auth, id)
	} else {
		// Resolve a named URI through the same scoped graph as namespace/tool
		// selection. A URI's authority is a selector, never an access grant.
		var rows []*stored.StoredNamespace
		rows, err = directoryPage(ctx, session, input.Jwt, input.Auth, "", id, "", false, 2, 0)
		if err == nil {
			switch {
			case len(rows) == 0:
				err = fail(404, "namespace not found or access denied")
			case len(rows) != 1 || rows[0].Name != id:
				err = fail(409, "namespace name is ambiguous")
			default:
				row = rows[0]
			}
		}
	}
	if err != nil {
		return err
	}
	if !row.McpEnabled {
		return fail(404, "namespace skill is unavailable")
	}
	client, err := catalogRuntime(ctx)
	if err != nil {
		return err
	}
	bearer, err := catalogBearer(ctx, session)
	if err != nil {
		return err
	}
	raw, err := client.ReadCatalog(sdk.WithNamespaceSelection(ctx, row.NamespaceId), bearer, "skills/get", input.Request.Uri, nil)
	if err != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.Join(fail(504, "Namespace skill retrieval deadline expired."), err)
	}
	var rpc *runtimeadmin.CatalogError
	if errors.Is(err, runtimeadmin.ErrNamespaceUnavailable) || errors.As(err, &rpc) {
		return errors.Join(fail(404, "skill not found or access denied"), err)
	}
	if err != nil {
		return errors.Join(fail(503, "namespace skill is unavailable"), err)
	}
	var result schema.GetSkillResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return errors.Join(fail(503, "The namespace returned an invalid skill."), err)
	}
	if result.Skill.Uri != input.Request.Uri {
		return fail(503, "namespace skill identity does not match")
	}
	zero := 0
	result.CacheScope = schema.CacheableResultCacheScopePrivate
	result.TtlMs = &zero
	output.Response = &result
	return nil
}
